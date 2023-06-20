// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
