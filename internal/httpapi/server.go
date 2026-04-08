package httpapi

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/liuwanfu/srvdog/internal/model"
)

type Dependencies struct {
	Summary      func() model.Summary
	History      func(string) ([]model.Sample, error)
	Realtime     func() []model.Sample
	TouchViewer  func(string)
	SetRetention func(int) error
	Export       func(string, string) ([]byte, string, error)
	ClearHistory func() error
	StaticFS     fs.FS
}

type Server struct {
	deps Dependencies
}

func NewServer(deps Dependencies) *Server {
	return &Server{deps: deps}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.FS(s.deps.StaticFS))
	mux.Handle("GET /app.js", fileServer)
	mux.Handle("GET /styles.css", fileServer)
	mux.Handle("GET /", http.HandlerFunc(s.handleIndex))
	mux.Handle("GET /api/summary", http.HandlerFunc(s.handleSummary))
	mux.Handle("GET /api/history", http.HandlerFunc(s.handleHistory))
	mux.Handle("GET /api/realtime", http.HandlerFunc(s.handleRealtime))
	mux.Handle("POST /api/heartbeat", http.HandlerFunc(s.handleHeartbeat))
	mux.Handle("POST /api/settings/retention", http.HandlerFunc(s.handleRetention))
	mux.Handle("GET /api/export", http.HandlerFunc(s.handleExport))
	mux.Handle("POST /api/history/clear", http.HandlerFunc(s.handleClearHistory))
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	data, err := fs.ReadFile(s.deps.StaticFS, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.deps.Summary())
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "24h"
	}
	samples, err := s.deps.History(window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"window":  window,
		"samples": samples,
	})
}

func (s *Server) handleRealtime(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"samples": s.deps.Realtime(),
	})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	id := body.ID
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		id = r.Header.Get("X-Viewer-ID")
	}
	if id != "" && s.deps.TouchViewer != nil {
		s.deps.TouchViewer(id)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRetention(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.deps.SetRetention(body.Days); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"retention_days": body.Days})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "24h"
	}
	data, contentType, err := s.deps.Export(format, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ext := "json"
	if format == "csv" {
		ext = "csv"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\"srvdog-export-"+window+"."+ext+"\"")
	_, _ = w.Write(data)
}

func (s *Server) handleClearHistory(w http.ResponseWriter, _ *http.Request) {
	if err := s.deps.ClearHistory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
