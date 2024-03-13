// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	alicloudLoadBalancerSpecAnnotationKey = "service.beta.kubernetes.io/alibaba-cloud-loadbalancer-spec"
)

// MutateExternalTrafficPolicy mutates ServiceExternalTrafficPolicyType to Local of LoadBalancer type service
func MutateExternalTrafficPolicy(newObj, oldObj *corev1.Service) {
	if newObj.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}
	newObj.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal

	// Do not overwrite '.spec.healthCheckNodePort'
	if oldObj != nil &&
		oldObj.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal &&
		newObj.Spec.HealthCheckNodePort == 0 {
		newObj.Spec.HealthCheckNodePort = oldObj.Spec.HealthCheckNodePort
	}
}

// MutateAnnotation mutates annotation of LoadBalancer type service
func MutateAnnotation(newObj, _ *corev1.Service, loadBalancerSpec string) {
	if newObj.Spec.Type == corev1.ServiceTypeLoadBalancer {
		metav1.SetMetaDataAnnotation(&newObj.ObjectMeta, alicloudLoadBalancerSpecAnnotationKey, loadBalancerSpec)
	}
}
