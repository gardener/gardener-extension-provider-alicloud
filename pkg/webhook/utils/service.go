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
func MutateExternalTrafficPolicy(new, old *corev1.Service) {
	if new.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}
	new.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal

	// Do not overwrite '.spec.healthCheckNodePort'
	if old != nil &&
		old.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal &&
		new.Spec.HealthCheckNodePort == 0 {
		new.Spec.HealthCheckNodePort = old.Spec.HealthCheckNodePort
	}
}

// MutateAnnotation mutates annotation of LoadBalancer type service
func MutateAnnotation(new, old *corev1.Service, loadBalancerSpec string) {
	if new.Spec.Type == corev1.ServiceTypeLoadBalancer {
		metav1.SetMetaDataAnnotation(&new.ObjectMeta, alicloudLoadBalancerSpecAnnotationKey, loadBalancerSpec)
	}
}
