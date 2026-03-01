package controller

import (
	"testing"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestGetServicePort(t *testing.T) {
	tests := []struct {
		name    string
		cluster *asdbcev1alpha1.AerospikeCECluster
		want    int
	}{
		{
			name:    "nil AerospikeConfig returns default",
			cluster: &asdbcev1alpha1.AerospikeCECluster{},
			want:    defaultAeroPort,
		},
		{
			name: "empty config returns default",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					AerospikeConfig: &asdbcev1alpha1.AerospikeConfigSpec{
						Value: map[string]any{},
					},
				},
			},
			want: defaultAeroPort,
		},
		{
			name: "no network section returns default",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					AerospikeConfig: &asdbcev1alpha1.AerospikeConfigSpec{
						Value: map[string]any{
							"service": map[string]any{"cluster-name": "test"},
						},
					},
				},
			},
			want: defaultAeroPort,
		},
		{
			name: "no service in network returns default",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					AerospikeConfig: &asdbcev1alpha1.AerospikeConfigSpec{
						Value: map[string]any{
							"network": map[string]any{
								"heartbeat": map[string]any{"port": 3002},
							},
						},
					},
				},
			},
			want: defaultAeroPort,
		},
		{
			name: "custom port as int",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					AerospikeConfig: &asdbcev1alpha1.AerospikeConfigSpec{
						Value: map[string]any{
							"network": map[string]any{
								"service": map[string]any{"port": 4000},
							},
						},
					},
				},
			},
			want: 4000,
		},
		{
			name: "custom port as float64 (JSON deserialization)",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					AerospikeConfig: &asdbcev1alpha1.AerospikeConfigSpec{
						Value: map[string]any{
							"network": map[string]any{
								"service": map[string]any{"port": float64(5000)},
							},
						},
					},
				},
			},
			want: 5000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getServicePort(tc.cluster)
			if got != tc.want {
				t.Errorf("getServicePort() = %d, want %d", got, tc.want)
			}
		})
	}
}
