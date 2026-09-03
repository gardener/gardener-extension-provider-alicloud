// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_ControlPlaneConfig sets default values for ControlPlaneConfig.
func SetDefaults_ControlPlaneConfig(obj *ControlPlaneConfig) {
	if obj.Storage == nil {
		obj.Storage = &Storage{}
	}
	if obj.Storage.ManagedDefaultStorageClass == nil {
		obj.Storage.ManagedDefaultStorageClass = ptr.To(true)
	}
	if obj.Storage.ManagedDefaultVolumeSnapshotClass == nil {
		obj.Storage.ManagedDefaultVolumeSnapshotClass = ptr.To(true)
	}
}
