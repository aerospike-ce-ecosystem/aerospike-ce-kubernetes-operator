package controller

import (
	"testing"
)

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want map[string]any
	}{
		{
			name: "nil map returns empty map (not nil)",
			in:   nil,
			want: map[string]any{},
		},
		{
			name: "empty map returns empty map",
			in:   map[string]string{},
			want: map[string]any{},
		},
		{
			name: "single entry is converted",
			in:   map[string]string{"app": "aerospike"},
			want: map[string]any{"app": "aerospike"},
		},
		{
			name: "multiple entries are all converted",
			in: map[string]string{
				"app":  "aerospike",
				"team": "platform",
				"env":  "prod",
			},
			want: map[string]any{
				"app":  "aerospike",
				"team": "platform",
				"env":  "prod",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toStringMap(tc.in)
			if got == nil {
				t.Fatal("toStringMap() returned nil, want non-nil map")
			}
			if len(got) != len(tc.want) {
				t.Fatalf("toStringMap() returned %d entries, want %d", len(got), len(tc.want))
			}
			for k, wantVal := range tc.want {
				gotVal, ok := got[k]
				if !ok {
					t.Errorf("toStringMap() missing key %q", k)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("toStringMap()[%q] = %v, want %v", k, gotVal, wantVal)
				}
			}
		})
	}
}
