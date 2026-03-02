package controller

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name      string
		failCount int32
		want      time.Duration
	}{
		{
			name:      "zero failures returns default retry interval",
			failCount: 0,
			want:      defaultReconcileRetryInterval,
		},
		{
			name:      "negative failures returns default retry interval",
			failCount: -1,
			want:      defaultReconcileRetryInterval,
		},
		{
			name:      "1 failure: 2^1 = 2s",
			failCount: 1,
			want:      2 * time.Second,
		},
		{
			name:      "2 failures: 2^2 = 4s",
			failCount: 2,
			want:      4 * time.Second,
		},
		{
			name:      "3 failures: 2^3 = 8s",
			failCount: 3,
			want:      8 * time.Second,
		},
		{
			name:      "4 failures: 2^4 = 16s",
			failCount: 4,
			want:      16 * time.Second,
		},
		{
			name:      "5 failures: 2^5 = 32s",
			failCount: 5,
			want:      32 * time.Second,
		},
		{
			name:      "6 failures: 2^6 = 64s",
			failCount: 6,
			want:      64 * time.Second,
		},
		{
			name:      "7 failures: 2^7 = 128s",
			failCount: 7,
			want:      128 * time.Second,
		},
		{
			name:      "8 failures: 2^8 = 256s",
			failCount: 8,
			want:      256 * time.Second,
		},
		{
			name:      "9 failures: capped at 2^8 = 256s (exponent capped at 8)",
			failCount: 9,
			want:      256 * time.Second,
		},
		{
			name:      "10 failures: capped at 2^8 = 256s",
			failCount: 10,
			want:      256 * time.Second,
		},
		{
			name:      "100 failures: capped at 2^8 = 256s",
			failCount: 100,
			want:      256 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateBackoff(tc.failCount)
			if got != tc.want {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tc.failCount, got, tc.want)
			}
		})
	}
}

func TestCalculateBackoff_MonotonicallyIncreasing(t *testing.T) {
	// Backoff should be monotonically increasing from failCount=1 onward.
	// failCount=0 returns the default retry interval which is a special case.
	prev := calculateBackoff(1)
	for i := int32(2); i <= 12; i++ {
		cur := calculateBackoff(i)
		if cur < prev {
			t.Errorf("calculateBackoff(%d) = %v < calculateBackoff(%d) = %v: backoff should be monotonically increasing",
				i, cur, i-1, prev)
		}
		prev = cur
	}
}

func TestCalculateBackoff_NeverExceedsMax(t *testing.T) {
	maxDuration := time.Duration(maxBackoffSeconds) * time.Second
	for i := int32(0); i <= 1000; i++ {
		got := calculateBackoff(i)
		if got > maxDuration {
			t.Errorf("calculateBackoff(%d) = %v exceeds maxBackoffSeconds (%v)", i, got, maxDuration)
		}
	}
}

func TestCircuitBreakerConstants(t *testing.T) {
	if maxFailedReconciles <= 0 {
		t.Errorf("maxFailedReconciles should be positive, got %d", maxFailedReconciles)
	}
	if maxBackoffSeconds <= 0 {
		t.Errorf("maxBackoffSeconds should be positive, got %d", maxBackoffSeconds)
	}
	if reconcileTimeout <= 0 {
		t.Errorf("reconcileTimeout should be positive, got %v", reconcileTimeout)
	}
	if reconcileTimeout != 5*time.Minute {
		t.Errorf("reconcileTimeout = %v, want 5m", reconcileTimeout)
	}
	if maxFailedReconciles != 10 {
		t.Errorf("maxFailedReconciles = %d, want 10", maxFailedReconciles)
	}
}
