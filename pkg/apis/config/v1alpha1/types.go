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

package v1alpha1

import (
	healthcheckconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the Alicloud provider.
type ControllerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	// +optional
	ClientConnection *componentbaseconfigv1alpha1.ClientConnectionConfiguration `json:"clientConnection,omitempty"`
	// MachineImageOwnerSecretRef is the secret reference which contains credential of AliCloud subaccount for customized images.
	// We currently assume multiple customized images should always be under this account.
	MachineImageOwnerSecretRef *corev1.SecretReference `json:"machineImageOwnerSecretRef,omitempty"`
	// ToBeSharedImageIDs specifies custom image IDs which need to be shared by shoots
	ToBeSharedImageIDs []string `json:"toBeSharedImageIDs,omitempty"`
	// Service is the service configuration
	Service Service `json:"service"`
	// ETCD is the etcd configuration.
	ETCD ETCD `json:"etcd"`
	// HealthCheckConfig is the config for the health check controller
	// +optional
	HealthCheckConfig *healthcheckconfigv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`
	// CSI is the config for CSI plugin components
	// +optional
	CSI *CSI `json:"csi,omitempty"`
}

// Service is a load balancer service configuration.
type Service struct {
	// BackendLoadBalancerSpec specifies the type of backend Alicloud load balancer, default is slb.s1.small.
	BackendLoadBalancerSpec string `json:"backendLoadBalancerSpec"`
}

// ETCD is an etcd configuration.
type ETCD struct {
	// ETCDStorage is the etcd storage configuration.
	Storage ETCDStorage `json:"storage"`
	// ETCDBackup is the etcd backup configuration.
	Backup ETCDBackup `json:"backup"`
}

// ETCDStorage is an etcd storage configuration.
type ETCDStorage struct {
	// ClassName is the name of the storage class used in etcd-main volume claims.
	// +optional
	ClassName *string `json:"className,omitempty"`
	// Capacity is the storage capacity used in etcd-main volume claims.
	// +optional
	Capacity *resource.Quantity `json:"capacity,omitempty"`
}

// ETCDBackup is an etcd backup configuration.
type ETCDBackup struct {
	// Schedule is the etcd backup schedule.
	// +optional
	Schedule *string `json:"schedule,omitempty"`
}

// CSI is csi components configuration.
type CSI struct {
	// EnableADController enables disks to be attached/detached from csi-provisioner
	// +optional
	// Deprecated
	EnableADController *bool `json:"enableADController,omitempty"`
}
