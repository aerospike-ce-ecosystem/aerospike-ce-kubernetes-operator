package storage

import (
	"testing"
)

func TestIsLocalStorageClass_Match(t *testing.T) {
	localClasses := []string{"local-path", "openebs-hostpath"}
	if !IsLocalStorageClass("local-path", localClasses) {
		t.Error("expected local-path to match")
	}
}

func TestIsLocalStorageClass_NoMatch(t *testing.T) {
	localClasses := []string{"local-path", "openebs-hostpath"}
	if IsLocalStorageClass("standard", localClasses) {
		t.Error("expected standard to NOT match")
	}
}

func TestIsLocalStorageClass_EmptyList(t *testing.T) {
	if IsLocalStorageClass("local-path", nil) {
		t.Error("nil list should not match")
	}
	if IsLocalStorageClass("local-path", []string{}) {
		t.Error("empty list should not match")
	}
}

func TestParsePodName_Valid(t *testing.T) {
	tests := []struct {
		name        string
		podName     string
		wantSTS     string
		wantOrdinal int32
	}{
		{"simple", "my-cluster-1-0", "my-cluster-1", 0},
		{"multi-digit", "my-cluster-1-12", "my-cluster-1", 12},
		{"complex-name", "aerospike-ce-cluster-rack-2-3", "aerospike-ce-cluster-rack-2", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, ordinal, ok := ParsePodName(tt.podName)
			if !ok {
				t.Fatal("expected parsing to succeed")
			}
			if sts != tt.wantSTS {
				t.Errorf("stsName = %q, want %q", sts, tt.wantSTS)
			}
			if ordinal != tt.wantOrdinal {
				t.Errorf("ordinal = %d, want %d", ordinal, tt.wantOrdinal)
			}
		})
	}
}

func TestParsePodName_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		podName string
	}{
		{"no-dash", "mycluster0"},
		{"trailing-dash", "my-cluster-"},
		{"no-number", "my-cluster-abc"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := ParsePodName(tt.podName)
			if ok {
				t.Error("expected parsing to fail")
			}
		})
	}
}
