package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/liuwanfu/srvdog/internal/clash"
	"github.com/liuwanfu/srvdog/internal/model"
)

func TestSummaryHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/summary", nil)
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		Summary: func() model.Summary {
			return model.Summary{Mode: "low"}
		},
		StaticFS: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("ok")},
			"app.js":     &fstest.MapFile{Data: []byte("ok")},
			"styles.css": &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHeartbeatHandlerAcceptsQueryID(t *testing.T) {
	var touched string
	req := httptest.NewRequest(http.MethodPost, "/api/heartbeat?id=tab-1", nil)
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		TouchViewer: func(id string) {
			touched = id
		},
		StaticFS: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("ok")},
			"app.js":     &fstest.MapFile{Data: []byte("ok")},
			"styles.css": &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if touched != "tab-1" {
		t.Fatalf("touched = %q, want tab-1", touched)
	}
}

func TestChartUtilsScriptServed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chart-utils.js", nil)
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		StaticFS: fstest.MapFS{
			"index.html":     &fstest.MapFile{Data: []byte("ok")},
			"app.js":         &fstest.MapFile{Data: []byte("ok")},
			"chart-utils.js": &fstest.MapFile{Data: []byte("chart utils")},
			"styles.css":     &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "chart utils" {
		t.Fatalf("body = %q, want chart utils", rec.Body.String())
	}
}

func TestClashStatusHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/clash/status", nil)
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		ClashStatus: func() (clash.Status, error) {
			return clash.Status{Token: "abc123"}, nil
		},
		StaticFS: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("ok")},
			"app.js":     &fstest.MapFile{Data: []byte("ok")},
			"styles.css": &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"token":"abc123"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestClashConfigSaveHandler(t *testing.T) {
	var got string
	req := httptest.NewRequest(http.MethodPut, "/api/clash/config", strings.NewReader(`{"content":"mode: rule\n"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		SaveClashConfig: func(content string) error {
			got = content
			return nil
		},
		StaticFS: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("ok")},
			"app.js":     &fstest.MapFile{Data: []byte("ok")},
			"styles.css": &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got != "mode: rule\n" {
		t.Fatalf("saved content = %q", got)
	}
}

func TestClashRotateTokenHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/clash/token/rotate", nil)
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		RotateClashToken: func() (clash.Status, error) {
			return clash.Status{Token: "newtoken"}, nil
		},
		ValidateClashConfig: func(context.Context) error { return nil },
		StaticFS: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("ok")},
			"app.js":     &fstest.MapFile{Data: []byte("ok")},
			"styles.css": &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"token":"newtoken"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
