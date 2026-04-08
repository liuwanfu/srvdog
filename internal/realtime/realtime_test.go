package realtime

import (
	"testing"
	"time"
)

func TestViewerTrackerExpires(t *testing.T) {
	tracker := NewViewerTracker(2 * time.Second)
	tracker.Touch("viewer-1", time.Unix(0, 0))
	if !tracker.Active(time.Unix(1, 0)) {
		t.Fatal("expected active viewer")
	}
	if tracker.Active(time.Unix(3, 0)) {
		t.Fatal("expected viewer to expire")
	}
}
