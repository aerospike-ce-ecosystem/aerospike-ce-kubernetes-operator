package controller

import (
	"testing"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestGetEffectiveConfig(t *testing.T) {
	r := &AerospikeClusterReconciler{}

	tests := []struct {
		name          string
		clusterConfig *ackov1alpha1.AerospikeConfigSpec
		rackConfig    *ackov1alpha1.AerospikeConfigSpec
		wantNil       bool
		// checkKey and checkVal are used for non-nil results to verify a specific top-level key.
		checkKey string
		checkVal any
	}{
		{
			name:          "both nil returns nil",
			clusterConfig: nil,
			rackConfig:    nil,
			wantNil:       true,
		},
		{
			name: "cluster config set, rack nil returns cluster config",
			clusterConfig: &ackov1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"service": map[string]any{
						"cluster-name": "test-cluster",
					},
				},
			},
			rackConfig: nil,
			wantNil:    false,
			checkKey:   "service",
		},
		{
			name:          "cluster config nil, rack config set returns rack config",
			clusterConfig: nil,
			rackConfig: &ackov1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"service": map[string]any{
						"proto-fd-max": 10000,
					},
				},
			},
			wantNil:  false,
			checkKey: "service",
		},
		{
			name: "both set returns deep merge with rack overriding cluster",
			clusterConfig: &ackov1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"service": map[string]any{
						"cluster-name": "test-cluster",
						"proto-fd-max": 15000,
					},
					"network": map[string]any{
						"service": map[string]any{
							"port": 3000,
						},
					},
				},
			},
			rackConfig: &ackov1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"service": map[string]any{
						"proto-fd-max": 20000,
					},
				},
			},
			wantNil:  false,
			checkKey: "service",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &ackov1alpha1.AerospikeCluster{
				Spec: ackov1alpha1.AerospikeClusterSpec{
					AerospikeConfig: tc.clusterConfig,
				},
			}
			rack := &ackov1alpha1.Rack{
				ID:              0,
				AerospikeConfig: tc.rackConfig,
			}

			got := r.getEffectiveConfig(cluster, rack)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("getEffectiveConfig() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("getEffectiveConfig() = nil, want non-nil")
			}

			if tc.checkKey != "" {
				if _, ok := got.Value[tc.checkKey]; !ok {
					t.Errorf("getEffectiveConfig() result missing key %q", tc.checkKey)
				}
			}
		})
	}
}

func TestGetEffectiveConfig_MergeOverride(t *testing.T) {
	r := &AerospikeClusterReconciler{}

	cluster := &ackov1alpha1.AerospikeCluster{
		Spec: ackov1alpha1.AerospikeClusterSpec{
			AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"service": map[string]any{
						"cluster-name": "test-cluster",
						"proto-fd-max": 15000,
					},
					"network": map[string]any{
						"service": map[string]any{
							"port": 3000,
						},
					},
				},
			},
		},
	}
	rack := &ackov1alpha1.Rack{
		ID: 1,
		AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{
			Value: map[string]any{
				"service": map[string]any{
					"proto-fd-max": 20000,
				},
			},
		},
	}

	got := r.getEffectiveConfig(cluster, rack)
	if got == nil {
		t.Fatal("getEffectiveConfig() = nil, want non-nil")
	}

	// Rack override should win for proto-fd-max
	svc, ok := got.Value["service"].(map[string]any)
	if !ok {
		t.Fatal("expected 'service' to be map[string]any")
	}
	if svc["proto-fd-max"] != 20000 {
		t.Errorf("proto-fd-max = %v, want 20000", svc["proto-fd-max"])
	}

	// Cluster-level cluster-name should be preserved
	if svc["cluster-name"] != "test-cluster" {
		t.Errorf("cluster-name = %v, want 'test-cluster'", svc["cluster-name"])
	}

	// Cluster-level network section should be preserved
	net, ok := got.Value["network"].(map[string]any)
	if !ok {
		t.Fatal("expected 'network' to be map[string]any")
	}
	netSvc, ok := net["service"].(map[string]any)
	if !ok {
		t.Fatal("expected 'network.service' to be map[string]any")
	}
	if netSvc["port"] != 3000 {
		t.Errorf("network.service.port = %v, want 3000", netSvc["port"])
	}
}

func TestGetEffectiveConfig_ClusterConfigOnly(t *testing.T) {
	r := &AerospikeClusterReconciler{}

	clusterConfig := &ackov1alpha1.AerospikeConfigSpec{
		Value: map[string]any{
			"service": map[string]any{
				"cluster-name": "my-cluster",
			},
		},
	}
	cluster := &ackov1alpha1.AerospikeCluster{
		Spec: ackov1alpha1.AerospikeClusterSpec{
			AerospikeConfig: clusterConfig,
		},
	}
	rack := &ackov1alpha1.Rack{ID: 0}

	got := r.getEffectiveConfig(cluster, rack)
	if got != clusterConfig {
		t.Errorf("when rack config is nil, getEffectiveConfig should return cluster config pointer directly")
	}
}

func TestGetEffectiveConfig_RackConfigOnly(t *testing.T) {
	r := &AerospikeClusterReconciler{}

	rackConfig := &ackov1alpha1.AerospikeConfigSpec{
		Value: map[string]any{
			"service": map[string]any{
				"proto-fd-max": 10000,
			},
		},
	}
	cluster := &ackov1alpha1.AerospikeCluster{
		Spec: ackov1alpha1.AerospikeClusterSpec{
			AerospikeConfig: nil,
		},
	}
	rack := &ackov1alpha1.Rack{
		ID:              0,
		AerospikeConfig: rackConfig,
	}

	got := r.getEffectiveConfig(cluster, rack)
	if got != rackConfig {
		t.Errorf("when cluster config is nil, getEffectiveConfig should return rack config pointer directly")
	}
}
