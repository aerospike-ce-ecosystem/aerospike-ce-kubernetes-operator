package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
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
) error {
	log := logf.FromContext(ctx)

	cmName := utils.ConfigMapName(cluster.Name, rack.ID)

	// Build effective config for this rack
	effectiveConfig := r.getEffectiveConfig(cluster, rack)
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

	// Generate aerospike.conf
	confText, err := configgen.GenerateConfig(effectiveConfig.Value)
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
		if err := ctrl.SetControllerReference(cluster, cm, r.Scheme); err != nil {
			return fmt.Errorf("setting controller reference: %w", err)
		}
		log.Info("Creating ConfigMap", "name", cmName)
		return r.Create(ctx, cm)
	} else if err != nil {
		return fmt.Errorf("getting ConfigMap %s: %w", cmName, err)
	}

	// Update if data changed
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
