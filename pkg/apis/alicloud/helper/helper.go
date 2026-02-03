// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
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
func AppendMachineImage(machineImages []api.MachineImage, machineImage api.MachineImage) []api.MachineImage {
	expectEncripted := machineImage.Encrypted
	if expectEncripted == nil {
		expectEncripted = ptr.To(false)
	}
	if _, err := FindMachineImage(machineImages, machineImage.Name, machineImage.Version, *expectEncripted); err != nil {
		return append(machineImages, machineImage)
	}

	return machineImages
}

// FindImageForRegionFromCloudProfile takes a list of machine images, and the desired image name, version, and region.
// It tries to find the image with the given name and version in the desired region.
// If no image is found then an error is returned.
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

// FindImageInCloudProfile takes a list of machine images and tries to find the first entry
// whose name, version, region, architecture, capabilities and zone matches with the given ones. If no such entry is
// found then an error will be returned.
func FindImageInCloudProfile(
	cloudProfileConfig *api.CloudProfileConfig,
	name, version, region string,
	machineCapabilities gardencorev1beta1.Capabilities,
	capabilityDefinitions []gardencorev1beta1.CapabilityDefinition,
) (*api.MachineImageFlavor, error) {
	if cloudProfileConfig == nil {
		return nil, fmt.Errorf("cloud profile config is nil")
	}
	machineImages := cloudProfileConfig.MachineImages

	capabilitySet, err := findMachineImageFlavor(machineImages, name, version, region, machineCapabilities, capabilityDefinitions)
	if err != nil {
		return nil, fmt.Errorf("could not find an Image for region %q, image %q, version %q that supports %v: %w", region, name, version, machineCapabilities, err)
	}

	if capabilitySet != nil && len(capabilitySet.Regions) > 0 && capabilitySet.Regions[0].ID != "" {
		return capabilitySet, nil
	}
	return nil, fmt.Errorf("could not find an Image for region %q, image %q, version %q that supports %v", region, name, version, machineCapabilities)
}

// FindImageInWorkerStatus takes a list of machine images from the worker status and tries to find the first entry
// whose name, version, architecture, capabilities and zone matches with the given ones. If no such entry is
// found then an error will be returned.
func FindImageInWorkerStatus(machineImages []api.MachineImage, name string, version string, machineCapabilities gardencorev1beta1.Capabilities, capabilityDefinitions []gardencorev1beta1.CapabilityDefinition) (*api.MachineImage, error) {
	// If no capabilityDefinitions are specified, return the (legacy) architecture format field as no Capabilities are used.
	if len(capabilityDefinitions) == 0 {
		for _, statusMachineImage := range machineImages {
			if statusMachineImage.Name == name && statusMachineImage.Version == version {
				return &statusMachineImage, nil
			}
		}
		return nil, fmt.Errorf("no machine image found for image %q with version %q", name, version)
	}

	// If capabilityDefinitions are specified, we need to find the best matching capability set.
	for _, statusMachineImage := range machineImages {
		var statusMachineImageV1alpha1 v1alpha1.MachineImage
		if err := v1alpha1.Convert_alicloud_MachineImage_To_v1alpha1_MachineImage(&statusMachineImage, &statusMachineImageV1alpha1, nil); err != nil {
			return nil, fmt.Errorf("failed to convert machine image: %w", err)
		}
		if statusMachineImage.Name == name && statusMachineImage.Version == version && gardencorev1beta1helper.AreCapabilitiesCompatible(statusMachineImageV1alpha1.Capabilities, machineCapabilities, capabilityDefinitions) {
			return &statusMachineImage, nil
		}
	}
	return nil, fmt.Errorf("no machine image found for image %q with version %q and capabilities %v", name, version, machineCapabilities)
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

			if len(capabilityDefinitions) == 0 {
				for _, mapping := range version.Regions {
					if region == mapping.Name {
						return &api.MachineImageFlavor{
							Regions:      []api.RegionIDMapping{mapping},
							Capabilities: gardencorev1beta1.Capabilities{},
						}, nil
					}
				}
				continue
			}

			filteredCapabilityFlavors := filterCapabilityFlavorsByRegion(version.CapabilityFlavors, region)
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
