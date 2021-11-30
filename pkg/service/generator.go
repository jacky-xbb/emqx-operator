package service

import (
	"fmt"
	"reflect"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NewSecretForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Secret {
	emqxEnterprise, ok := emqx.(*v1alpha2.EmqxEnterprise)
	if ok == true && emqxEnterprise.GetLicense() != "" {
		stringData := map[string]string{"emqx.lic": emqxEnterprise.GetLicense()}
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Labels:          labels,
				Name:            emqx.GetSecretName(),
				Namespace:       emqx.GetNamespace(),
				OwnerReferences: ownerRefs,
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: stringData,
		}
	} else {
		return nil
	}
}

func NewHeadLessSvcForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	ports, _, _ := generatePorts(emqx)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetHeadlessServiceName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Ports:     ports,
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone,
		},
	}
}

func NewListenerSvcForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	listener := emqx.GetListener()
	ports, _, _ := generatePorts(emqx)

	listenerSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Type:                     listener.Type,
			LoadBalancerIP:           listener.LoadBalancerIP,
			LoadBalancerSourceRanges: listener.LoadBalancerSourceRanges,
			ExternalIPs:              listener.ExternalIPs,
			Ports:                    ports,
			Selector:                 labels,
		},
	}
	return listenerSvc
}

func NewConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	acl := emqx.GetACL()
	cmForAcl := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            acl["name"],
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{"acl.conf": acl["conf"]},
	}

	return cmForAcl
}

func NewConfigMapForLoadedModules(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	modules := emqx.GetLoadedModules()
	cmForPM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            modules["name"],
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{"loaded_modules": modules["conf"]},
	}

	return cmForPM
}

func NewConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	plugins := emqx.GetLoadedPlugins()
	cmForPG := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            plugins["name"],
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{"loaded_plugins": plugins["conf"]},
	}

	return cmForPG
}

func NewEmqxStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	_, ports, env := generatePorts(emqx)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            emqx.GetName(),
			Namespace:       emqx.GetNamespace(),
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: emqx.GetHeadlessServiceName(),
			Replicas:    emqx.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			PodManagementPolicy:  appsv1.ParallelPodManagement,
			VolumeClaimTemplates: getVolumeClaimTemplates(emqx),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// TODO merge labels
					Labels: labels,
					// TODO
					// Annotations:
				},
				Spec: corev1.PodSpec{
					// TODO initContainers
					// InitContainers

					// TODO
					// Affinity: getAffinity(rc.Spec.Affinity, labels),
					ServiceAccountName: emqx.GetServiceAccountName(),
					SecurityContext:    getSecurityContext(),
					Tolerations:        emqx.GetToleRations(),
					NodeSelector:       emqx.GetNodeSelector(),
					Containers: []corev1.Container{
						{
							Name:            emqx.GetName(),
							Image:           emqx.GetImage(),
							ImagePullPolicy: getPullPolicy(emqx.GetImagePullPolicy()),
							Resources:       emqx.GetResource(),
							Env:             mergeEnv(env, emqx.GetEnv()),
							Ports:           ports,
							VolumeMounts:    getEmqxVolumeMounts(emqx),
						},
					},
					Volumes: getEmqxVolumes(emqx),
				},
			},
		},
	}

	return sts
}

func getSecurityContext() *corev1.PodSecurityContext {
	emqxUserGroup := int64(1000)
	runAsNonRoot := true
	fsGroupChangeAlways := corev1.FSGroupChangeAlways

	return &corev1.PodSecurityContext{
		FSGroup:             &emqxUserGroup,
		FSGroupChangePolicy: &fsGroupChangeAlways,
		RunAsNonRoot:        &runAsNonRoot,
		RunAsUser:           &emqxUserGroup,
		SupplementalGroups:  []int64{emqxUserGroup},
	}
}

func getVolumeClaimTemplates(emqx v1alpha2.Emqx) []corev1.PersistentVolumeClaim {
	storageSpec := emqx.GetStorage()
	if reflect.ValueOf(storageSpec).IsNil() {
		return []corev1.PersistentVolumeClaim{}
	} else {
		return []corev1.PersistentVolumeClaim{
			genVolumeClaimTemplate(emqx, emqx.GetDataVolumeName()),
			genVolumeClaimTemplate(emqx, emqx.GetLogVolumeName()),
		}
	}
}

func getEmqxVolumeMounts(emqx v1alpha2.Emqx) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      emqx.GetDataVolumeName(),
			MountPath: constants.EMQX_DATA_DIR,
		},
	)
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      emqx.GetLogVolumeName(),
			MountPath: constants.EMQX_LOG_DIR,
		},
	)
	acl := emqx.GetACL()
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      acl["name"],
			MountPath: acl["mountPath"],
			SubPath:   acl["subPath"],
		},
	)
	modules := emqx.GetLoadedModules()
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      modules["name"],
			MountPath: modules["mountPath"],
			SubPath:   modules["subPath"],
		},
	)
	plugins := emqx.GetLoadedPlugins()
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      plugins["name"],
			MountPath: plugins["mountPath"],
			SubPath:   plugins["subPath"],
		},
	)
	emqxEnterprise, ok := emqx.(*v1alpha2.EmqxEnterprise)
	if ok == true && emqxEnterprise.GetLicense() != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      emqx.GetSecretName(),
				MountPath: constants.EMQX_LIC_DIR,
				SubPath:   constants.EMQX_LIC_SUBPATH,
				ReadOnly:  true,
			},
		)
	}
	return volumeMounts
}

func getEmqxVolumes(emqx v1alpha2.Emqx) []corev1.Volume {
	volumes := []corev1.Volume{}
	storageSpec := emqx.GetStorage()
	if reflect.ValueOf(storageSpec).IsNil() {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetDataVolumeName(),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetLogVolumeName(),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}

	acl := emqx.GetACL()
	volumes = append(volumes,
		corev1.Volume{
			Name: acl["name"],
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: acl["name"],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  acl["subPath"],
							Path: acl["subPath"],
						},
					},
				},
			},
		},
	)

	plugins := emqx.GetLoadedPlugins()
	volumes = append(volumes,
		corev1.Volume{
			Name: plugins["name"],
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: plugins["name"],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  plugins["subPath"],
							Path: plugins["subPath"],
						},
					},
				},
			},
		},
	)

	modules := emqx.GetLoadedModules()
	volumes = append(volumes,
		corev1.Volume{
			Name: modules["name"],
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: modules["name"],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  modules["subPath"],
							Path: modules["subPath"],
						},
					},
				},
			},
		},
	)
	emqxEnterprise, ok := emqx.(*v1alpha2.EmqxEnterprise)
	if ok == true && emqxEnterprise.GetLicense() != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetSecretName(),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: emqx.GetSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  "emqx.lic",
								Path: "emqx.lic",
							},
						},
					},
				},
			},
		)
	}
	return volumes
}

// TODO
// func getAffinity(affinity *corev1.Affinity, labels map[string]string) *corev1.Affinity {
// 	if affinity != nil {
// 		return affinity
// 	}

// 	// Return a SOFT anti-affinity
// 	return &corev1.Affinity{
// 		PodAntiAffinity: &corev1.PodAntiAffinity{
// 			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
// 				{
// 					Weight: 100,
// 					PodAffinityTerm: corev1.PodAffinityTerm{
// 						TopologyKey: util.HostnameTopologyKey,
// 						LabelSelector: &metav1.LabelSelector{
// 							MatchLabels: labels,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// }
func getPullPolicy(specPolicy corev1.PullPolicy) corev1.PullPolicy {
	if specPolicy == "" {
		return corev1.PullIfNotPresent
	}
	return specPolicy
}

func genVolumeClaimTemplate(emqx v1alpha2.Emqx, Name string) corev1.PersistentVolumeClaim {
	template := emqx.GetStorage().VolumeClaimTemplate
	boolTrue := true
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: template.APIVersion,
			Kind:       template.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        Name,
			Namespace:   emqx.GetNamespace(),
			Annotations: emqx.GetAnnotations(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         emqx.GetAPIVersion(),
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               emqx.GetKind(),
					Name:               emqx.GetName(),
					UID:                emqx.GetUID(),
				},
			},
		},
		Spec:   template.Spec,
		Status: template.Status,
	}
	if pvc.Spec.AccessModes == nil {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	return pvc
}

func generatePorts(emqx v1alpha2.Emqx) ([]corev1.ServicePort, []corev1.ContainerPort, []corev1.EnvVar) {
	var servicePorts []corev1.ServicePort
	var containerPorts []corev1.ContainerPort
	var env []corev1.EnvVar
	listener := emqx.GetListener()
	if !reflect.ValueOf(listener.Ports.MQTT).IsZero() {
		env = append(env, corev1.EnvVar{
			Name:  "EMQX_LISTENER__TCP__EXTERNAL",
			Value: fmt.Sprint(listener.Ports.MQTT),
		})
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "mqtt",
			ContainerPort: listener.Ports.MQTT,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "mqtt",
			Port:     listener.Ports.MQTT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: listener.Ports.MQTT,
			},
		})
	}
	if !reflect.ValueOf(listener.Ports.MQTTS).IsZero() {
		env = append(env, corev1.EnvVar{
			Name:  "EMQX_LISTENER__SSL__EXTERNAL",
			Value: fmt.Sprint(listener.Ports.MQTTS),
		})
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "mqtts",
			ContainerPort: listener.Ports.MQTTS,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "mqtts",
			Port:     listener.Ports.MQTTS,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: listener.Ports.MQTTS,
			},
		})
	}
	if !reflect.ValueOf(listener.Ports.WS).IsZero() {
		env = append(env, corev1.EnvVar{
			Name:  "EMQX_LISTENER__WS__EXTERNAL",
			Value: fmt.Sprint(listener.Ports.WS),
		})
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "ws",
			ContainerPort: listener.Ports.WS,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "ws",
			Port:     listener.Ports.WS,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: listener.Ports.WS,
			},
		})
	}
	if !reflect.ValueOf(listener.Ports.WSS).IsZero() {
		env = append(env, corev1.EnvVar{
			Name:  "EMQX_LISTENER__WSS__EXTERNAL",
			Value: fmt.Sprint(listener.Ports.WSS),
		})
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "wss",
			ContainerPort: listener.Ports.WSS,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "wss",
			Port:     listener.Ports.WSS,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: listener.Ports.WSS,
			},
		})
	}
	if !reflect.ValueOf(listener.Ports.Dashboard).IsZero() {
		env = append(env, corev1.EnvVar{
			Name:  "EMQX_DASHBOARD__LISTENER__HTTP",
			Value: fmt.Sprint(listener.Ports.Dashboard),
		})
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "dashboard",
			ContainerPort: listener.Ports.Dashboard,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "dashboard",
			Port:     listener.Ports.Dashboard,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: listener.Ports.Dashboard,
			},
		})
	}
	if !reflect.ValueOf(listener.Ports.API).IsZero() {
		env = append(env, corev1.EnvVar{
			Name:  "EMQX_MANAGEMENT__LISTENER__HTTP",
			Value: fmt.Sprint(listener.Ports.API),
		})
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "api",
			ContainerPort: listener.Ports.API,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "api",
			Port:     listener.Ports.API,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: listener.Ports.API,
			},
		})
	}
	return servicePorts, containerPorts, env
}

func contains(Env []corev1.EnvVar, Name string) int {
	for index, value := range Env {
		if value.Name == Name {
			return index
		}
	}
	return -1
}

func mergeEnv(env1, env2 []corev1.EnvVar) []corev1.EnvVar {
	for _, value := range env2 {
		r := contains(env1, value.Name)
		if r == -1 {
			env1 = append(env1, value)
		}
	}
	return env1
}