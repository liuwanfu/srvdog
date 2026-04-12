package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/liuwanfu/srvdog/internal/clash"
	"github.com/liuwanfu/srvdog/internal/collector"
	"github.com/liuwanfu/srvdog/internal/history"
	"github.com/liuwanfu/srvdog/internal/model"
	"github.com/liuwanfu/srvdog/internal/realtime"
)

type Config struct {
	ListenAddr           string
	DataDir              string
	LowFrequency         time.Duration
	HighFrequency        time.Duration
	DockerFrequency      time.Duration
	ViewerTimeout        time.Duration
	HousekeepingInterval time.Duration
	RealtimeCapacity     int
	DefaultRetentionDays int
	ClashTokenFile       string
	ClashSiteDir         string
	ClashPublicBaseURL   string
	ClashGeodataScript   string
	ClashGeodataLogPath  string
	ClashMihomoImage     string
}

type Service struct {
	cfg             Config
	hostCollector   *collector.HostCollector
	dockerCollector *collector.DockerCollector
	historyStore    *history.Store
	settings        *SettingsStore
	realtime        *realtime.Buffer
	viewerTracker   *realtime.ViewerTracker
	clashManager    *clash.Manager

	mu          sync.RWMutex
	lastSample  model.Sample
	lastDocker  []model.DockerContainer
	dockerError string
}

func DefaultConfig() Config {
	return Config{
		ListenAddr:           "127.0.0.1:8090",
		DataDir:              "data",
		LowFrequency:         5 * time.Minute,
		HighFrequency:        2 * time.Second,
		DockerFrequency:      30 * time.Second,
		ViewerTimeout:        20 * time.Second,
		HousekeepingInterval: time.Hour,
		RealtimeCapacity:     1800,
		DefaultRetentionDays: 7,
		ClashTokenFile:       envOrDefault("SRVDOG_CLASH_TOKEN_FILE", "/root/mihomo-subscription/token"),
		ClashSiteDir:         envOrDefault("SRVDOG_CLASH_SITE_DIR", "/opt/cypht/data/site-wg"),
		ClashPublicBaseURL:   envOrDefault("SRVDOG_CLASH_PUBLIC_BASE_URL", "http://107.174.48.241/wg"),
		ClashGeodataScript:   envOrDefault("SRVDOG_CLASH_GEODATA_SCRIPT", "/usr/local/bin/update-mihomo-geodata.sh"),
		ClashGeodataLogPath:  envOrDefault("SRVDOG_CLASH_GEODATA_LOG_PATH", "/var/log/update-mihomo-geodata.log"),
		ClashMihomoImage:     envOrDefault("SRVDOG_CLASH_MIHOMO_IMAGE", "docker.io/metacubex/mihomo:Alpha"),
	}
}

func NewService(cfg Config) (*Service, error) {
	if cfg.DefaultRetentionDays < 1 || cfg.DefaultRetentionDays > 30 {
		return nil, fmt.Errorf("retention days must be between 1 and 30")
	}
	settings, err := NewSettingsStore(filepath.Join(cfg.DataDir, "settings.json"), Settings{
		RetentionDays: cfg.DefaultRetentionDays,
	})
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:             cfg,
		hostCollector:   collector.NewHostCollector("/"),
		dockerCollector: collector.NewDockerCollector(5 * time.Second),
		historyStore:    &history.Store{Dir: filepath.Join(cfg.DataDir, "history")},
		settings:        settings,
		realtime:        realtime.NewBuffer(cfg.RealtimeCapacity),
		viewerTracker:   realtime.NewViewerTracker(cfg.ViewerTimeout),
		clashManager: clash.NewManager(clash.Config{
			DataDir:             cfg.DataDir,
			TokenFile:           cfg.ClashTokenFile,
			SiteDir:             cfg.ClashSiteDir,
			PublicBaseURL:       cfg.ClashPublicBaseURL,
			MihomoImage:         cfg.ClashMihomoImage,
			GeodataUpdateScript: cfg.ClashGeodataScript,
			GeodataLogPath:      cfg.ClashGeodataLogPath,
		}),
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.cleanupExpired(); err != nil {
		return err
	}
	if err := s.collectLow(); err != nil {
		return err
	}
	s.collectDocker(ctx)

	go s.runLowLoop(ctx)
	go s.runHighLoop(ctx)
	go s.runDockerLoop(ctx)
	go s.runHousekeepingLoop(ctx)
	return nil
}

func (s *Service) runLowLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.LowFrequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.collectLow()
		}
	}
}

func (s *Service) runHighLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.HighFrequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.viewerTracker.Active(time.Now()) {
				continue
			}
			_ = s.collectRealtime()
		}
	}
}

func (s *Service) runDockerLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.DockerFrequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectDocker(ctx)
		}
	}
}

func (s *Service) runHousekeepingLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.HousekeepingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.cleanupExpired()
		}
	}
}

func (s *Service) collectLow() error {
	sample, err := s.hostCollector.Collect(time.Now())
	if err != nil {
		return err
	}
	s.updateSample(sample)
	return s.historyStore.Append(sample)
}

func (s *Service) collectRealtime() error {
	sample, err := s.hostCollector.Collect(time.Now())
	if err != nil {
		return err
	}
	s.updateSample(sample)
	s.realtime.Add(sample)
	return nil
}

func (s *Service) collectDocker(ctx context.Context) {
	containers, err := s.dockerCollector.Collect(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.dockerError = err.Error()
		return
	}
	s.lastDocker = containers
	s.dockerError = ""
}

func (s *Service) updateSample(sample model.Sample) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSample = sample
}

func (s *Service) Summary() model.Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return model.Summary{
		UpdatedAt:     s.lastSample.Timestamp,
		Mode:          s.currentMode(),
		RetentionDays: s.settings.Get().RetentionDays,
		Sample:        s.lastSample,
		Docker:        append([]model.DockerContainer(nil), s.lastDocker...),
		DockerError:   s.dockerError,
	}
}

func (s *Service) History(window string) ([]model.Sample, error) {
	start, end, err := parseWindow(window, time.Now())
	if err != nil {
		return nil, err
	}
	return s.historyStore.ReadRange(start, end)
}

func (s *Service) Realtime() []model.Sample {
	return s.realtime.Snapshot()
}

func (s *Service) TouchViewer(id string) {
	s.viewerTracker.Touch(id, time.Now())
}

func (s *Service) SetRetention(days int) error {
	if days < 1 || days > 30 {
		return fmt.Errorf("retention days must be between 1 and 30")
	}
	if err := s.settings.SetRetention(days); err != nil {
		return err
	}
	return s.cleanupExpired()
}

func (s *Service) Export(format, window string) ([]byte, string, error) {
	samples, err := s.History(window)
	if err != nil {
		return nil, "", err
	}
	realtimeSamples := s.Realtime()
	switch format {
	case "", "json":
		return history.ExportJSON(window, samples, realtimeSamples)
	case "csv":
		return history.ExportCSV(samples, realtimeSamples)
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func (s *Service) ClearHistory() error {
	if err := s.historyStore.Clear(); err != nil {
		return err
	}
	s.realtime.Clear()
	return nil
}

func (s *Service) currentInterval() time.Duration {
	if s.viewerTracker.Active(time.Now()) {
		return s.cfg.HighFrequency
	}
	return s.cfg.LowFrequency
}

func (s *Service) currentMode() string {
	if s.viewerTracker.Active(time.Now()) {
		return "high"
	}
	return "low"
}

func (s *Service) cleanupExpired() error {
	retention := s.settings.Get().RetentionDays
	cutoff := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -retention)
	return s.historyStore.CleanupExpired(cutoff)
}

func parseWindow(window string, now time.Time) (time.Time, time.Time, error) {
	end := now.UTC()
	switch window {
	case "", "1h":
		return end.Add(-time.Hour), end, nil
	case "6h":
		return end.Add(-6 * time.Hour), end, nil
	case "24h":
		return end.Add(-24 * time.Hour), end, nil
	case "7d":
		return end.AddDate(0, 0, -7), end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported window: %s", window)
	}
}

func (s *Service) ClashStatus() (clash.Status, error) {
	return s.clashManager.Status()
}

func (s *Service) ClashConfig() (clash.Document, error) {
	return s.clashManager.GetConfig()
}

func (s *Service) SaveClashConfig(content string) error {
	return s.clashManager.SaveConfigDraft(content)
}

func (s *Service) ValidateClashConfig(ctx context.Context) error {
	return s.clashManager.ValidateConfig(ctx)
}

func (s *Service) PublishClashConfig(ctx context.Context) error {
	return s.clashManager.PublishConfig(ctx)
}

func (s *Service) ClashScript() (clash.Document, error) {
	return s.clashManager.GetScript()
}

func (s *Service) SaveClashScript(content string) error {
	return s.clashManager.SaveScriptDraft(content)
}

func (s *Service) ValidateClashScript(ctx context.Context) error {
	return s.clashManager.ValidateScript(ctx)
}

func (s *Service) PublishClashScript(ctx context.Context) error {
	return s.clashManager.PublishScript(ctx)
}

func (s *Service) UpdateClashGeodata(ctx context.Context) error {
	return s.clashManager.UpdateGeodata(ctx)
}

func (s *Service) RotateClashToken() (clash.Status, error) {
	return s.clashManager.RotateToken()
}

func (s *Service) ClashLogs(limit int) (clash.Logs, error) {
	return s.clashManager.ReadLogs(limit)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
