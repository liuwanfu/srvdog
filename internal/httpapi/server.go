package httpapi

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/liuwanfu/srvdog/internal/clash"
	"github.com/liuwanfu/srvdog/internal/model"
)

type Dependencies struct {
	Summary             func() model.Summary
	History             func(string) ([]model.Sample, error)
	Realtime            func() []model.Sample
	TouchViewer         func(string)
	SetRetention        func(int) error
	Export              func(string, string) ([]byte, string, error)
	ClearHistory        func() error
	ClashStatus         func() (clash.Status, error)
	ClashConfig         func() (clash.Document, error)
	SaveClashConfig     func(string) error
	ValidateClashConfig func(context.Context) error
	PublishClashConfig  func(context.Context) error
	ClashScript         func() (clash.Document, error)
	SaveClashScript     func(string) error
	ValidateClashScript func(context.Context) error
	PublishClashScript  func(context.Context) error
	UpdateClashGeodata  func(context.Context) error
	RotateClashToken    func() (clash.Status, error)
	ClashLogs           func(int) (clash.Logs, error)
	StaticFS            fs.FS
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
	mux.Handle("GET /api/clash/status", http.HandlerFunc(s.handleClashStatus))
	mux.Handle("GET /api/clash/config", http.HandlerFunc(s.handleClashConfig))
	mux.Handle("PUT /api/clash/config", http.HandlerFunc(s.handleClashSaveConfig))
	mux.Handle("POST /api/clash/config/validate", http.HandlerFunc(s.handleClashValidateConfig))
	mux.Handle("POST /api/clash/config/publish", http.HandlerFunc(s.handleClashPublishConfig))
	mux.Handle("GET /api/clash/script", http.HandlerFunc(s.handleClashScript))
	mux.Handle("PUT /api/clash/script", http.HandlerFunc(s.handleClashSaveScript))
	mux.Handle("POST /api/clash/script/validate", http.HandlerFunc(s.handleClashValidateScript))
	mux.Handle("POST /api/clash/script/publish", http.HandlerFunc(s.handleClashPublishScript))
	mux.Handle("POST /api/clash/geodata/update", http.HandlerFunc(s.handleClashUpdateGeodata))
	mux.Handle("POST /api/clash/token/rotate", http.HandlerFunc(s.handleClashRotateToken))
	mux.Handle("GET /api/clash/logs", http.HandlerFunc(s.handleClashLogs))
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

func (s *Server) handleClashStatus(w http.ResponseWriter, _ *http.Request) {
	status, err := s.deps.ClashStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleClashConfig(w http.ResponseWriter, _ *http.Request) {
	doc, err := s.deps.ClashConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleClashSaveConfig(w http.ResponseWriter, r *http.Request) {
	content, err := decodeContent(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.deps.SaveClashConfig(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Server) handleClashValidateConfig(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.ValidateClashConfig(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "valid"})
}

func (s *Server) handleClashPublishConfig(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.PublishClashConfig(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (s *Server) handleClashScript(w http.ResponseWriter, _ *http.Request) {
	doc, err := s.deps.ClashScript()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleClashSaveScript(w http.ResponseWriter, r *http.Request) {
	content, err := decodeContent(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.deps.SaveClashScript(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Server) handleClashValidateScript(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.ValidateClashScript(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "valid"})
}

func (s *Server) handleClashPublishScript(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.PublishClashScript(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (s *Server) handleClashUpdateGeodata(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.UpdateClashGeodata(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleClashRotateToken(w http.ResponseWriter, _ *http.Request) {
	status, err := s.deps.RotateClashToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleClashLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	logs, err := s.deps.ClashLogs(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func decodeContent(r *http.Request) (string, error) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.Content, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
