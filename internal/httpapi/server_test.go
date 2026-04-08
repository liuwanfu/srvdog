package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

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
