// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
// resource.
type CloudProfileConfig struct {
	metav1.TypeMeta
	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to provider-specific identifiers.
	MachineImages []MachineImages
}

// MachineImages is a mapping from logical names and versions to provider-specific identifiers.
type MachineImages struct {
	// Name is the logical name of the machine image.
	Name string
	// Versions contains versions and a provider-specific identifier.
	Versions []MachineImageVersion
}

// MachineImageVersion contains a version and a provider-specific identifier.
type MachineImageVersion struct {
	// Version is the version of the image.
	Version string
	// TODO add "// deprecated" once cloudprofiles are migrated to use CapabilityFlavors
	// Regions is a mapping to the correct ID for the machine image in the supported regions.
	Regions []RegionIDMapping
	// CapabilityFlavors is grouping of region machine image IDs by capabilities.
	CapabilityFlavors []MachineImageFlavor
}

// MachineImageFlavor groups all RegionIDMappings for a specific set of capabilities.
type MachineImageFlavor struct {
	// Regions is a mapping to the correct ID for the machine image in the supported regions.
	Regions []RegionIDMapping
	// Capabilities that are supported by the machine image IDs in this set.
	Capabilities gardencorev1beta1.Capabilities
}

// GetCapabilities returns the Capabilities of a MachineImageFlavor
func (cs *MachineImageFlavor) GetCapabilities() gardencorev1beta1.Capabilities {
	return cs.Capabilities
}

// RegionIDMapping is a mapping to the correct ID for the machine image in the given region.
type RegionIDMapping struct {
	// Name is the name of the region.
	Name string
	// ID is the id of the image.
	ID string
}
