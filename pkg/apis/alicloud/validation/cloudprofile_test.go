// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

var _ = Describe("CloudProfileConfig validation", func() {
	DescribeTableSubtree("#ValidateCloudProfileConfig", func(isCapabilitiesCloudProfile bool) {
		var (
			capabilityDefinitions []v1beta1.CapabilityDefinition
			machineImages         []core.MachineImage
			cloudProfileConfig    *apisalicloud.CloudProfileConfig
			machineImageName      string
			machineImageVersion   string
			fldPath               *field.Path
		)
		BeforeEach(func() {
			regions := []apisalicloud.RegionIDMapping{{
				Name: "china",
				ID:   "some-image-id",
			}}
			var capabilityFlavors []apisalicloud.MachineImageFlavor

			if isCapabilitiesCloudProfile {
				capabilityDefinitions = []v1beta1.CapabilityDefinition{{
					Name:   v1beta1constants.ArchitectureName,
					Values: []string{"amd64"},
				}}
				capabilityFlavors = []apisalicloud.MachineImageFlavor{{
					Regions: regions,
					Capabilities: v1beta1.Capabilities{
						v1beta1constants.ArchitectureName: []string{"amd64"},
					}}}
				regions = nil
			}
			machineImageName = "ubuntu"
			machineImageVersion = "1.2.3"
			cloudProfileConfig = &apisalicloud.CloudProfileConfig{
				MachineImages: []apisalicloud.MachineImages{
					{
						Name: machineImageName,
						Versions: []apisalicloud.MachineImageVersion{
							{
								Version:           machineImageVersion,
								Regions:           regions,
								CapabilityFlavors: capabilityFlavors,
							},
						},
					},
				},
			}
			machineImages = []core.MachineImage{
				{
					Name: machineImageName,
					Versions: []core.MachineImageVersion{
						{
							ExpirableVersion: core.ExpirableVersion{Version: machineImageVersion},
							Architectures:    []string{"amd64"},
						},
					},
				},
			}
		})

		Context("machine image validation", func() {
			It("should pass validation with valid config", func() {
				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)
				Expect(errorList).To(BeEmpty())
			})

			It("should enforce that at least one machine image has been defined", func() {
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.machineImages[0]"),
				}))))
			})

			It("should forbid images with empty regions", func() {
				var fieldMatcher string
				if isCapabilitiesCloudProfile {
					fieldMatcher = "machineImages[0].versions[0].capabilityFlavors[0].regions"
					cloudProfileConfig.MachineImages[0].Versions[0].CapabilityFlavors[0].Regions = []apisalicloud.RegionIDMapping{}
				} else {
					fieldMatcher = "machineImages[0].versions[0].regions"
					cloudProfileConfig.MachineImages[0].Versions[0].Regions = []apisalicloud.RegionIDMapping{}
				}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Detail": Equal("must provide at least one region for machine image \"ubuntu\" and version \"1.2.3\""),
					"Field":  Equal(fieldMatcher),
				}))))
			})

			It("should forbid unsupported machine image configuration", func() {
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{{}}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].versions"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.machineImages[0]"),
				}))))
			})

			It("should forbid unsupported machine image version configuration", func() {
				var matcher types.GomegaMatcher

				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{
					{
						Name:     "abc",
						Versions: []apisalicloud.MachineImageVersion{{}},
					},
				}
				if isCapabilitiesCloudProfile {
					matcher = Equal("machineImages[0].versions[0].capabilityFlavors[0].regions")
					cloudProfileConfig.MachineImages[0].Versions[0].CapabilityFlavors = []apisalicloud.MachineImageFlavor{{}}
				} else {
					matcher = Equal("machineImages[0].versions[0].regions")
				}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machineImages[0].versions[0].version"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": matcher,
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Detail": Equal("must provide an image mapping for image \"ubuntu\" in providerConfig"),
					"Field":  Equal("spec.machineImages[0]"),
				}))))
			})

			It("should forbid unsupported machine image region configuration", func() {
				var machineImageVersion apisalicloud.MachineImageVersion
				var nameMatcher, idMatcher types.GomegaMatcher
				if isCapabilitiesCloudProfile {
					nameMatcher = Equal("machineImages[0].versions[0].capabilityFlavors[0].regions[0].name")
					idMatcher = Equal("machineImages[0].versions[0].capabilityFlavors[0].regions[0].id")
					machineImageVersion = apisalicloud.MachineImageVersion{
						Version: "1.2.3",
						CapabilityFlavors: []apisalicloud.MachineImageFlavor{{
							Regions:      []apisalicloud.RegionIDMapping{{}},
							Capabilities: v1beta1.Capabilities{v1beta1constants.ArchitectureName: {"amd64"}},
						}},
					}
				} else {
					nameMatcher = Equal("machineImages[0].versions[0].regions[0].name")
					idMatcher = Equal("machineImages[0].versions[0].regions[0].id")
					machineImageVersion = apisalicloud.MachineImageVersion{
						Version: "1.2.3",
						Regions: []apisalicloud.RegionIDMapping{{}},
					}
				}
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{
					{
						Name:     "abc",
						Versions: []apisalicloud.MachineImageVersion{machineImageVersion},
					},
				}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Detail": Equal("must provide a name"),
					"Field":  nameMatcher,
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Detail": Equal("must provide an id"),
					"Field":  idMatcher,
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.machineImages[0]"),
				}))))
			})

			It("should reject when machineImage.regions and machineImage.capabilityFlavors is set", func() {
				var fieldMatcher types.GomegaMatcher
				if isCapabilitiesCloudProfile {
					fieldMatcher = Equal("machineImages[0].versions[0].regions")
				} else {
					fieldMatcher = Equal("machineImages[0].versions[0].capabilityFlavors")
				}
				cloudProfileConfig.MachineImages[0].Versions[0].Regions = append(cloudProfileConfig.MachineImages[0].Versions[0].Regions, apisalicloud.RegionIDMapping{
					Name: "china",
					ID:   "id-1234",
				})
				cloudProfileConfig.MachineImages[0].Versions[0].CapabilityFlavors = append(cloudProfileConfig.MachineImages[0].Versions[0].CapabilityFlavors, apisalicloud.MachineImageFlavor{
					Regions: []apisalicloud.RegionIDMapping{{Name: "china", ID: "id-1234"}},
				})

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, machineImages, capabilityDefinitions, fldPath)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  fieldMatcher,
					"Detail": ContainSubstring("must not be set as CloudProfile"),
				}))))
			})
		})
	},
		Entry("CloudProfile uses regions only", false),
		Entry("CloudProfile uses capabilities", true))
})
