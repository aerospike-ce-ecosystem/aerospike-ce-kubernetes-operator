package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

// BuildVolumes converts the AerospikeStorageSpec into Kubernetes Volumes and
// VolumeMounts for the Aerospike container.
func BuildVolumes(storageSpec *v1alpha1.AerospikeStorageSpec) ([]corev1.Volume, []corev1.VolumeMount) {
	if storageSpec == nil {
		return nil, nil
	}

	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount

	for i := range storageSpec.Volumes {
		vol := &storageSpec.Volumes[i]

		k8sVol := volumeForSpec(vol)
		if k8sVol != nil {
			volumes = append(volumes, *k8sVol)
		}

		if vol.Aerospike != nil {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      vol.Name,
				MountPath: vol.Aerospike.Path,
			})
		}
	}

	return volumes, mounts
}

// volumeForSpec creates a Kubernetes Volume from a VolumeSpec.
// For PersistentVolume sources, no Volume is generated because the
// StatefulSet uses volumeClaimTemplates instead.
func volumeForSpec(vol *v1alpha1.VolumeSpec) *corev1.Volume {
	src := &vol.Source

	switch {
	case src.EmptyDir != nil:
		return &corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: src.EmptyDir,
			},
		}
	case src.Secret != nil:
		return &corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: src.Secret,
			},
		}
	case src.ConfigMap != nil:
		return &corev1.Volume{
			Name: vol.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: src.ConfigMap,
			},
		}
	case src.PersistentVolume != nil:
		// PVC-backed volumes are handled via volumeClaimTemplates, not inline volumes.
		return nil
	}

	return nil
}

// BuildVolumeClaimTemplates generates PersistentVolumeClaim templates for a
// StatefulSet from volumes that use a PersistentVolume source.
func BuildVolumeClaimTemplates(storageSpec *v1alpha1.AerospikeStorageSpec) []corev1.PersistentVolumeClaim {
	if storageSpec == nil {
		return nil
	}

	var claims []corev1.PersistentVolumeClaim

	for i := range storageSpec.Volumes {
		vol := &storageSpec.Volumes[i]
		pv := vol.Source.PersistentVolume
		if pv == nil {
			continue
		}

		accessModes := pv.AccessModes
		if len(accessModes) == 0 {
			accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		}

		volumeMode := pv.VolumeMode
		if volumeMode == "" {
			volumeMode = corev1.PersistentVolumeFilesystem
		}

		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: vol.Name,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: accessModes,
				VolumeMode:  &volumeMode,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(pv.Size),
					},
				},
			},
		}

		if pv.StorageClass != "" {
			pvc.Spec.StorageClassName = &pv.StorageClass
		}

		if pv.Selector != nil {
			pvc.Spec.Selector = pv.Selector
		}

		claims = append(claims, pvc)
	}

	return claims
}

// VolumeMountsForContainer returns the volume mounts that target the given
// container name from the Sidecars or InitContainers attachment lists.
func VolumeMountsForContainer(storageSpec *v1alpha1.AerospikeStorageSpec, containerName string, isSidecar bool) []corev1.VolumeMount {
	if storageSpec == nil {
		return nil
	}

	var mounts []corev1.VolumeMount

	for i := range storageSpec.Volumes {
		vol := &storageSpec.Volumes[i]

		var attachments []v1alpha1.VolumeAttachment
		if isSidecar {
			attachments = vol.Sidecars
		} else {
			attachments = vol.InitContainers
		}

		for _, att := range attachments {
			if att.ContainerName == containerName {
				mounts = append(mounts, corev1.VolumeMount{
					Name:      vol.Name,
					MountPath: att.Path,
				})
			}
		}
	}

	return mounts
}
