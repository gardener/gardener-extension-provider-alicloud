// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package helper

import (
	"fmt"

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
		checkedVal = Bool(false)
	}

	return *checkedVal == expectEncrypted
}

// Bool returns a pointer to of the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// FindMachineImage takes a list of machine images and tries to find the first entry
// whose name, version and encrypted flag matches with the given name and version encrypted flag.
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
		expectEncripted = Bool(false)
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
