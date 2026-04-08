package collector

import "testing"

func TestRateWindowAverage(t *testing.T) {
	win := newRateWindow(3)
	win.push(100)
	win.push(200)
	win.push(300)
	if got := win.average(); got != 200 {
		t.Fatalf("average = %v, want 200", got)
	}
}

func TestRateWindowRespectsLimit(t *testing.T) {
	win := newRateWindow(2)
	win.push(100)
	win.push(200)
	win.push(300)
	if got := win.average(); got != 250 {
		t.Fatalf("average = %v, want 250", got)
	}
}
