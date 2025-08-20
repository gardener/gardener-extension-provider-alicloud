// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	apisconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the Alicloud provider.
type ControllerConfiguration struct {
	metav1.TypeMeta

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection *configv1alpha1.ClientConnectionConfiguration
	// MachineImageOwnerSecretRef is the secret reference which contains credential of AliCloud subaccount for customized images.
	// We currently assume multiple customized images should always be under this account.
	MachineImageOwnerSecretRef *corev1.SecretReference
	// ToBeSharedImageIDs specifies custom image IDs which need to be shared by shoots
	ToBeSharedImageIDs []string
	// Service is the service configuration
	Service Service
	// ETCD is the etcd configuration.
	ETCD ETCD
	// HealthCheckConfig is the config for the health check controller
	HealthCheckConfig *apisconfigv1alpha1.HealthCheckConfig
	// CSI is the config for CSI plugin components
	CSI *CSI
}

// Service is a load balancer service configuration.
type Service struct {
	// BackendLoadBalancerSpec specifies the type of backend Alicloud load balancer, default is slb.s1.small.
	BackendLoadBalancerSpec string
}

// ETCD is an etcd configuration.
type ETCD struct {
	// ETCDStorage is the etcd storage configuration.
	Storage ETCDStorage
	// ETCDBackup is the etcd backup configuration.
	Backup ETCDBackup
}

// ETCDStorage is an etcd storage configuration.
type ETCDStorage struct {
	// ClassName is the name of the storage class used in etcd-main volume claims.
	ClassName *string
	// Capacity is the storage capacity used in etcd-main volume claims.
	Capacity *resource.Quantity
}

// ETCDBackup is an etcd backup configuration.
type ETCDBackup struct {
	// Schedule is the etcd backup schedule.
	Schedule *string
}

// CSI is csi components configuration.
type CSI struct {
	// EnableADController enables disks to be attached/detached from csi-attacher
	// Deprecated
	EnableADController *bool
}
