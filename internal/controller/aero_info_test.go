package controller

import "testing"

func TestParseMigrateStat(t *testing.T) {
	tests := []struct {
		name  string
		stats string
		key   string
		want  int64
	}{
		{
			name:  "key found with non-zero value",
			stats: "cluster_size=3;migrate_progress_send=42;migrate_progress_recv=0",
			key:   "migrate_progress_send",
			want:  42,
		},
		{
			name:  "key found with zero value",
			stats: "cluster_size=3;migrate_progress_send=0;migrate_progress_recv=0",
			key:   "migrate_progress_send",
			want:  0,
		},
		{
			name:  "recv key found",
			stats: "cluster_size=3;migrate_progress_send=0;migrate_progress_recv=15",
			key:   "migrate_progress_recv",
			want:  15,
		},
		{
			name:  "key not found returns zero",
			stats: "cluster_size=3;other_stat=10",
			key:   "migrate_progress_send",
			want:  0,
		},
		{
			name:  "empty stats returns zero",
			stats: "",
			key:   "migrate_progress_send",
			want:  0,
		},
		{
			name:  "non-numeric value returns zero",
			stats: "migrate_progress_send=abc",
			key:   "migrate_progress_send",
			want:  0,
		},
		{
			name:  "key with spaces around value",
			stats: "migrate_progress_send = 10 ",
			key:   "migrate_progress_send",
			want:  10,
		},
		{
			name:  "partial key match does not match",
			stats: "migrate_progress_send_extra=5;migrate_progress_send=3",
			key:   "migrate_progress_send",
			want:  3,
		},
		{
			name:  "large value",
			stats: "migrate_progress_send=999999999",
			key:   "migrate_progress_send",
			want:  999999999,
		},
		{
			name:  "realistic aerospike statistics response",
			stats: "cluster_size=3;cluster_key=ABC123;cluster_integrity=true;migrate_partitions_remaining=100;migrate_progress_send=50;migrate_progress_recv=25;objects=1000000",
			key:   "migrate_progress_send",
			want:  50,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMigrateStat(tc.stats, tc.key)
			if got != tc.want {
				t.Errorf("parseMigrateStat(%q, %q) = %d, want %d", tc.stats, tc.key, got, tc.want)
			}
		})
	}
}
