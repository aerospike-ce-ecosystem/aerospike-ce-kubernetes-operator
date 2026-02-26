package controller

import (
	"testing"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestGetRacks(t *testing.T) {
	r := &AerospikeCEClusterReconciler{}

	tests := []struct {
		name    string
		cluster *asdbcev1alpha1.AerospikeCECluster
		wantLen int
		wantIDs []int
	}{
		{
			name: "nil RackConfig returns default rack with ID 0",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					RackConfig: nil,
				},
			},
			wantLen: 1,
			wantIDs: []int{0},
		},
		{
			name: "empty Racks slice returns default rack with ID 0",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					RackConfig: &asdbcev1alpha1.RackConfig{
						Racks: []asdbcev1alpha1.Rack{},
					},
				},
			},
			wantLen: 1,
			wantIDs: []int{0},
		},
		{
			name: "populated Racks returns racks as-is",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					RackConfig: &asdbcev1alpha1.RackConfig{
						Racks: []asdbcev1alpha1.Rack{
							{ID: 1},
							{ID: 2},
							{ID: 3},
						},
					},
				},
			},
			wantLen: 3,
			wantIDs: []int{1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := r.getRacks(tc.cluster)
			if len(got) != tc.wantLen {
				t.Fatalf("getRacks() returned %d racks, want %d", len(got), tc.wantLen)
			}
			for i, wantID := range tc.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("getRacks()[%d].ID = %d, want %d", i, got[i].ID, wantID)
				}
			}
		})
	}
}

func TestGetRackSize(t *testing.T) {
	r := &AerospikeCEClusterReconciler{}

	tests := []struct {
		name      string
		totalSize int32
		racks     []asdbcev1alpha1.Rack
		rackIndex int
		want      int32
	}{
		{
			name:      "1 rack, 5 pods",
			totalSize: 5,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}},
			rackIndex: 0,
			want:      5,
		},
		{
			name:      "2 racks even split (6 pods) - rack 0",
			totalSize: 6,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}},
			rackIndex: 0,
			want:      3,
		},
		{
			name:      "2 racks even split (6 pods) - rack 1",
			totalSize: 6,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}},
			rackIndex: 1,
			want:      3,
		},
		{
			name:      "2 racks uneven split (5 pods) - rack 0 gets extra",
			totalSize: 5,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}},
			rackIndex: 0,
			want:      3,
		},
		{
			name:      "2 racks uneven split (5 pods) - rack 1",
			totalSize: 5,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}},
			rackIndex: 1,
			want:      2,
		},
		{
			name:      "3 racks, 7 pods - rack 0 gets extra",
			totalSize: 7,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}, {ID: 2}},
			rackIndex: 0,
			want:      3,
		},
		{
			name:      "3 racks, 7 pods - rack 1",
			totalSize: 7,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}, {ID: 2}},
			rackIndex: 1,
			want:      2,
		},
		{
			name:      "3 racks, 7 pods - rack 2",
			totalSize: 7,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}, {ID: 2}},
			rackIndex: 2,
			want:      2,
		},
		{
			name:      "rack count > pod count (3 racks, 2 pods) - rack 0",
			totalSize: 2,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}, {ID: 2}},
			rackIndex: 0,
			want:      1,
		},
		{
			name:      "rack count > pod count (3 racks, 2 pods) - rack 1",
			totalSize: 2,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}, {ID: 2}},
			rackIndex: 1,
			want:      1,
		},
		{
			name:      "rack count > pod count (3 racks, 2 pods) - rack 2",
			totalSize: 2,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}, {ID: 2}},
			rackIndex: 2,
			want:      0,
		},
		{
			name:      "0 pods - all racks get 0 (rack 0)",
			totalSize: 0,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}},
			rackIndex: 0,
			want:      0,
		},
		{
			name:      "0 pods - all racks get 0 (rack 1)",
			totalSize: 0,
			racks:     []asdbcev1alpha1.Rack{{ID: 0}, {ID: 1}},
			rackIndex: 1,
			want:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					Size: tc.totalSize,
				},
			}
			got := r.getRackSize(cluster, tc.racks, tc.rackIndex)
			if got != tc.want {
				t.Errorf("getRackSize(size=%d, numRacks=%d, rackIndex=%d) = %d, want %d",
					tc.totalSize, len(tc.racks), tc.rackIndex, got, tc.want)
			}
		})
	}
}

func TestGetRackSize_SumInvariant(t *testing.T) {
	r := &AerospikeCEClusterReconciler{}

	tests := []struct {
		totalSize int32
		numRacks  int
	}{
		{0, 1},
		{1, 1},
		{5, 1},
		{6, 2},
		{5, 2},
		{7, 3},
		{2, 3},
		{8, 3},
		{1, 4},
		{8, 8},
		{3, 5},
	}

	for _, tc := range tests {
		racks := make([]asdbcev1alpha1.Rack, tc.numRacks)
		for i := range racks {
			racks[i] = asdbcev1alpha1.Rack{ID: i}
		}
		cluster := &asdbcev1alpha1.AerospikeCECluster{
			Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
				Size: tc.totalSize,
			},
		}

		var sum int32
		for i := range racks {
			sum += r.getRackSize(cluster, racks, i)
		}
		if sum != tc.totalSize {
			t.Errorf("sum of getRackSize(totalSize=%d, numRacks=%d) = %d, want %d",
				tc.totalSize, tc.numRacks, sum, tc.totalSize)
		}
	}
}
