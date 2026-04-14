// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// FindVSwitchForPurposeAndZone takes a list of vswitches and tries to find the first entry
// whose purpose and zone matches with the given purpose and zone. If no such entry is found then
// an error will be returned.
func FindVSwitchForPurposeAndZone(vswitches []api.VSwitch, purpose api.Purpose, zone string) (*api.VSwitch, error) {
	for _, vswitch := range vswitches {
		if vswitch.Purpose == purpose && vswitch.Zone == zone {
			return &vswitch, nil
		}
	}
	return nil, fmt.Errorf("no vswitch with purpose %q in zone %q found", purpose, zone)
}

// FindVSwitchForPurpose takes a list of vswitches and tries to find the first entry
// whose purpose matches with the given purpose.
// If no such entry is found then an error will be returned.
func FindVSwitchForPurpose(vswitches []api.VSwitch, purpose api.Purpose) (*api.VSwitch, error) {
	for _, vswitch := range vswitches {
		if vswitch.Purpose == purpose {
			return &vswitch, nil
		}
	}
	return nil, fmt.Errorf("no vswitch with purpose %q found", purpose)
}

// FindSecurityGroupByPurpose takes a list of security groups and tries to find the first entry
// whose purpose matches with the given purpose. If no such entry is found then an error will be
// returned.
func FindSecurityGroupByPurpose(securityGroups []api.SecurityGroup, purpose api.Purpose) (*api.SecurityGroup, error) {
	for _, securityGroup := range securityGroups {
		if securityGroup.Purpose == purpose {
			return &securityGroup, nil
		}
	}
	return nil, fmt.Errorf("cannot find security group with purpose %q", purpose)
}

func matchEncryptedFlag(encrypted *bool, expectEncrypted bool) bool {
	checkedVal := encrypted
	if checkedVal == nil {
		checkedVal = ptr.To(false)
	}

	return *checkedVal == expectEncrypted
}

// FindMachineImage takes a list of machine images and tries to find the first entry
// whose name, version and encrypted flag matches with the given name, version and encrypted flag.
// If no such entry is found then an error will be returned.
func FindMachineImage(machineImages []api.MachineImage, imageName, imageVersion string, encrypted bool) (*api.MachineImage, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == imageName && machineImage.Version == imageVersion && matchEncryptedFlag(machineImage.Encrypted, encrypted) {
			return &machineImage, nil
		}
	}

	if encrypted {
		return nil, fmt.Errorf("no encrypted machine image name %q in version %q found", imageName, imageVersion)
	}
	return nil, fmt.Errorf("no machine image name %q in version %q found", imageName, imageVersion)
}

// AppendMachineImage will append a given MachineImage to an existing image list.
// If a same image (by checking name, version and encrypted flag) already exists, nothing happens
func AppendMachineImage(machineImages []api.MachineImage, machineImage api.MachineImage, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition) []api.MachineImage {
	// support for cloudprofile machine images without capabilities
	if len(capabilityDefinitions) == 0 {
		for _, image := range machineImages {
			imageEncrypted := image.Encrypted != nil && *image.Encrypted
			if image.Name == machineImage.Name && image.Version == machineImage.Version && matchEncryptedFlag(machineImage.Encrypted, imageEncrypted) {
				// If the image already exists without capabilities, we can just return the existing list.
				return machineImages
			}
		}
		return append(machineImages, api.MachineImage{
			Name:      machineImage.Name,
			Version:   machineImage.Version,
			ID:        machineImage.ID,
			Encrypted: machineImage.Encrypted,
		})
	}

	defaultedCapabilities := gardencorev1beta1.GetCapabilitiesWithAppliedDefaults(machineImage.Capabilities, capabilityDefinitions)

	for _, existingMachineImage := range machineImages {
		existingDefaultedCapabilities := gardencorev1beta1.GetCapabilitiesWithAppliedDefaults(existingMachineImage.Capabilities, capabilityDefinitions)
		existingEncrypted := existingMachineImage.Encrypted != nil && *existingMachineImage.Encrypted
		if existingMachineImage.Name == machineImage.Name && existingMachineImage.Version == machineImage.Version && matchEncryptedFlag(machineImage.Encrypted, existingEncrypted) && gardencorev1beta1helper.AreCapabilitiesEqual(defaultedCapabilities, existingDefaultedCapabilities) {
			// If the image already exists with the same capabilities return the existing list.
			return machineImages
		}
	}

	// If the image does not exist, we create a new machine image entry with the capabilities.
	machineImages = append(machineImages, api.MachineImage{
		Name:         machineImage.Name,
		Version:      machineImage.Version,
		ID:           machineImage.ID,
		Encrypted:    machineImage.Encrypted,
		Capabilities: machineImage.Capabilities,
	})

	return machineImages
}

// FindImageForRegionFromCloudProfile takes a list of machine images, and the desired image name, version, and region.
// It tries to find the image with the given name and version in the desired region.
// If no image is found then an error is returned.
// only used for non capability based cloud profiles.
func FindImageForRegionFromCloudProfile(cloudProfileConfig *api.CloudProfileConfig, imageName, imageVersion, regionName string) (string, error) {
	if cloudProfileConfig != nil {
		for _, machineImage := range cloudProfileConfig.MachineImages {
			if machineImage.Name != imageName {
				continue
			}
			for _, version := range machineImage.Versions {
				if imageVersion != version.Version {
					continue
				}
				for _, mapping := range version.Regions {
					if regionName == mapping.Name {
						return mapping.ID, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("could not find an image for name %q in version %q", imageName, imageVersion)
}

// FindImageInCloudProfileFlavor takes a list of machine images and tries to find the first entry
// whose name, version, region, architecture, capabilities and zone matches with the given ones. If no such entry is
// found then an error will be returned.
func FindImageInCloudProfileFlavor(
	cloudProfileConfig *api.CloudProfileConfig,
	name, version, region string,
	machineCapabilities gardencorev1beta1.Capabilities,
	capabilityDefinitions []gardencorev1beta1.CapabilityDefinition,
) (*api.MachineImageFlavor, error) {
	if cloudProfileConfig == nil {
		return nil, fmt.Errorf("cloud profile config is nil")
	}
	machineImages := cloudProfileConfig.MachineImages

	imageFlavor, err := findMachineImageFlavor(machineImages, name, version, region, machineCapabilities, capabilityDefinitions)
	if err != nil {
		return nil, fmt.Errorf("could not find an Image for region %q, image %q, version %q that supports %v: %w", region, name, version, machineCapabilities, err)
	}

	if imageFlavor != nil && len(imageFlavor.Regions) > 0 && imageFlavor.Regions[0].ID != "" {
		return imageFlavor, nil
	}
	return nil, fmt.Errorf("could not find an Image for region %q, image %q, version %q that supports %v", region, name, version, machineCapabilities)
}

func findMachineImageFlavor(
	machineImages []api.MachineImages,
	imageName, imageVersion, region string,
	machineCapabilities gardencorev1beta1.Capabilities,
	capabilityDefinitions []gardencorev1beta1.CapabilityDefinition,
) (*api.MachineImageFlavor, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name != imageName {
			continue
		}
		for _, version := range machineImage.Versions {
			if imageVersion != version.Version {
				continue
			}

			// - New format: capabilityFlavors
			// - Old format: regions only amd64 architecture (converted to capability flavors)
			var capabilityFlavors []api.MachineImageFlavor
			if len(version.CapabilityFlavors) > 0 {
				// New format: use capabilityFlavors in version
				capabilityFlavors = version.CapabilityFlavors
			} else if len(version.Regions) > 0 {
				// Old format: convert regions only amd64 architecture to capability flavors
				capabilityFlavors = append(capabilityFlavors, api.MachineImageFlavor{
					Capabilities: gardencorev1beta1.Capabilities{
						v1beta1constants.ArchitectureName: []string{v1beta1constants.ArchitectureAMD64},
					},
					Regions: version.Regions,
				})
			} else {
				continue
			}
			filteredCapabilityFlavors := filterCapabilityFlavorsByRegion(capabilityFlavors, region)
			bestMatch, err := worker.FindBestImageFlavor(filteredCapabilityFlavors, machineCapabilities, capabilityDefinitions)
			if err != nil {
				return nil, fmt.Errorf("could not determine best flavor %w", err)
			}
			return bestMatch, nil
		}
	}
	return nil, nil
}

// filterCapabilityFlavorsByRegion returns a new list with capabilityFlavors that only contain RegionIDMappings
// of the region to filter for.
func filterCapabilityFlavorsByRegion(capabilityFlavors []api.MachineImageFlavor, regionName string) []*api.MachineImageFlavor {
	var compatibleFlavors []*api.MachineImageFlavor

	for _, capabilityFlavor := range capabilityFlavors {
		var regionIDMapping *api.RegionIDMapping
		for _, region := range capabilityFlavor.Regions {
			if region.Name == regionName {
				regionIDMapping = &region
			}
		}
		if regionIDMapping != nil {
			compatibleFlavors = append(compatibleFlavors, &api.MachineImageFlavor{
				Regions:      []api.RegionIDMapping{*regionIDMapping},
				Capabilities: capabilityFlavor.Capabilities,
			})
		}
	}
	return compatibleFlavors
}

// EnsureUniformMachineImages ensures that all machine images are in the same format, either with or without Capabilities.
func EnsureUniformMachineImages(images []api.MachineImage, definitions []gardencorev1beta1.CapabilityDefinition) []api.MachineImage {
	var uniformMachineImages []api.MachineImage

	if len(definitions) == 0 {
		// transform images that were added with Capabilities to the legacy format without Capabilities
		for _, img := range images {
			if len(img.Capabilities) == 0 {
				// image is already legacy format
				uniformMachineImages = AppendMachineImage(uniformMachineImages, img, definitions)
				continue
			}
			uniformMachineImages = AppendMachineImage(uniformMachineImages, api.MachineImage{
				Name:      img.Name,
				Version:   img.Version,
				ID:        img.ID,
				Encrypted: img.Encrypted,
			}, definitions)
		}
		return uniformMachineImages
	}

	// transform images that were added without Capabilities to contain a MachineImageFlavor with defaulted Architecture
	for _, img := range images {
		if len(img.Capabilities) > 0 {
			// image is already in the new format with Capabilities
			uniformMachineImages = AppendMachineImage(uniformMachineImages, img, definitions)
		} else {
			uniformMachineImages = AppendMachineImage(uniformMachineImages, api.MachineImage{
				Name:         img.Name,
				Version:      img.Version,
				ID:           img.ID,
				Encrypted:    img.Encrypted,
				Capabilities: gardencorev1beta1.Capabilities{v1beta1constants.ArchitectureName: []string{v1beta1constants.ArchitectureAMD64}},
			}, definitions)
		}
	}
	return uniformMachineImages
}

// UniformSingleMachineImage ensure that machine image are in the same format, either with or without Capabilities.
func UniformSingleMachineImage(img *api.MachineImage, definitions []gardencorev1beta1.CapabilityDefinition) *api.MachineImage {
	if len(definitions) == 0 {
		// transform image that were added with Capabilities to the legacy format without Capabilities
		if len(img.Capabilities) == 0 {
			// image is already legacy format
			return img
		} else {
			return &api.MachineImage{
				Name:      img.Name,
				Version:   img.Version,
				ID:        img.ID,
				Encrypted: img.Encrypted,
			}
		}
	}

	// transform image that were added without Capabilities to contain a MachineImageFlavor with defaulted Architecture
	if len(img.Capabilities) > 0 {
		// image is already in the new format with Capabilities
		return img
	} else {
		return &api.MachineImage{
			Name:         img.Name,
			Version:      img.Version,
			ID:           img.ID,
			Encrypted:    img.Encrypted,
			Capabilities: gardencorev1beta1.Capabilities{v1beta1constants.ArchitectureName: []string{v1beta1constants.ArchitectureAMD64}},
		}
	}
}
