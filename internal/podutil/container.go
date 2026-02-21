package podutil

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

const (
	AerospikeContainerName = "aerospike-server"
	InitContainerName      = "aerospike-init"

	ServicePort   int32 = 3000
	FabricPort    int32 = 3001
	HeartbeatPort int32 = 3002
	InfoPort      int32 = 3003

	configMapVolumeMountPath = "/configmap"
	aerospikeConfigPath      = "/etc/aerospike"
)

// BuildAerospikeContainer creates the main Aerospike server container spec.
func BuildAerospikeContainer(cluster *v1alpha1.AerospikeCECluster, volumeMounts []corev1.VolumeMount) corev1.Container {
	c := corev1.Container{
		Name:  AerospikeContainerName,
		Image: cluster.Spec.Image,
		Command: []string{
			"/usr/bin/asd",
			"--foreground",
		},
		Ports: []corev1.ContainerPort{
			{Name: "service", ContainerPort: ServicePort, Protocol: corev1.ProtocolTCP},
			{Name: "fabric", ContainerPort: FabricPort, Protocol: corev1.ProtocolTCP},
			{Name: "heartbeat", ContainerPort: HeartbeatPort, Protocol: corev1.ProtocolTCP},
			{Name: "info", ContainerPort: InfoPort, Protocol: corev1.ProtocolTCP},
		},
		VolumeMounts: volumeMounts,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(ServicePort),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			FailureThreshold:    3,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(ServicePort),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       30,
			TimeoutSeconds:      5,
			FailureThreshold:    3,
		},
	}

	if cluster.Spec.PodSpec != nil && cluster.Spec.PodSpec.AerospikeContainerSpec != nil {
		spec := cluster.Spec.PodSpec.AerospikeContainerSpec
		if spec.Resources != nil {
			c.Resources = *spec.Resources
		}
		if spec.SecurityContext != nil {
			c.SecurityContext = spec.SecurityContext
		}
	}

	return c
}

// BuildInitContainer creates the init container that runs the aerospike-init.sh script
// to copy and process configuration files (placeholder substitution, volume initialization).
func BuildInitContainer(
	cluster *v1alpha1.AerospikeCECluster,
	configMapName string,
	storageSpec *v1alpha1.AerospikeStorageSpec,
	volumeMounts []corev1.VolumeMount,
) corev1.Container {
	// Ensure the init container has mounts for both the configmap source
	// and the aerospike config destination.
	initMounts := make([]corev1.VolumeMount, 0, 2+len(volumeMounts))
	initMounts = append(initMounts,
		corev1.VolumeMount{
			Name:      configMapVolumeName,
			MountPath: configMapVolumeMountPath,
			ReadOnly:  true,
		},
		corev1.VolumeMount{
			Name:      aerospikeConfigVolumeName,
			MountPath: aerospikeConfigPath,
		},
	)
	initMounts = append(initMounts, volumeMounts...)

	env := buildInitEnvVars()
	if initVolumes := buildInitVolumesEnv(storageSpec); initVolumes != "" {
		env = append(env, corev1.EnvVar{
			Name:  "INIT_VOLUMES",
			Value: initVolumes,
		})
	}

	return corev1.Container{
		Name:  InitContainerName,
		Image: cluster.Spec.Image,
		Command: []string{
			"bash", "/configmap/aerospike-init.sh",
		},
		Env:          env,
		VolumeMounts: initMounts,
	}
}

// buildInitEnvVars returns the Downward API environment variables for the init container.
func buildInitEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "NODE_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
	}
}

// buildInitVolumesEnv converts storage spec InitMethod fields into an INIT_VOLUMES
// environment variable value. Format: "method1:path1,method2:path2"
func buildInitVolumesEnv(storageSpec *v1alpha1.AerospikeStorageSpec) string {
	if storageSpec == nil {
		return ""
	}

	var parts []string
	for i := range storageSpec.Volumes {
		vol := &storageSpec.Volumes[i]
		if vol.InitMethod == "" || vol.InitMethod == v1alpha1.VolumeInitMethodNone {
			continue
		}
		if vol.Aerospike == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%s", vol.InitMethod, vol.Aerospike.Path))
	}

	return strings.Join(parts, ",")
}
