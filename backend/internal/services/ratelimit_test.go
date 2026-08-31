package services

import (
	"testing"
	"time"
)

func TestFixedWindowLimiterAllowsUpToLimit(t *testing.T) {
	lim := NewFixedWindowLimiter(3, time.Minute, 0)
	for i := 0; i < 3; i++ {
		if !lim.Allow("10.0.0.1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if lim.Allow("10.0.0.1") {
		t.Fatal("request 4 should be rejected")
	}
}

func TestFixedWindowLimiterIndependentPerKey(t *testing.T) {
	lim := NewFixedWindowLimiter(1, time.Minute, 0)
	if !lim.Allow("10.0.0.1") {
		t.Fatal("first key should be allowed")
	}
	if lim.Allow("10.0.0.1") {
		t.Fatal("second hit for first key should be rejected")
	}
	if !lim.Allow("10.0.0.2") {
		t.Fatal("separate key should be allowed")
	}
}

func TestFixedWindowLimiterResetsAfterWindow(t *testing.T) {
	lim := NewFixedWindowLimiter(1, 50*time.Millisecond, 0)
	if !lim.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}
	if lim.Allow("user-1") {
		t.Fatal("second request in same window should be rejected")
	}
	time.Sleep(60 * time.Millisecond)
	if !lim.Allow("user-1") {
		t.Fatal("request after window expiry should be allowed")
	}
}

func TestFixedWindowLimiterCapsTrackedKeys(t *testing.T) {
	lim := NewFixedWindowLimiter(10, time.Minute, 2)
	if !lim.Allow("a") {
		t.Fatal("key a should be allowed")
	}
	if !lim.Allow("b") {
		t.Fatal("key b should be allowed")
	}
	// Adding a third distinct key evicts the expired window of the eldest by
	// pruning; the new key must still be allowed so a fresh client is not
	// blocked just because the map grew.
	if !lim.Allow("c") {
		t.Fatal("key c should be allowed")
	}
}
