// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControlPlaneConfig contains configuration settings for the control plane.
type ControlPlaneConfig struct {
	metav1.TypeMeta

	// CloudControllerManager contains configuration settings for the cloud-controller-manager.
	CloudControllerManager *CloudControllerManagerConfig

	// CSI is the config for CSI plugin components.
	CSI *CSI

	// Storage contains configuration for the default StorageClass and VolumeSnapshotClass.
	// +optional
	Storage *Storage
}

// CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.
type CloudControllerManagerConfig struct {
	// FeatureGates contains information about enabled feature gates.
	FeatureGates map[string]bool
}

// CSI is csi components configuration.
type CSI struct {
	// EnableADController enables disks to be attached/detached from controller server of CSI Plugin.
	EnableADController *bool
}

// Storage contains configuration for the default StorageClass and VolumeSnapshotClass.
type Storage struct {
	// ManagedDefaultStorageClass controls if the 'default' StorageClass is marked as default.
	// Defaults to true.
	ManagedDefaultStorageClass *bool
	// ManagedDefaultVolumeSnapshotClass controls if the 'default' VolumeSnapshotClass is marked as default.
	// Defaults to true.
	ManagedDefaultVolumeSnapshotClass *bool
}
