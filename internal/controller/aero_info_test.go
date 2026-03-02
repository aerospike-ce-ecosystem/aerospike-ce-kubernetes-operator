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
			name:  "migrate_partitions_remaining non-zero",
			stats: "cluster_size=3;migrate_partitions_remaining=42;objects=1000000",
			key:   "migrate_partitions_remaining",
			want:  42,
		},
		{
			name:  "migrate_partitions_remaining zero",
			stats: "cluster_size=3;migrate_partitions_remaining=0;objects=1000000",
			key:   "migrate_partitions_remaining",
			want:  0,
		},
		{
			name:  "key not found returns zero",
			stats: "cluster_size=3;other_stat=10",
			key:   "migrate_partitions_remaining",
			want:  0,
		},
		{
			name:  "empty stats returns zero",
			stats: "",
			key:   "migrate_partitions_remaining",
			want:  0,
		},
		{
			name:  "non-numeric value returns zero",
			stats: "migrate_partitions_remaining=abc",
			key:   "migrate_partitions_remaining",
			want:  0,
		},
		{
			name:  "key with spaces around value",
			stats: "migrate_partitions_remaining = 10 ",
			key:   "migrate_partitions_remaining",
			want:  10,
		},
		{
			name:  "partial key match does not match",
			stats: "migrate_partitions_remaining_extra=5;migrate_partitions_remaining=3",
			key:   "migrate_partitions_remaining",
			want:  3,
		},
		{
			name:  "large value",
			stats: "migrate_partitions_remaining=999999999",
			key:   "migrate_partitions_remaining",
			want:  999999999,
		},
		{
			name:  "realistic aerospike CE 8.x statistics response",
			stats: "cluster_size=3;cluster_key=ABC123;cluster_integrity=true;migrate_partitions_remaining=100;objects=1000000",
			key:   "migrate_partitions_remaining",
			want:  100,
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
