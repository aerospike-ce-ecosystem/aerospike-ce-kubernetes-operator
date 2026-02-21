package podutil

import (
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

// BuildInitContainer creates the init container that copies Aerospike
// configuration files from the ConfigMap volume to /etc/aerospike/.
func BuildInitContainer(configMapName string, volumeMounts []corev1.VolumeMount) corev1.Container {
	// Ensure the init container has mounts for both the configmap source
	// and the aerospike config destination.
	initMounts := []corev1.VolumeMount{
		{
			Name:      configMapVolumeName,
			MountPath: configMapVolumeMountPath,
			ReadOnly:  true,
		},
		{
			Name:      aerospikeConfigVolumeName,
			MountPath: aerospikeConfigPath,
		},
	}
	initMounts = append(initMounts, volumeMounts...)

	return corev1.Container{
		Name:  InitContainerName,
		Image: "busybox:1.36",
		Command: []string{
			"sh", "-c",
			"cp /configmap/* /etc/aerospike/",
		},
		VolumeMounts: initMounts,
	}
}
