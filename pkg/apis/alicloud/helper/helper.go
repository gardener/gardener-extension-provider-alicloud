// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"fmt"

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
