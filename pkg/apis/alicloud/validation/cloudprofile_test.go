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

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

var _ = Describe("CloudProfileConfig validation", func() {
	Describe("#ValidateCloudProfileConfig", func() {
		var cloudProfileConfig *apisalicloud.CloudProfileConfig

		BeforeEach(func() {
			cloudProfileConfig = &apisalicloud.CloudProfileConfig{
				MachineImages: []apisalicloud.MachineImages{
					{
						Name: "ubuntu",
						Versions: []apisalicloud.MachineImageVersion{
							{
								Version: "1.2.3",
								Regions: []apisalicloud.RegionIDMapping{
									{
										Name: "china",
										ID:   "some-image-id",
									},
								},
							},
						},
					},
				},
			}
		})

		Context("machine image validation", func() {
			It("should enforce that at least one machine image has been defined", func() {
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, field.NewPath("root"))

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages"),
				}))))
			})

			It("should forbid unsupported machine image configuration", func() {
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{{}}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, field.NewPath("root"))

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages[0].versions"),
				}))))
			})

			It("should forbid unsupported machine image version configuration", func() {
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{
					{
						Name:     "abc",
						Versions: []apisalicloud.MachineImageVersion{{}},
					},
				}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, field.NewPath("root"))

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages[0].versions[0].version"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages[0].versions[0].regions"),
				}))))
			})

			It("should forbid unsupported machine image region configuration", func() {
				cloudProfileConfig.MachineImages = []apisalicloud.MachineImages{
					{
						Name: "abc",
						Versions: []apisalicloud.MachineImageVersion{
							{
								Version: "1.2.3",
								Regions: []apisalicloud.RegionIDMapping{{}},
							},
						},
					},
				}

				errorList := ValidateCloudProfileConfig(cloudProfileConfig, field.NewPath("root"))

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages[0].versions[0].regions[0].name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("root.machineImages[0].versions[0].regions[0].id"),
				}))))
			})
		})
	})
})
