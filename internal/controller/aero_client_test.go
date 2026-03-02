package controller

import (
	"testing"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestGetServicePort(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ackov1alpha1.AerospikeCluster
		want    int
	}{
		{
			name:    "nil AerospikeConfig returns default",
			cluster: &ackov1alpha1.AerospikeCluster{},
			want:    defaultAeroPort,
		},
		{
			name: "empty config returns default",
			cluster: &ackov1alpha1.AerospikeCluster{
				Spec: ackov1alpha1.AerospikeClusterSpec{
					AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
						Value: map[string]any{},
					},
				},
			},
			want: defaultAeroPort,
		},
		{
			name: "no network section returns default",
			cluster: &ackov1alpha1.AerospikeCluster{
				Spec: ackov1alpha1.AerospikeClusterSpec{
					AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
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
			cluster: &ackov1alpha1.AerospikeCluster{
				Spec: ackov1alpha1.AerospikeClusterSpec{
					AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
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
			cluster: &ackov1alpha1.AerospikeCluster{
				Spec: ackov1alpha1.AerospikeClusterSpec{
					AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
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
			cluster: &ackov1alpha1.AerospikeCluster{
				Spec: ackov1alpha1.AerospikeClusterSpec{
					AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
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
