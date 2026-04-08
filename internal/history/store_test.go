package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liuwanfu/srvdog/internal/model"
)

func TestCleanupExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "2026-03-31.jsonl")
	newFile := filepath.Join(dir, "2026-04-08.jsonl")
	if err := os.WriteFile(oldFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := Store{Dir: dir}
	cutoff := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -7)
	if err := store.CleanupExpired(cutoff); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("expected old file removed, got err=%v", err)
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("expected new file to remain, got err=%v", err)
	}
}

func TestAppendAndReadRange(t *testing.T) {
	dir := t.TempDir()
	store := Store{Dir: dir}
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	sample := model.Sample{Timestamp: now, CPUPercent: 12.5}

	if err := store.Append(sample); err != nil {
		t.Fatal(err)
	}
	out, err := store.ReadRange(now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].CPUPercent != 12.5 {
		t.Fatalf("cpu = %v, want 12.5", out[0].CPUPercent)
	}
}
