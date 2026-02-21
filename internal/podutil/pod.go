package podutil

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/storage"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

const (
	configMapVolumeName        = "aerospike-config-map"
	aerospikeConfigVolumeName  = "aerospike-config"
	defaultTerminationGraceSec = int64(30)
)

// BuildPodTemplateSpec builds the complete PodTemplateSpec for a StatefulSet
// managing Aerospike pods in the given rack.
func BuildPodTemplateSpec(
	cluster *v1alpha1.AerospikeCECluster,
	rack *v1alpha1.Rack,
	rackID int,
	configMapName string,
	configHash string,
) corev1.PodTemplateSpec {
	// Determine the effective storage spec (rack override or cluster-level).
	storageSpec := cluster.Spec.Storage
	if rack != nil && rack.Storage != nil {
		storageSpec = rack.Storage
	}

	// Build volumes and mounts from storage spec.
	volumes, aerospikeMounts := storage.BuildVolumes(storageSpec)

	// Add the configmap volume (source) and the writable aerospike-config volume.
	volumes = append(volumes,
		corev1.Volume{
			Name: configMapVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		},
		corev1.Volume{
			Name: aerospikeConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)

	// Ensure the aerospike container has the config volume mounted.
	aerospikeMounts = append(aerospikeMounts, corev1.VolumeMount{
		Name:      aerospikeConfigVolumeName,
		MountPath: aerospikeConfigPath,
	})

	// Labels.
	labels := utils.LabelsForRack(cluster.Name, rackID)

	// Annotations.
	annotations := map[string]string{
		utils.ConfigHashAnnotation: configHash,
	}

	// Merge user-provided pod metadata.
	if cluster.Spec.PodSpec != nil && cluster.Spec.PodSpec.Metadata != nil {
		for k, v := range cluster.Spec.PodSpec.Metadata.Labels {
			labels[k] = v
		}
		for k, v := range cluster.Spec.PodSpec.Metadata.Annotations {
			annotations[k] = v
		}
	}

	// Build containers.
	initContainer := BuildInitContainer(configMapName, nil)
	aerospikeContainer := BuildAerospikeContainer(cluster, aerospikeMounts)

	// Init containers: operator init + user-defined.
	initContainers := []corev1.Container{initContainer}

	// Sidecars.
	var sidecars []corev1.Container

	if cluster.Spec.PodSpec != nil {
		for _, ic := range cluster.Spec.PodSpec.InitContainers {
			mounts := storage.VolumeMountsForContainer(storageSpec, ic.Name, false)
			ic.VolumeMounts = append(ic.VolumeMounts, mounts...)
			initContainers = append(initContainers, ic)
		}

		for _, sc := range cluster.Spec.PodSpec.Sidecars {
			mounts := storage.VolumeMountsForContainer(storageSpec, sc.Name, true)
			sc.VolumeMounts = append(sc.VolumeMounts, mounts...)
			sidecars = append(sidecars, sc)
		}
	}

	allContainers := []corev1.Container{aerospikeContainer}
	allContainers = append(allContainers, sidecars...)

	// Termination grace period.
	terminationGrace := defaultTerminationGraceSec
	if cluster.Spec.PodSpec != nil && cluster.Spec.PodSpec.TerminationGracePeriodSeconds != nil {
		terminationGrace = *cluster.Spec.PodSpec.TerminationGracePeriodSeconds
	}

	podSpec := corev1.PodSpec{
		InitContainers:                initContainers,
		Containers:                    allContainers,
		Volumes:                       volumes,
		TerminationGracePeriodSeconds: &terminationGrace,
	}

	// Pod-level settings from cluster spec.
	if cluster.Spec.PodSpec != nil {
		applyPodSpecSettings(&podSpec, cluster.Spec.PodSpec)
	}

	// Rack-level overrides.
	if rack != nil && rack.PodSpec != nil {
		applyRackPodSpecOverrides(&podSpec, rack.PodSpec, rackID)
	} else if rack != nil {
		// Apply zone/region/node affinity from rack definition.
		applyRackAffinity(&podSpec, rack)
	}

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: podSpec,
	}
}

// applyPodSpecSettings applies cluster-level pod spec settings.
func applyPodSpecSettings(podSpec *corev1.PodSpec, spec *v1alpha1.AerospikeCEPodSpec) {
	if spec.Affinity != nil {
		podSpec.Affinity = spec.Affinity
	}

	if len(spec.Tolerations) > 0 {
		podSpec.Tolerations = spec.Tolerations
	}

	if len(spec.NodeSelector) > 0 {
		podSpec.NodeSelector = spec.NodeSelector
	}

	if spec.SecurityContext != nil {
		podSpec.SecurityContext = spec.SecurityContext
	}

	if spec.ServiceAccountName != "" {
		podSpec.ServiceAccountName = spec.ServiceAccountName
	}

	if spec.DNSPolicy != "" {
		podSpec.DNSPolicy = spec.DNSPolicy
	}

	if spec.HostNetwork {
		podSpec.HostNetwork = true
	}

	if len(spec.ImagePullSecrets) > 0 {
		podSpec.ImagePullSecrets = spec.ImagePullSecrets
	}
}

// applyRackPodSpecOverrides overrides pod spec settings with rack-specific values.
func applyRackPodSpecOverrides(podSpec *corev1.PodSpec, rackPod *v1alpha1.RackPodSpec, rackID int) {
	if rackPod.Affinity != nil {
		podSpec.Affinity = rackPod.Affinity
	}

	if len(rackPod.Tolerations) > 0 {
		podSpec.Tolerations = rackPod.Tolerations
	}

	if len(rackPod.NodeSelector) > 0 {
		podSpec.NodeSelector = rackPod.NodeSelector
	}
}

// applyRackAffinity sets node affinity based on rack zone/region/nodeName fields.
func applyRackAffinity(podSpec *corev1.PodSpec, rack *v1alpha1.Rack) {
	var terms []corev1.NodeSelectorRequirement

	if rack.Zone != "" {
		terms = append(terms, corev1.NodeSelectorRequirement{
			Key:      "topology.kubernetes.io/zone",
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{rack.Zone},
		})
	}

	if rack.Region != "" {
		terms = append(terms, corev1.NodeSelectorRequirement{
			Key:      "topology.kubernetes.io/region",
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{rack.Region},
		})
	}

	if rack.NodeName != "" {
		terms = append(terms, corev1.NodeSelectorRequirement{
			Key:      "kubernetes.io/hostname",
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{rack.NodeName},
		})
	}

	if len(terms) == 0 {
		return
	}

	if podSpec.Affinity == nil {
		podSpec.Affinity = &corev1.Affinity{}
	}

	podSpec.Affinity.NodeAffinity = &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{MatchExpressions: terms},
			},
		},
	}
}

// PodNameForIndex returns the pod name for a given StatefulSet and ordinal index.
func PodNameForIndex(stsName string, index int) string {
	return fmt.Sprintf("%s-%d", stsName, index)
}
