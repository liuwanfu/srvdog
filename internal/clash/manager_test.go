package clash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusReadsCurrentTokenAndPublishedPaths(t *testing.T) {
	root := t.TempDir()
	tokenFile := filepath.Join(root, "token")
	siteDir := filepath.Join(root, "site-wg")
	token := "abc123"
	tokenDir := filepath.Join(siteDir, token)

	if err := os.MkdirAll(tokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenFile, []byte(token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "config.yaml"), []byte("mode: rule\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "geoip.dat"), []byte("geoip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "geosite.dat"), []byte("geosite"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(Config{
		DataDir:       filepath.Join(root, "data"),
		TokenFile:     tokenFile,
		SiteDir:       siteDir,
		PublicBaseURL: "https://example.com/wg",
	})

	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}

	if status.Token != token {
		t.Fatalf("status.Token = %q, want %q", status.Token, token)
	}
	if status.ConfigPath != filepath.Join(tokenDir, "config.yaml") {
		t.Fatalf("status.ConfigPath = %q", status.ConfigPath)
	}
	if status.SubscriptionURL != "https://example.com/wg/"+token+"/config.yaml" {
		t.Fatalf("status.SubscriptionURL = %q", status.SubscriptionURL)
	}
	if status.GeoIPURL != "https://example.com/wg/"+token+"/geoip.dat" {
		t.Fatalf("status.GeoIPURL = %q", status.GeoIPURL)
	}
	if status.GeoSiteURL != "https://example.com/wg/"+token+"/geosite.dat" {
		t.Fatalf("status.GeoSiteURL = %q", status.GeoSiteURL)
	}
}

func TestConfigPrefersDraftContentOverPublishedConfig(t *testing.T) {
	root := t.TempDir()
	manager := newTestManager(t, root, testPaths{
		token:           "abc123",
		publishedConfig: "mode: rule\nrules:\n  - MATCH,PROXY\n",
	})

	if err := manager.SaveConfigDraft("mode: global\n"); err != nil {
		t.Fatal(err)
	}

	doc, err := manager.GetConfig()
	if err != nil {
		t.Fatal(err)
	}

	if !doc.HasDraft {
		t.Fatal("expected doc.HasDraft = true")
	}
	if doc.Source != "draft" {
		t.Fatalf("doc.Source = %q, want draft", doc.Source)
	}
	if doc.Content != "mode: global\n" {
		t.Fatalf("doc.Content = %q", doc.Content)
	}
}

func TestPublishScriptUpdatesScriptFileAndPublishedConfig(t *testing.T) {
	root := t.TempDir()
	manager := newTestManager(t, root, testPaths{
		token:           "abc123",
		publishedConfig: "mode: rule\nrules:\n  - MATCH,PROXY\n",
		script:          "",
		geoIP:           "geoip",
		geoSite:         "geosite",
	})
	runner := &stubRunner{}
	manager.cfg.Runner = runner
	manager.cfg.MihomoImage = "metacubex/mihomo:Alpha"

	scriptDraft := "script:\n  shortcuts:\n    google: host == \"gemini.google.com\"\n"
	if err := manager.SaveScriptDraft(scriptDraft); err != nil {
		t.Fatal(err)
	}

	if err := manager.PublishScript(context.Background()); err != nil {
		t.Fatal(err)
	}

	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	scriptBytes, err := os.ReadFile(status.ScriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(scriptBytes) != scriptDraft {
		t.Fatalf("published script = %q", string(scriptBytes))
	}

	configBytes, err := os.ReadFile(status.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	configText := string(configBytes)
	if !strings.Contains(configText, managedScriptStartMarker) {
		t.Fatalf("published config missing managed script block: %s", configText)
	}
	if !strings.Contains(configText, "shortcuts:") {
		t.Fatalf("published config missing script content: %s", configText)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "docker" {
		t.Fatalf("runner calls = %#v", runner.calls)
	}
}

func TestPublishConfigUsesDraftAndPreservesPublishedScript(t *testing.T) {
	root := t.TempDir()
	manager := newTestManager(t, root, testPaths{
		token:           "abc123",
		publishedConfig: "mode: rule\nrules:\n  - MATCH,PROXY\n",
		script:          "script:\n  shortcuts:\n    cn: host == \"qq.com\"\n",
		geoIP:           "geoip",
		geoSite:         "geosite",
	})
	runner := &stubRunner{}
	manager.cfg.Runner = runner

	if err := manager.SaveConfigDraft("mode: global\nrules:\n  - MATCH,DIRECT\n"); err != nil {
		t.Fatal(err)
	}

	if err := manager.PublishConfig(context.Background()); err != nil {
		t.Fatal(err)
	}

	status, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	configBytes, err := os.ReadFile(status.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	configText := string(configBytes)
	if !strings.Contains(configText, "mode: global") {
		t.Fatalf("published config = %q", configText)
	}
	if !strings.Contains(configText, "shortcuts:") {
		t.Fatalf("published config missing script block = %q", configText)
	}
	if _, err := os.Stat(manager.configDraftPath()); !os.IsNotExist(err) {
		t.Fatalf("expected config draft removed, got err=%v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "docker" {
		t.Fatalf("runner calls = %#v", runner.calls)
	}
}

func TestRotateTokenCopiesCurrentPublishedFilesAndRemovesOldDirectory(t *testing.T) {
	root := t.TempDir()
	manager := newTestManager(t, root, testPaths{
		token:           "oldtoken",
		publishedConfig: "mode: rule\nrules:\n  - MATCH,PROXY\n",
		script:          "script:\n  shortcuts:\n    cn: host == \"qq.com\"\n",
		geoIP:           "geoip",
		geoSite:         "geosite",
	})
	manager.cfg.TokenGenerator = func() (string, error) {
		return "newtoken", nil
	}

	status, err := manager.RotateToken()
	if err != nil {
		t.Fatal(err)
	}

	if status.Token != "newtoken" {
		t.Fatalf("status.Token = %q, want newtoken", status.Token)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(status.ConfigPath), "config.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "site-wg", "oldtoken")); !os.IsNotExist(err) {
		t.Fatalf("expected old token dir removed, got err=%v", err)
	}
	tokenBytes, err := os.ReadFile(filepath.Join(root, "token"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(tokenBytes)) != "newtoken" {
		t.Fatalf("token file = %q", string(tokenBytes))
	}
}

func TestUpdateGeodataAndReadLogs(t *testing.T) {
	root := t.TempDir()
	manager := newTestManager(t, root, testPaths{
		token:           "abc123",
		publishedConfig: "mode: rule\nrules:\n  - MATCH,PROXY\n",
		script:          "",
		geoIP:           "geoip",
		geoSite:         "geosite",
	})
	runner := &stubRunner{}
	manager.cfg.Runner = runner
	manager.cfg.GeodataUpdateScript = "/usr/local/bin/update-mihomo-geodata.sh"
	manager.cfg.GeodataLogPath = filepath.Join(root, "update-mihomo-geodata.log")

	if err := os.WriteFile(manager.cfg.GeodataLogPath, []byte("geodata line 1\ngeodata line 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := manager.appendOperation("publish config success"); err != nil {
		t.Fatal(err)
	}

	if err := manager.UpdateGeodata(context.Background()); err != nil {
		t.Fatal(err)
	}

	logs, err := manager.ReadLogs(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "/usr/local/bin/update-mihomo-geodata.sh" {
		t.Fatalf("runner calls = %#v", runner.calls)
	}
	if len(logs.Operations) == 0 || !strings.Contains(logs.Operations[0], "publish config success") {
		t.Fatalf("operations logs = %#v", logs.Operations)
	}
	if len(logs.Geodata) != 2 {
		t.Fatalf("geodata logs = %#v", logs.Geodata)
	}
}

type testPaths struct {
	token           string
	publishedConfig string
	script          string
	geoIP           string
	geoSite         string
}

func newTestManager(t *testing.T, root string, paths testPaths) *Manager {
	t.Helper()

	tokenFile := filepath.Join(root, "token")
	siteDir := filepath.Join(root, "site-wg")
	tokenDir := filepath.Join(siteDir, paths.token)

	if err := os.MkdirAll(tokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenFile, []byte(paths.token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if paths.publishedConfig != "" {
		if err := os.WriteFile(filepath.Join(tokenDir, "config.yaml"), []byte(paths.publishedConfig), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "script.yaml"), []byte(paths.script), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "geoip.dat"), []byte(paths.geoIP), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "geosite.dat"), []byte(paths.geoSite), 0o644); err != nil {
		t.Fatal(err)
	}

	return NewManager(Config{
		DataDir:       filepath.Join(root, "data"),
		TokenFile:     tokenFile,
		SiteDir:       siteDir,
		PublicBaseURL: "https://example.com/wg",
	})
}

type stubRunner struct {
	calls []runnerCall
}

type runnerCall struct {
	name string
	args []string
}

func (s *stubRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	return []byte("ok"), nil
}
