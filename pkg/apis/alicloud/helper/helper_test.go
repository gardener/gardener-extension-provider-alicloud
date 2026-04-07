// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
)

const profileImageID = "id-1235"

var _ = Describe("Helper", func() {
	var (
		purpose           api.Purpose = "foo"
		purposeWrong      api.Purpose = "baz"
		imageCapabilities v1beta1.Capabilities
	)
	DescribeTable("#FindVSwitchForPurposeAndZone",
		func(vswitches []api.VSwitch, purpose api.Purpose, zone string, expectedVSwitch *api.VSwitch, expectErr bool) {
			subnet, err := FindVSwitchForPurposeAndZone(vswitches, purpose, zone)
			expectResults(subnet, expectedVSwitch, err, expectErr)
		},

		Entry("list is nil", nil, purpose, "europe", nil, true),
		Entry("empty list", []api.VSwitch{}, purpose, "europe", nil, true),
		Entry("entry not found (no purpose)", []api.VSwitch{{ID: "bar", Purpose: purposeWrong, Zone: "europe"}}, purpose, "europe", nil, true),
		Entry("entry not found (no zone)", []api.VSwitch{{ID: "bar", Purpose: purposeWrong, Zone: "europe"}}, purpose, "asia", nil, true),
		Entry("entry exists", []api.VSwitch{{ID: "bar", Purpose: purposeWrong, Zone: "europe"}}, purposeWrong, "europe", &api.VSwitch{ID: "bar", Purpose: purposeWrong, Zone: "europe"}, false),
	)

	DescribeTable("#FindVSwitchForPurpose",
		func(vswitches []api.VSwitch, purpose api.Purpose, expectedVSwitch *api.VSwitch, expectErr bool) {
			subnet, err := FindVSwitchForPurpose(vswitches, purpose)
			expectResults(subnet, expectedVSwitch, err, expectErr)
		},

		Entry("list is nil", nil, purpose, nil, true),
		Entry("empty list", []api.VSwitch{}, purpose, nil, true),
		Entry("entry not found (no purpose)", []api.VSwitch{{ID: "bar", Purpose: purposeWrong, Zone: "europe"}}, purpose, nil, true),
		Entry("entry exists", []api.VSwitch{{ID: "bar", Purpose: purposeWrong, Zone: "europe"}}, purposeWrong, &api.VSwitch{ID: "bar", Purpose: purposeWrong, Zone: "europe"}, false),
	)

	DescribeTable("#FindSecurityGroupByPurpose",
		func(securityGroups []api.SecurityGroup, purpose api.Purpose, expectedSecurityGroup *api.SecurityGroup, expectErr bool) {
			securityGroup, err := FindSecurityGroupByPurpose(securityGroups, purpose)
			expectResults(securityGroup, expectedSecurityGroup, err, expectErr)
		},

		Entry("list is nil", nil, purpose, nil, true),
		Entry("empty list", []api.SecurityGroup{}, purpose, nil, true),
		Entry("entry not found", []api.SecurityGroup{{ID: "bar", Purpose: purposeWrong}}, purpose, nil, true),
		Entry("entry exists", []api.SecurityGroup{{ID: "bar", Purpose: purpose}}, purpose, &api.SecurityGroup{ID: "bar", Purpose: purpose}, false),
	)

	DescribeTable("#FindMachineImage",
		func(machineImage []api.MachineImage, name, version string, encrypted bool, expectedMachineImage *api.MachineImage, expectErr bool) {
			found, err := FindMachineImage(machineImage, name, version, encrypted)
			expectResults(found, expectedMachineImage, err, expectErr)
		},

		Entry("list is nil", nil, "foo", "1.2.3", true, nil, true),
		Entry("empty list", []api.MachineImage{}, "foo", "1.2.3", true, nil, true),
		Entry("entry not found (no name)", []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123"}}, "foo", "1.2.3", true, nil, true),
		Entry("entry not found (no version)", []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123"}}, "foo", "1.2.4", true, nil, true),
		Entry("entry not found (empty encrypted)", []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123"}}, "bar", "1.2.3", true, nil, true),
		Entry("entry not found (false encrypted)", []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123", Encrypted: ptr.To(false)}}, "bar", "1.2.3", true, nil, true),

		Entry("entry exists (encrypted value exists)", []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123", Encrypted: ptr.To(true)}}, "bar", "1.2.3", true, &api.MachineImage{Name: "bar", Version: "1.2.3", ID: "id123", Encrypted: ptr.To(true)}, false),
		Entry("entry exists (empty encrypted value)", []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123"}}, "bar", "1.2.3", false, &api.MachineImage{Name: "bar", Version: "1.2.3", ID: "id123"}, false),
	)

	Describe("#AppendMachineImage",
		func() {

			It("should append a non-existing image", func() {
				existingImages := []api.MachineImage{{Name: "bar", Version: "1.2.3", ID: "id123", Encrypted: ptr.To(true)}}
				imageToInsert := api.MachineImage{Name: "bar", Version: "1.2.4", ID: "id123"}
				existingImages = AppendMachineImage(existingImages, imageToInsert, nil)
				Expect(existingImages).To(HaveLen(2))
				Expect(existingImages).To(ContainElement(imageToInsert))
			})

			It("should not append the image", func() {
				imageToInsert := api.MachineImage{Name: "bar", Version: "1.2.3", ID: "id123", Encrypted: ptr.To(false)}
				imageExisting := api.MachineImage{Name: "bar", Version: "1.2.3", ID: "id123"}
				existingImages := []api.MachineImage{imageExisting}
				existingImages = AppendMachineImage(existingImages, imageToInsert, nil)
				Expect(existingImages).To(HaveLen(1))
				Expect(existingImages[0]).To(Equal(imageExisting))
			})
		})

	DescribeTable("#FindImageForRegion for non capabilities",
		func(profileImages []api.MachineImages, imageName, version, region string, expectedImage string) {
			cfg := &api.CloudProfileConfig{}
			cfg.MachineImages = profileImages
			image, err := FindImageForRegionFromCloudProfile(cfg, imageName, version, region)

			Expect(image).To(Equal(expectedImage))
			if expectedImage != "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},

		Entry("list is nil", nil, "ubuntu", "1", "china", ""),

		Entry("profile empty list", []api.MachineImages{}, "ubuntu", "1", "china", ""),
		Entry("profile entry not found (image does not exist)", makeProfileMachineImages("debian", "1", "china", nil), "ubuntu", "1", "china", ""),
		Entry("profile entry not found (version does not exist)", makeProfileMachineImages("ubuntu", "2", "china", nil), "ubuntu", "1", "china", ""),
		Entry("profile entry", makeProfileMachineImages("ubuntu", "1", "china", nil), "ubuntu", "1", "china", profileImageID),
		Entry("profile non matching region", makeProfileMachineImages("ubuntu", "1", "china", nil), "ubuntu", "1", "eu", ""),
	)

	DescribeTable("#FindImageInCloudProfileFlavor for capabilities",
		func(profileImages []api.MachineImages, imageName, version, regionName string, arch *string, expectedID string) {
			var capabilityDefinitions []v1beta1.CapabilityDefinition
			var machineTypeCapabilities v1beta1.Capabilities

			capabilityDefinitions = []v1beta1.CapabilityDefinition{
				{Name: "architecture", Values: []string{"amd64", "arm64"}},
				{Name: "capability1", Values: []string{"value1", "value2", "value3"}},
			}
			machineTypeCapabilities = v1beta1.Capabilities{
				"architecture": []string{"amd64"},
				"capability1":  []string{"value2"},
			}
			imageCapabilities = v1beta1.Capabilities{
				"architecture": []string{"amd64"},
				"capability1":  []string{"value2"},
			}

			machineTypeCapabilities["architecture"] = []string{*arch}
			cfg := &api.CloudProfileConfig{}
			cfg.MachineImages = profileImages

			imageFlavor, err := FindImageInCloudProfileFlavor(cfg, imageName, version, regionName, machineTypeCapabilities, capabilityDefinitions)

			if expectedID != "" {
				Expect(err).NotTo(HaveOccurred())
				Expect(imageFlavor.Regions[0].ID).To(Equal(expectedID))
			} else {
				Expect(err).To(HaveOccurred())
			}
		},

		Entry("list is nil", nil, "ubuntu", "1", "china", ptr.To("amd64"), ""),

		Entry("profile empty list", []api.MachineImages{}, "ubuntu", "1", "china", ptr.To("amd64"), ""),
		Entry("profile entry not found (image does not exist)", makeProfileMachineImages("debian", "1", "china", imageCapabilities), "ubuntu", "1", "china", ptr.To("amd64"), ""),
		Entry("profile entry not found (version does not exist)", makeProfileMachineImages("ubuntu", "2", "china", imageCapabilities), "ubuntu", "1", "china", ptr.To("amd64"), ""),
		Entry("profile entry not found (architecture does not exist)", makeProfileMachineImages("ubuntu", "1", "china", imageCapabilities), "ubuntu", "1", "china", ptr.To("arm64"), ""),
		Entry("profile entry", makeProfileMachineImages("ubuntu", "1", "china", imageCapabilities), "ubuntu", "1", "china", ptr.To("amd64"), profileImageID),
		Entry("profile non matching region", makeProfileMachineImages("ubuntu", "1", "china", imageCapabilities), "ubuntu", "1", "europe", ptr.To("amd64"), ""),
	)

	Describe("Mixed format with capabilities", func() {
		var (
			capabilityDefinitions   []v1beta1.CapabilityDefinition
			machineTypeCapabilities v1beta1.Capabilities
			region                  = "eu-west-1"
		)

		BeforeEach(func() {
			capabilityDefinitions = []v1beta1.CapabilityDefinition{
				{Name: "architecture", Values: []string{"amd64", "arm64"}},
			}
			machineTypeCapabilities = v1beta1.Capabilities{
				"architecture": []string{"amd64"},
			}
		})

		DescribeTable("#FindImageInCloudProfileFlavor with old format (regions)",
			func(profileImages []api.MachineImages, imageName, version, regionName string, arch *string, expectedID string) {
				machineTypeCapabilities["architecture"] = []string{*arch}
				cfg := &api.CloudProfileConfig{}
				cfg.MachineImages = profileImages

				imageFlavor, err := FindImageInCloudProfileFlavor(cfg, imageName, version, regionName, machineTypeCapabilities, capabilityDefinitions)

				if expectedID != "" {
					Expect(err).NotTo(HaveOccurred())
					Expect(imageFlavor.Regions[0].ID).To(Equal(expectedID))
				} else {
					Expect(err).To(HaveOccurred())
				}
			},

			Entry("finds amd64 image using old format with capabilities defined",
				makeProfileMachineImagesOldFormat("ubuntu", "22.04", region, "id-ubuntu-amd64"),
				"ubuntu", "22.04", region, ptr.To("amd64"), "id-ubuntu-amd64"),

			Entry("does not find image when architecture mismatch (old format)",
				makeProfileMachineImagesOldFormat("ubuntu", "22.04", region, "id-ubuntu-amd64"),
				"ubuntu", "22.04", region, ptr.To("arm64"), ""),

			Entry("does not find image when region mismatch (old format)",
				makeProfileMachineImagesOldFormat("ubuntu", "22.04", region, "id-ubuntu-amd64"),
				"ubuntu", "22.04", "us-east-1", ptr.To("amd64"), ""),
		)

		DescribeTable("#FindImageInCloudProfileFlavor with mixed format (some versions old, some new)",
			func(imageName, version, regionName string, arch *string, expectedID string) {
				machineTypeCapabilities["architecture"] = []string{*arch}
				cfg := &api.CloudProfileConfig{}
				cfg.MachineImages = makeProfileMachineImagesMixedFormat()

				imageFlavor, err := FindImageInCloudProfileFlavor(cfg, imageName, version, regionName, machineTypeCapabilities, capabilityDefinitions)

				if expectedID != "" {
					Expect(err).NotTo(HaveOccurred())
					Expect(imageFlavor.Regions[0].ID).To(Equal(expectedID))
				} else {
					Expect(err).To(HaveOccurred())
				}
			},

			// Version 22.04 uses old format (regions)
			Entry("finds amd64 image from old format version",
				"ubuntu", "22.04", "eu-west-1", ptr.To("amd64"), "id-ubuntu-2204-amd64-eu"),
			Entry("does not find arm64 image from old format version (not defined)",
				"ubuntu", "22.04", "eu-west-1", ptr.To("arm64"), ""),
			Entry("finds amd64 image from old format version in different region",
				"ubuntu", "22.04", "us-east-1", ptr.To("amd64"), "id-ubuntu-2204-amd64-us"),

			// Version 23.10 uses new format (capabilityFlavors)
			Entry("finds amd64 image from new format version",
				"ubuntu", "23.10", "eu-west-1", ptr.To("amd64"), "id-ubuntu-2310-amd64-eu"),
			Entry("finds amd64 image from new format version in different region",
				"ubuntu", "23.10", "us-east-1", ptr.To("amd64"), "id-ubuntu-2310-amd64-us"),
			Entry("does not find arm64 image from new format version (not defined)",
				"ubuntu", "23.10", "eu-west-1", ptr.To("arm64"), ""),
		)

	})

	DescribeTable("EnsureUniformMachineImages", func(capabilityDefinitions []gardencorev1beta1.CapabilityDefinition, expectedImages []api.MachineImage) {
		machineImages := []api.MachineImage{
			// images with capability sets
			{
				Name:    "some-image",
				Version: "1.2.1",
				ID:      "ami-for-arm64",
				Capabilities: gardencorev1beta1.Capabilities{
					v1beta1constants.ArchitectureName: []string{"arm64"},
				},
			},
			{
				Name:    "some-image",
				Version: "1.2.2",
				ID:      "ami-for-amd64",
				Capabilities: gardencorev1beta1.Capabilities{
					v1beta1constants.ArchitectureName: []string{"amd64"},
				},
			},
			// legacy image entry without capability sets
			{
				Name:      "some-image",
				Version:   "1.2.3",
				ID:        "ami-for-amd64",
				Encrypted: ptr.To(false),
			},
			{
				Name:    "some-image",
				Version: "1.2.2",
				ID:      "ami-for-amd64",
			},
			{
				Name:      "some-image",
				Version:   "1.2.1",
				ID:        "ami-for-amd64",
				Encrypted: ptr.To(true),
			},
		}
		actualImages := EnsureUniformMachineImages(machineImages, capabilityDefinitions)
		Expect(actualImages).To(ContainElements(expectedImages))

	},
		Entry("should return images with Architecture", nil, []api.MachineImage{
			// images with capability sets
			{
				Name:    "some-image",
				Version: "1.2.1",
				ID:      "ami-for-arm64",
			},
			{
				Name:    "some-image",
				Version: "1.2.2",
				ID:      "ami-for-amd64",
			},
			// legacy image entry without capability sets
			{
				Name:      "some-image",
				Version:   "1.2.3",
				ID:        "ami-for-amd64",
				Encrypted: ptr.To(false),
			},
			{
				Name:      "some-image",
				Version:   "1.2.1",
				ID:        "ami-for-amd64",
				Encrypted: ptr.To(true),
			},
		}),
		Entry("should return images with Capabilities", []gardencorev1beta1.CapabilityDefinition{{
			Name:   v1beta1constants.ArchitectureName,
			Values: []string{"amd64", "arm64"},
		}}, []api.MachineImage{
			// images with capability sets
			{
				Name:    "some-image",
				Version: "1.2.1",
				ID:      "ami-for-arm64",
				Capabilities: gardencorev1beta1.Capabilities{
					v1beta1constants.ArchitectureName: []string{"arm64"},
				},
			},
			{
				Name:    "some-image",
				Version: "1.2.2",
				ID:      "ami-for-amd64",
				Capabilities: gardencorev1beta1.Capabilities{
					v1beta1constants.ArchitectureName: []string{"amd64"},
				},
			},
			// legacy image entry without capability sets
			{
				Name:      "some-image",
				Version:   "1.2.3",
				ID:        "ami-for-amd64",
				Encrypted: ptr.To(false),
				Capabilities: gardencorev1beta1.Capabilities{
					v1beta1constants.ArchitectureName: []string{"amd64"},
				}},
			{
				Name:      "some-image",
				Version:   "1.2.1",
				ID:        "ami-for-amd64",
				Encrypted: ptr.To(true),
				Capabilities: gardencorev1beta1.Capabilities{
					v1beta1constants.ArchitectureName: []string{"amd64"},
				},
			},
		}),
	)

})

func makeProfileMachineImages(name, version, region string, capabilities v1beta1.Capabilities) []api.MachineImages {
	versions := []api.MachineImageVersion{{
		Version: version,
	}}

	if capabilities == nil {
		versions[0].Regions = []api.RegionIDMapping{{
			Name: region,
			ID:   profileImageID,
		}}
	} else {
		versions[0].CapabilityFlavors = []api.MachineImageFlavor{{
			Capabilities: capabilities,
			Regions: []api.RegionIDMapping{{
				Name: region,
				ID:   profileImageID,
			}},
		}}
	}

	return []api.MachineImages{
		{
			Name:     name,
			Versions: versions,
		},
	}
}

func expectResults(result, expected interface{}, err error, expectErr bool) {
	if !expectErr {
		Expect(result).To(Equal(expected))
		Expect(err).NotTo(HaveOccurred())
	} else {
		Expect(result).To(BeNil())
		Expect(err).To(HaveOccurred())
	}
}

// makeProfileMachineImagesOldFormat creates machine images using the old format (regions with architecture)
// for use in tests with capabilities defined
func makeProfileMachineImagesOldFormat(name, version, region, id string) []api.MachineImages {
	return []api.MachineImages{
		{
			Name: name,
			Versions: []api.MachineImageVersion{{
				Version: version,
				Regions: []api.RegionIDMapping{{
					Name: region,
					ID:   id,
				}},
			}},
		},
	}
}

// makeProfileMachineImagesMixedFormat creates machine images with mixed format:
// - Version 22.04 uses old format (regions)
// - Version 23.10 uses new format (capabilityFlavors)
func makeProfileMachineImagesMixedFormat() []api.MachineImages {
	return []api.MachineImages{
		{
			Name: "ubuntu",
			Versions: []api.MachineImageVersion{
				{
					// Old format: regions
					Version: "22.04",
					Regions: []api.RegionIDMapping{
						{
							Name: "eu-west-1",
							ID:   "id-ubuntu-2204-amd64-eu",
						},
						{
							Name: "us-east-1",
							ID:   "id-ubuntu-2204-amd64-us",
						},
					},
				},
				{
					// New format: capabilityFlavors
					Version: "23.10",
					CapabilityFlavors: []api.MachineImageFlavor{
						{
							Capabilities: v1beta1.Capabilities{
								"architecture": []string{"amd64"},
							},
							Regions: []api.RegionIDMapping{
								{
									Name: "eu-west-1",
									ID:   "id-ubuntu-2310-amd64-eu",
								},
								{
									Name: "us-east-1",
									ID:   "id-ubuntu-2310-amd64-us",
								},
							},
						},
					},
				},
			},
		},
	}
}
