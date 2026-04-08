package app

import (
	"testing"
	"time"
)

func TestSamplingModeSwitch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataDir = t.TempDir()
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	svc.viewerTracker.Touch("tab-1", time.Now())
	if got := svc.currentInterval(); got != 2*time.Second {
		t.Fatalf("interval = %v, want 2s", got)
	}
}
