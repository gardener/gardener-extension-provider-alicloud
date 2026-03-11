// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"maps"
	"slices"

	gardencoreapi "github.com/gardener/gardener/pkg/api"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/utils"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// ValidateCloudProfileConfig validates a CloudProfileConfig object.
func ValidateCloudProfileConfig(cpConfig *apisalicloud.CloudProfileConfig, machineImages []core.MachineImage, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	machineImagesPath := fldPath.Child("machineImages")

	// Validate machine images section
	allErrs = append(allErrs, validateMachineImages(cpConfig.MachineImages, capabilityDefinitions, machineImagesPath)...)

	// Validate machine image mappings
	allErrs = append(allErrs, validateMachineImageMapping(machineImages, cpConfig, capabilityDefinitions, field.NewPath("spec").Child("machineImages"))...)

	return allErrs
}

// validateMachineImages validates the machine images section of CloudProfileConfig
func validateMachineImages(machineImages []apisalicloud.MachineImages, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Ensure at least one machine image is provided
	if len(machineImages) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "must provide at least one machine image"))
		return allErrs
	}

	// Validate each machine image
	for i, machineImage := range machineImages {
		idxPath := fldPath.Index(i)
		allErrs = append(allErrs, ValidateProviderMachineImage(machineImage, capabilityDefinitions, idxPath)...)
	}

	return allErrs
}

// ValidateProviderMachineImage validates a CloudProfileConfig MachineImages entry.
func ValidateProviderMachineImage(providerImage apisalicloud.MachineImages, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, validationPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(providerImage.Name) == 0 {
		allErrs = append(allErrs, field.Required(validationPath.Child("name"), "must provide a name"))
	}

	if len(providerImage.Versions) == 0 {
		allErrs = append(allErrs, field.Required(validationPath.Child("versions"), fmt.Sprintf("must provide at least one version for machine image %q", providerImage.Name)))
	}

	// Validate each version
	for j, version := range providerImage.Versions {
		jdxPath := validationPath.Child("versions").Index(j)
		allErrs = append(allErrs, validateMachineImageVersion(providerImage, capabilityDefinitions, version, jdxPath)...)
	}

	return allErrs
}

// validateMachineImageVersion validates a specific machine image version
func validateMachineImageVersion(providerImage apisalicloud.MachineImages, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, version apisalicloud.MachineImageVersion, jdxPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(version.Version) == 0 {
		allErrs = append(allErrs, field.Required(jdxPath.Child("version"), "must provide a version"))
	}

	hasRegions := len(version.Regions) > 0
	hasCapabilityFlavors := len(version.CapabilityFlavors) > 0

	if len(capabilityDefinitions) > 0 {
		// When CloudProfile defines capabilities, allow either old format (regions) or new format (capabilityFlavors) per version
		if hasRegions && hasCapabilityFlavors {
			allErrs = append(allErrs, field.Forbidden(jdxPath.Child("regions"), "must not be set together with capabilityFlavors. Use one format per version."))
		} else if hasCapabilityFlavors {
			allErrs = append(allErrs, validateCapabilityFlavors(providerImage, version, capabilityDefinitions, jdxPath)...)
		} else {
			// Using old format with regions - validate regions without architecture restriction (architecture is validated separately)
			allErrs = append(allErrs, validateRegionsWithCapabilities(version.Regions, providerImage.Name, version.Version, jdxPath)...)
		}
	} else {
		// Without capabilities, only old format with regions is supported
		allErrs = append(allErrs, validateRegions(version.Regions, providerImage.Name, version.Version, capabilityDefinitions, jdxPath)...)
		if hasCapabilityFlavors {
			allErrs = append(allErrs, field.Forbidden(jdxPath.Child("capabilityFlavors"), "must not be set as CloudProfile does not define capabilities. Use regions instead."))
		}
	}
	return allErrs
}

// validateCapabilityFlavors validates the capability flavors of a machine image version.
func validateCapabilityFlavors(providerImage apisalicloud.MachineImages, version apisalicloud.MachineImageVersion, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, jdxPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate each flavor's capabilities and regions
	for k, capabilitySet := range version.CapabilityFlavors {
		kdxPath := jdxPath.Child("capabilityFlavors").Index(k)
		allErrs = append(allErrs, gutil.ValidateCapabilities(capabilitySet.Capabilities, capabilityDefinitions, kdxPath.Child("capabilities"))...)
		allErrs = append(allErrs, validateRegions(capabilitySet.Regions, providerImage.Name, version.Version, capabilityDefinitions, kdxPath)...)
	}
	return allErrs
}

// validateRegionsWithCapabilities validates regions when using old format with capabilities CloudProfile.
// This allows architecture field in regions since it will be converted to capability flavors.
func validateRegionsWithCapabilities(regions []apisalicloud.RegionIDMapping, name, version string, jdxPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(regions) == 0 {
		return append(allErrs, field.Required(jdxPath.Child("regions"), fmt.Sprintf("must provide at least one region for machine image %q and version %q", name, version)))
	}

	for k, region := range regions {
		kdxPath := jdxPath.Child("regions").Index(k)

		if len(region.Name) == 0 {
			allErrs = append(allErrs, field.Required(kdxPath.Child("name"), "must provide a name"))
		}
		if len(region.ID) == 0 {
			allErrs = append(allErrs, field.Required(kdxPath.Child("id"), "must provide an id"))
		}
	}
	return allErrs
}

// validateRegions validates the regions of a machine image version or capability flavor.
func validateRegions(regions []apisalicloud.RegionIDMapping, name, version string, _ []gardencorev1beta1.CapabilityDefinition, jdxPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(regions) == 0 {
		return append(allErrs, field.Required(jdxPath.Child("regions"), fmt.Sprintf("must provide at least one region for machine image %q and version %q", name, version)))
	}

	for k, region := range regions {
		kdxPath := jdxPath.Child("regions").Index(k)

		if len(region.Name) == 0 {
			allErrs = append(allErrs, field.Required(kdxPath.Child("name"), "must provide a name"))
		}
		if len(region.ID) == 0 {
			allErrs = append(allErrs, field.Required(kdxPath.Child("id"), "must provide an id"))
		}
	}
	return allErrs
}

// NewProviderImagesContext creates a new ImagesContext for provider images.
func NewProviderImagesContext(providerImages []apisalicloud.MachineImages) *gutil.ImagesContext[apisalicloud.MachineImages, apisalicloud.MachineImageVersion] {
	return gutil.NewImagesContext(
		utils.CreateMapFromSlice(providerImages, func(mi apisalicloud.MachineImages) string { return mi.Name }),
		func(mi apisalicloud.MachineImages) map[string]apisalicloud.MachineImageVersion {
			return utils.CreateMapFromSlice(mi.Versions, func(v apisalicloud.MachineImageVersion) string { return v.Version })
		},
	)
}

// validateMachineImageMapping validates that for each machine image there is a corresponding cpConfig image.
func validateMachineImageMapping(machineImages []core.MachineImage, cpConfig *apisalicloud.CloudProfileConfig, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	providerImages := NewProviderImagesContext(cpConfig.MachineImages)

	// validate machine images
	for idxImage, machineImage := range machineImages {
		if len(machineImage.Versions) == 0 {
			continue
		}
		machineImagePath := fldPath.Index(idxImage)
		// validate that for each machine image there is a corresponding cpConfig image
		if _, existsInConfig := providerImages.GetImage(machineImage.Name); !existsInConfig {
			allErrs = append(allErrs, field.Required(machineImagePath, fmt.Sprintf("must provide an image mapping for image %q in providerConfig", machineImage.Name)))
			continue
		}

		allErrs = append(allErrs, validateMachineImageVersionMapping(machineImage, machineImagePath, capabilityDefinitions, providerImages)...)
	}

	return allErrs
}

func validateMachineImageVersionMapping(machineImage core.MachineImage, machineImagePath *field.Path, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, providerImages *gutil.ImagesContext[apisalicloud.MachineImages, apisalicloud.MachineImageVersion]) field.ErrorList {
	allErrs := field.ErrorList{}

	// validate that for each machine image version entry a mapped entry in cpConfig exists
	for idxVersion, version := range machineImage.Versions {
		machineImageVersionPath := machineImagePath.Child("versions").Index(idxVersion)
		// check that each MachineImageFlavor in version.CapabilityFlavors has a corresponding imageVersion.CapabilityFlavors
		imageVersion, exists := providerImages.GetImageVersion(machineImage.Name, version.Version)
		if !exists {
			allErrs = append(allErrs, field.Required(machineImageVersionPath,
				fmt.Sprintf("machine image version %s@%s is not defined in the providerConfig", machineImage.Name, version.Version)))
			continue
		}
		if len(capabilityDefinitions) > 0 {
			allErrs = append(allErrs, validateImageFlavorMapping(machineImage, version, machineImageVersionPath, capabilityDefinitions, imageVersion)...)
		}
	}
	return allErrs
}

// validateImageFlavorMapping validates that each flavor in a version has a corresponding mapping
// This function handles both the new format (capabilityFlavors) and old format (regions with architecture)
func validateImageFlavorMapping(machineImage core.MachineImage, version core.MachineImageVersion, machineImageVersionPath *field.Path, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, imageVersion apisalicloud.MachineImageVersion) field.ErrorList {
	allErrs := field.ErrorList{}

	var v1beta1Version gardencorev1beta1.MachineImageVersion
	if err := gardencoreapi.Scheme.Convert(&version, &v1beta1Version, nil); err != nil {
		return append(allErrs, field.InternalError(machineImageVersionPath, err))
	}

	defaultedCapabilityFlavors := gardencorev1beta1helper.GetImageFlavorsWithAppliedDefaults(v1beta1Version.CapabilityFlavors, capabilityDefinitions)

	// Check if provider uses old format (regions with architecture) or new format (capabilityFlavors)
	if len(imageVersion.CapabilityFlavors) > 0 {
		// New format: validate against capabilityFlavors
		for idxCapability, defaultedCapabilitySet := range defaultedCapabilityFlavors {
			isFound := false
			// search for the corresponding imageVersion.MachineImageFlavor
			for _, providerCapabilitySet := range imageVersion.CapabilityFlavors {
				providerDefaultedCapabilities := gardencorev1beta1.GetCapabilitiesWithAppliedDefaults(providerCapabilitySet.Capabilities, capabilityDefinitions)
				if gardencorev1beta1helper.AreCapabilitiesEqual(defaultedCapabilitySet.Capabilities, providerDefaultedCapabilities) {
					isFound = true
					break
				}
			}
			if !isFound {
				allErrs = append(allErrs, field.Required(machineImageVersionPath.Child("capabilityFlavors").Index(idxCapability),
					fmt.Sprintf("missing providerConfig mapping for machine image version %s@%s and capabilitySet %v", machineImage.Name, version.Version, defaultedCapabilitySet.Capabilities)))
			}
		}
	} else if len(imageVersion.Regions) > 0 {
		// Old format: validate against regions with architecture
		// Collect architectures from regions
		architecturesMap := utils.CreateMapFromSlice(imageVersion.Regions, func(re apisalicloud.RegionIDMapping) string {
			return v1beta1constants.ArchitectureAMD64
		})
		availableArchitectures := slices.Collect(maps.Keys(architecturesMap))

		// For each expected capability flavor, check if the architecture capability is available in regions
		for idxCapability, defaultedCapabilitySet := range defaultedCapabilityFlavors {
			expectedArchitectures := defaultedCapabilitySet.Capabilities[v1beta1constants.ArchitectureName]
			for _, expectedArch := range expectedArchitectures {
				if !slices.Contains(availableArchitectures, expectedArch) {
					allErrs = append(allErrs, field.Required(machineImageVersionPath.Child("capabilityFlavors").Index(idxCapability),
						fmt.Sprintf("missing providerConfig mapping for machine image version %s@%s and architecture: %s", machineImage.Name, version.Version, expectedArch)))
				}
			}
		}
	} else {
		// No regions or capabilityFlavors - this is already validated elsewhere
		for idxCapability, defaultedCapabilitySet := range defaultedCapabilityFlavors {
			allErrs = append(allErrs, field.Required(machineImageVersionPath.Child("capabilityFlavors").Index(idxCapability),
				fmt.Sprintf("missing providerConfig mapping for machine image version %s@%s and capabilitySet %v", machineImage.Name, version.Version, defaultedCapabilitySet.Capabilities)))
		}
	}

	return allErrs
}
