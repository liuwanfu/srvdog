package realtime

import (
	"sync"
	"time"

	"github.com/liuwanfu/srvdog/internal/model"
)

type Buffer struct {
	mu      sync.RWMutex
	samples []model.Sample
	next    int
	full    bool
}

func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &Buffer{samples: make([]model.Sample, capacity)}
}

func (b *Buffer) Add(sample model.Sample) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.samples[b.next] = sample
	b.next = (b.next + 1) % len(b.samples)
	if b.next == 0 {
		b.full = true
	}
}

func (b *Buffer) Snapshot() []model.Sample {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.full && b.next == 0 {
		return nil
	}
	if !b.full {
		out := make([]model.Sample, b.next)
		copy(out, b.samples[:b.next])
		return out
	}
	out := make([]model.Sample, 0, len(b.samples))
	out = append(out, b.samples[b.next:]...)
	out = append(out, b.samples[:b.next]...)
	return out
}

func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.samples = make([]model.Sample, len(b.samples))
	b.next = 0
	b.full = false
}

type ViewerTracker struct {
	mu      sync.Mutex
	timeout time.Duration
	viewers map[string]time.Time
}

func NewViewerTracker(timeout time.Duration) *ViewerTracker {
	return &ViewerTracker{
		timeout: timeout,
		viewers: make(map[string]time.Time),
	}
}

func (t *ViewerTracker) Touch(id string, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if id == "" {
		return
	}
	t.viewers[id] = now
}

func (t *ViewerTracker) Active(now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cleanupLocked(now)
	return len(t.viewers) > 0
}

func (t *ViewerTracker) cleanupLocked(now time.Time) {
	for id, touchedAt := range t.viewers {
		if now.Sub(touchedAt) > t.timeout {
			delete(t.viewers, id)
		}
	}
}
