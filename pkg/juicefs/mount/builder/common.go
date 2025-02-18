/*
Copyright 2022 Juicedata Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"fmt"
	"github.com/juicedata/juicefs-csi-driver/pkg/config"
	corev1 "k8s.io/api/core/v1"
)

type Builder struct {
	jfsSetting *config.JfsSetting
}

func NewBuilder(setting *config.JfsSetting) *Builder {
	return &Builder{setting}
}

func (r *Builder) generateJuicePod() *corev1.Pod {
	pod := config.GeneratePodTemplate()

	volumes := r.getVolumes()
	volumeMounts := r.getVolumeMounts()
	i := 1
	for k, v := range r.jfsSetting.Configs {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      fmt.Sprintf("config-%v", i),
			MountPath: v,
		})
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("config-%v", i),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: k,
				},
			},
		})
		i++
	}

	pod.Spec.Volumes = volumes
	pod.Spec.Containers[0].VolumeMounts = volumeMounts
	pod.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: r.jfsSetting.SecretName,
			},
		},
	}}
	if r.jfsSetting.FormatCmd != "" {
		initContainer := r.getInitContainer()
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, volumeMounts...)
		pod.Spec.InitContainers = []corev1.Container{initContainer}
	}
	return pod
}

func (r *Builder) getVolumes() []corev1.Volume {
	dir := corev1.HostPathDirectoryOrCreate
	secretName := r.jfsSetting.SecretName
	volumes := []corev1.Volume{{
		Name: "jfs-dir",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: config.MountPointPath,
				Type: &dir,
			},
		}}, {
		Name: "jfs-root-dir",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: config.JFSConfigPath,
				Type: &dir,
			},
		},
	}}
	if r.jfsSetting.EncryptRsaKey != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "rsa-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{{
						Key:  "encrypt_rsa_key",
						Path: "rsa-key.pem",
					}},
				},
			},
		})
	}
	if r.jfsSetting.InitConfig != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "init_config",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Items: []corev1.KeyToPath{{
					Key:  "init_config",
					Path: r.jfsSetting.Name + ".conf",
				}},
			}},
		})
	}
	return volumes
}

func (r *Builder) getVolumeMounts() []corev1.VolumeMount {
	mp := corev1.MountPropagationBidirectional
	volumeMounts := []corev1.VolumeMount{{
		Name:             "jfs-dir",
		MountPath:        config.PodMountBase,
		MountPropagation: &mp,
	}, {
		Name:             "jfs-root-dir",
		MountPath:        "/root/.juicefs",
		MountPropagation: &mp,
	}}
	if r.jfsSetting.EncryptRsaKey != "" {
		if !r.jfsSetting.IsCe {
			volumeMounts = append(volumeMounts,
				corev1.VolumeMount{
					Name:      "rsa-key",
					MountPath: "/root/.rsa",
				},
			)
		}
	}
	if r.jfsSetting.InitConfig != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      "init_config",
				MountPath: "/root/.juicefs",
			},
		)
	}
	return volumeMounts
}
