package controller

import (
	"context"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/configgen"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/initcontainer"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

func (r *AerospikeCEClusterReconciler) reconcileConfigMap(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	rack *asdbcev1alpha1.Rack,
	effectiveConfig *asdbcev1alpha1.AerospikeConfigSpec,
) error {
	log := logf.FromContext(ctx)

	cmName := utils.ConfigMapName(cluster.Name, rack.ID)

	if effectiveConfig == nil {
		// Provide a minimal default config
		effectiveConfig = &asdbcev1alpha1.AerospikeConfigSpec{
			Value: map[string]any{
				"service": map[string]any{
					"cluster-name": cluster.Name,
				},
				"network": map[string]any{
					"service": map[string]any{
						"address": "any",
						"port":    3000,
					},
					"heartbeat": map[string]any{
						"mode": "mesh",
						"port": 3002,
					},
					"fabric": map[string]any{
						"address": "any",
						"port":    3001,
					},
				},
			},
		}
	}

	// Inject access-address placeholders based on network policy
	configgen.InjectAccessAddressPlaceholders(effectiveConfig.Value, cluster.Spec.AerospikeNetworkPolicy)

	// Collect all pod names across all racks for mesh seed injection
	racks := r.getRacks(cluster)
	totalPods := int32(0)
	for rackIdx := range racks {
		totalPods += r.getRackSize(cluster, racks, rackIdx)
	}
	allPodNames := make([]string, 0, totalPods)
	for rackIdx, rk := range racks {
		rackSize := r.getRackSize(cluster, racks, rackIdx)
		stsName := utils.StatefulSetName(cluster.Name, rk.ID)
		for i := range rackSize {
			allPodNames = append(allPodNames, fmt.Sprintf("%s-%d", stsName, i))
		}
	}

	// Determine heartbeat port
	heartbeatPort := 3002
	if netCfg, ok := effectiveConfig.Value["network"].(map[string]any); ok {
		if hbCfg, ok := netCfg["heartbeat"].(map[string]any); ok {
			if port, ok := hbCfg["port"]; ok {
				heartbeatPort = utils.IntFromAny(port, heartbeatPort)
			}
		}
	}

	serviceName := utils.HeadlessServiceName(cluster.Name)

	// Generate aerospike.conf with mesh seeds injected
	confText, err := configgen.GenerateConfForPod(
		effectiveConfig.Value,
		"", // podName not used for shared ConfigMap
		serviceName,
		cluster.Namespace,
		allPodNames,
		heartbeatPort,
	)
	if err != nil {
		return fmt.Errorf("generating aerospike.conf: %w", err)
	}

	// Build ConfigMap data
	data := initcontainer.GetConfigMapData(confText)

	labels := utils.LabelsForRack(cluster.Name, rack.ID)

	// Check if ConfigMap exists
	existing := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: cluster.Namespace}, existing)

	if errors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: cluster.Namespace,
				Labels:    labels,
			},
			Data: data,
		}
		if err := r.setOwnerRef(cluster, cm); err != nil {
			return err
		}
		log.Info("Creating ConfigMap", "name", cmName)
		return r.Create(ctx, cm)
	} else if err != nil {
		return fmt.Errorf("getting ConfigMap %s: %w", cmName, err)
	}

	// Update only if data or labels changed
	if maps.Equal(existing.Data, data) && maps.Equal(existing.Labels, labels) {
		return nil
	}
	existing.Data = data
	existing.Labels = labels
	log.Info("Updating ConfigMap", "name", cmName)
	return r.Update(ctx, existing)
}

// getEffectiveConfig returns the merged config for a rack.
func (r *AerospikeCEClusterReconciler) getEffectiveConfig(
	cluster *asdbcev1alpha1.AerospikeCECluster,
	rack *asdbcev1alpha1.Rack,
) *asdbcev1alpha1.AerospikeConfigSpec {
	if cluster.Spec.AerospikeConfig == nil {
		if rack.AerospikeConfig != nil {
			return rack.AerospikeConfig
		}
		return nil
	}

	if rack.AerospikeConfig == nil {
		return cluster.Spec.AerospikeConfig
	}

	merged := utils.DeepMerge(
		cluster.Spec.AerospikeConfig.Value,
		rack.AerospikeConfig.Value,
	)
	return &asdbcev1alpha1.AerospikeConfigSpec{Value: merged}
}
