package clash

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	managedScriptStartMarker = "# >>> srvdog managed script start >>>"
	managedScriptEndMarker   = "# <<< srvdog managed script end <<<"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type Config struct {
	DataDir             string
	TokenFile           string
	SiteDir             string
	PublicBaseURL       string
	MihomoImage         string
	GeodataUpdateScript string
	GeodataLogPath      string
	Runner              Runner
	Now                 func() time.Time
	TokenGenerator      func() (string, error)
}

type Status struct {
	Token           string `json:"token"`
	ConfigPath      string `json:"config_path"`
	ScriptPath      string `json:"script_path"`
	GeoIPPath       string `json:"geoip_path"`
	GeoSitePath     string `json:"geosite_path"`
	SubscriptionURL string `json:"subscription_url"`
	GeoIPURL        string `json:"geoip_url"`
	GeoSiteURL      string `json:"geosite_url"`
}

type Document struct {
	Content   string `json:"content"`
	Path      string `json:"path"`
	DraftPath string `json:"draft_path"`
	Source    string `json:"source"`
	HasDraft  bool   `json:"has_draft"`
}

type Logs struct {
	Operations []string `json:"operations"`
	Geodata    []string `json:"geodata"`
}

type Manager struct {
	cfg Config
}

func NewManager(cfg Config) *Manager {
	if cfg.MihomoImage == "" {
		cfg.MihomoImage = "metacubex/mihomo:Alpha"
	}
	if cfg.Runner == nil {
		cfg.Runner = osRunner{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.TokenGenerator == nil {
		cfg.TokenGenerator = randomToken
	}
	return &Manager{cfg: cfg}
}

func (m *Manager) Status() (Status, error) {
	tokenBytes, err := os.ReadFile(m.cfg.TokenFile)
	if err != nil {
		return Status{}, err
	}

	token := strings.TrimSpace(string(tokenBytes))
	tokenDir := filepath.Join(m.cfg.SiteDir, token)
	baseURL := strings.TrimRight(m.cfg.PublicBaseURL, "/")

	return Status{
		Token:           token,
		ConfigPath:      filepath.Join(tokenDir, "config.yaml"),
		ScriptPath:      filepath.Join(tokenDir, "script.yaml"),
		GeoIPPath:       filepath.Join(tokenDir, "geoip.dat"),
		GeoSitePath:     filepath.Join(tokenDir, "geosite.dat"),
		SubscriptionURL: baseURL + "/" + token + "/config.yaml",
		GeoIPURL:        baseURL + "/" + token + "/geoip.dat",
		GeoSiteURL:      baseURL + "/" + token + "/geosite.dat",
	}, nil
}

func (m *Manager) GetConfig() (Document, error) {
	status, err := m.Status()
	if err != nil {
		return Document{}, err
	}
	if draft, err := os.ReadFile(m.configDraftPath()); err == nil {
		return Document{
			Content:   string(draft),
			Path:      status.ConfigPath,
			DraftPath: m.configDraftPath(),
			Source:    "draft",
			HasDraft:  true,
		}, nil
	}
	content, err := os.ReadFile(status.ConfigPath)
	if err != nil {
		return Document{}, err
	}
	return Document{
		Content:   stripManagedScriptBlock(string(content)),
		Path:      status.ConfigPath,
		DraftPath: m.configDraftPath(),
		Source:    "published",
		HasDraft:  false,
	}, nil
}

func (m *Manager) SaveConfigDraft(content string) error {
	return writeFileAtomic(m.configDraftPath(), []byte(content), 0o644)
}

func (m *Manager) GetScript() (Document, error) {
	status, err := m.Status()
	if err != nil {
		return Document{}, err
	}
	if draft, err := os.ReadFile(m.scriptDraftPath()); err == nil {
		return Document{
			Content:   string(draft),
			Path:      status.ScriptPath,
			DraftPath: m.scriptDraftPath(),
			Source:    "draft",
			HasDraft:  true,
		}, nil
	}
	content, err := os.ReadFile(status.ScriptPath)
	if err != nil && !os.IsNotExist(err) {
		return Document{}, err
	}
	return Document{
		Content:   string(content),
		Path:      status.ScriptPath,
		DraftPath: m.scriptDraftPath(),
		Source:    "published",
		HasDraft:  false,
	}, nil
}

func (m *Manager) SaveScriptDraft(content string) error {
	return writeFileAtomic(m.scriptDraftPath(), []byte(content), 0o644)
}

func (m *Manager) PublishScript(ctx context.Context) error {
	scriptDoc, err := m.GetScript()
	if err != nil {
		return err
	}
	configDoc, err := m.GetConfig()
	if err != nil {
		return err
	}
	merged := mergeManagedScriptBlock(configDoc.Content, scriptDoc.Content)
	status, err := m.Status()
	if err != nil {
		return err
	}
	if err := m.validateMergedConfig(ctx, merged, status); err != nil {
		return err
	}
	if err := writeFileAtomic(status.ScriptPath, []byte(scriptDoc.Content), 0o644); err != nil {
		return err
	}
	if err := writeFileAtomic(status.ConfigPath, []byte(merged), 0o644); err != nil {
		return err
	}
	if scriptDoc.HasDraft {
		_ = os.Remove(m.scriptDraftPath())
	}
	return m.appendOperation("publish script success")
}

func (m *Manager) ValidateScript(ctx context.Context) error {
	scriptDoc, err := m.GetScript()
	if err != nil {
		return err
	}
	configDoc, err := m.GetConfig()
	if err != nil {
		return err
	}
	status, err := m.Status()
	if err != nil {
		return err
	}
	return m.validateMergedConfig(ctx, mergeManagedScriptBlock(configDoc.Content, scriptDoc.Content), status)
}

func (m *Manager) PublishConfig(ctx context.Context) error {
	configDoc, err := m.GetConfig()
	if err != nil {
		return err
	}
	scriptDoc, err := m.GetScript()
	if err != nil {
		return err
	}
	merged := mergeManagedScriptBlock(configDoc.Content, scriptDoc.Content)
	status, err := m.Status()
	if err != nil {
		return err
	}
	if err := m.validateMergedConfig(ctx, merged, status); err != nil {
		return err
	}
	if err := writeFileAtomic(status.ConfigPath, []byte(merged), 0o644); err != nil {
		return err
	}
	if configDoc.HasDraft {
		_ = os.Remove(m.configDraftPath())
	}
	return m.appendOperation("publish config success")
}

func (m *Manager) ValidateConfig(ctx context.Context) error {
	configDoc, err := m.GetConfig()
	if err != nil {
		return err
	}
	scriptDoc, err := m.GetScript()
	if err != nil {
		return err
	}
	status, err := m.Status()
	if err != nil {
		return err
	}
	return m.validateMergedConfig(ctx, mergeManagedScriptBlock(configDoc.Content, scriptDoc.Content), status)
}

func (m *Manager) RotateToken() (Status, error) {
	oldStatus, err := m.Status()
	if err != nil {
		return Status{}, err
	}
	newToken, err := m.cfg.TokenGenerator()
	if err != nil {
		return Status{}, err
	}
	newDir := filepath.Join(m.cfg.SiteDir, newToken)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		return Status{}, err
	}
	if err := copyFile(oldStatus.ConfigPath, filepath.Join(newDir, "config.yaml")); err != nil {
		return Status{}, err
	}
	if _, err := os.Stat(oldStatus.ScriptPath); err == nil {
		if err := copyFile(oldStatus.ScriptPath, filepath.Join(newDir, "script.yaml")); err != nil {
			return Status{}, err
		}
	}
	if err := copyFile(oldStatus.GeoIPPath, filepath.Join(newDir, "geoip.dat")); err != nil {
		return Status{}, err
	}
	if err := copyFile(oldStatus.GeoSitePath, filepath.Join(newDir, "geosite.dat")); err != nil {
		return Status{}, err
	}
	if err := writeFileAtomic(m.cfg.TokenFile, []byte(newToken+"\n"), 0o600); err != nil {
		return Status{}, err
	}
	if err := os.RemoveAll(filepath.Dir(oldStatus.ConfigPath)); err != nil {
		return Status{}, err
	}
	if err := m.appendOperation("rotate token success"); err != nil {
		return Status{}, err
	}
	return m.Status()
}

func (m *Manager) UpdateGeodata(ctx context.Context) error {
	if m.cfg.GeodataUpdateScript == "" {
		return fmt.Errorf("geodata update script is not configured")
	}
	if _, err := m.cfg.Runner.Run(ctx, m.cfg.GeodataUpdateScript); err != nil {
		return err
	}
	return m.appendOperation("update geodata success")
}

func (m *Manager) ReadLogs(limit int) (Logs, error) {
	if limit <= 0 {
		limit = 100
	}
	operations, err := readTailLines(m.operationsLogPath(), limit)
	if err != nil {
		return Logs{}, err
	}
	geodata, err := readTailLines(m.cfg.GeodataLogPath, limit)
	if err != nil {
		return Logs{}, err
	}
	return Logs{
		Operations: operations,
		Geodata:    geodata,
	}, nil
}

func (m *Manager) configDraftPath() string {
	return filepath.Join(m.clashDataDir(), "config.draft.yaml")
}

func (m *Manager) scriptDraftPath() string {
	return filepath.Join(m.clashDataDir(), "script.draft.yaml")
}

func (m *Manager) operationsLogPath() string {
	return filepath.Join(m.clashDataDir(), "operations.log")
}

func (m *Manager) clashDataDir() string {
	return filepath.Join(m.cfg.DataDir, "clash")
}

func (m *Manager) validateMergedConfig(ctx context.Context, merged string, status Status) error {
	tmpDir, err := os.MkdirTemp("", "srvdog-clash-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := writeFileAtomic(filepath.Join(tmpDir, "config.yaml"), []byte(merged), 0o644); err != nil {
		return err
	}
	if err := copyFile(status.GeoIPPath, filepath.Join(tmpDir, "geoip.dat")); err != nil {
		return err
	}
	if err := copyFile(status.GeoSitePath, filepath.Join(tmpDir, "geosite.dat")); err != nil {
		return err
	}

	args := []string{
		"run", "--rm",
		"-v", tmpDir + ":/root/.config/mihomo",
		m.cfg.MihomoImage,
		"-t", "-f", "/root/.config/mihomo/config.yaml",
	}
	if _, err := m.cfg.Runner.Run(ctx, "docker", args...); err != nil {
		return fmt.Errorf("validate merged config: %w", err)
	}
	return nil
}

func (m *Manager) appendOperation(message string) error {
	line := fmt.Sprintf("%s %s\n", m.cfg.Now().UTC().Format(time.RFC3339), message)
	if err := os.MkdirAll(filepath.Dir(m.operationsLogPath()), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(m.operationsLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func stripManagedScriptBlock(content string) string {
	start := strings.Index(content, managedScriptStartMarker)
	end := strings.Index(content, managedScriptEndMarker)
	if start == -1 || end == -1 || end < start {
		return content
	}
	end += len(managedScriptEndMarker)
	stripped := content[:start] + content[end:]
	stripped = strings.ReplaceAll(stripped, "\n\n\n", "\n\n")
	return strings.TrimLeft(stripped, "\n")
}

func mergeManagedScriptBlock(baseConfig, script string) string {
	baseConfig = strings.TrimSpace(stripManagedScriptBlock(baseConfig))
	script = strings.TrimSpace(script)
	if script == "" {
		if baseConfig == "" {
			return ""
		}
		return baseConfig + "\n"
	}

	block := managedScriptStartMarker + "\n" + script + "\n" + managedScriptEndMarker
	if idx := strings.Index(baseConfig, "\nrules:"); idx >= 0 {
		return baseConfig[:idx] + "\n\n" + block + baseConfig[idx:] + "\n"
	}
	return baseConfig + "\n\n" + block + "\n"
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func readTailLines(path string, limit int) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type osRunner struct{}

func (osRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
