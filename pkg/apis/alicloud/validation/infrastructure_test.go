// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

var _ = Describe("InfrastructureConfig validation", func() {
	var (
		infrastructureConfig *apisalicloud.InfrastructureConfig

		pods                 = "100.96.0.0/11"
		services             = "100.64.0.0/13"
		nodes                = "10.250.0.0/16"
		vpc                  = "10.0.0.0/8"
		invalidCIDR          = "invalid-cidr"
		networking           = core.Networking{}
		validNatGatewayZones []string
	)

	BeforeEach(func() {
		networking = core.Networking{
			Pods:     &pods,
			Services: &services,
			Nodes:    &nodes,
		}
		infrastructureConfig = &apisalicloud.InfrastructureConfig{
			Networks: apisalicloud.Networks{
				VPC: apisalicloud.VPC{
					CIDR: &vpc,
				},
				Zones: []apisalicloud.Zone{
					{
						Name:    "zone1",
						Workers: "10.250.3.0/24",
					},
					{
						Name:    "zone2",
						Workers: "10.250.4.0/24",
					},
				},
			},
		}
		validNatGatewayZones = make([]string, 2)
		validNatGatewayZones = append(validNatGatewayZones, "zone1")
		validNatGatewayZones = append(validNatGatewayZones, "zone2")
	})

	Describe("#ValidateInfrastructureConfig", func() {
		Context("CIDR", func() {
			It("should forbid invalid VPC CIDRs", func() {
				infrastructureConfig.Networks.VPC.CIDR = &invalidCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.vpc.cidr"),
					"Detail": Equal("invalid CIDR address: invalid-cidr"),
				}))
			})

			It("should forbid invalid workers CIDR", func() {
				infrastructureConfig.Networks.Zones[0].Workers = invalidCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal("invalid CIDR address: invalid-cidr"),
				}))
			})

			It("should forbid workers CIDR which are not in Nodes CIDR", func() {
				infrastructureConfig.Networks.Zones[0].Workers = "1.1.1.1/32"

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal(`must be a subset of "<nil>" ("10.250.0.0/16")`),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal(`must be a subset of "networks.vpc.cidr" ("10.0.0.0/8")`),
				}))
			})

			It("should forbid Node which are not in VPC CIDR", func() {
				notOverlappingCIDR := "1.1.1.1/32"
				networking.Nodes = &notOverlappingCIDR
				infrastructureConfig.Networks.Zones[0].Workers = notOverlappingCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Detail": Equal(`must be a subset of "networks.vpc.cidr" ("10.0.0.0/8")`),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal(`must be a subset of "networks.vpc.cidr" ("10.0.0.0/8")`),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[1].workers"),
					"Detail": Equal(`must be a subset of "<nil>" ("1.1.1.1/32")`),
				}))
			})

			It("should forbid Pod CIDR to overlap with VPC CIDR", func() {
				networking.Pods = pointer.String("10.0.0.1/32")

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Detail": Equal(`must not overlap with "networks.vpc.cidr" ("10.0.0.0/8")`),
				}))
			})

			It("should forbid Services CIDR to overlap with VPC CIDR", func() {
				networking.Services = pointer.String("10.0.0.1/32")

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Detail": Equal(`must not overlap with "networks.vpc.cidr" ("10.0.0.0/8")`),
				}))
			})

			It("should forbid non canonical CIDRs", func() {
				var (
					vpcCIDR     = "10.0.0.3/8"
					nodeCIDR    = "10.250.0.3/16"
					podCIDR     = "100.96.0.4/11"
					serviceCIDR = "100.64.0.5/13"
				)

				networking.Nodes = &nodeCIDR
				networking.Pods = &podCIDR
				networking.Services = &serviceCIDR
				infrastructureConfig.Networks.Zones[0].Workers = "10.250.3.8/24"
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{CIDR: &vpcCIDR}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)

				Expect(errorList).To(HaveLen(2))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.vpc.cidr"),
					"Detail": Equal("must be valid canonical CIDR"),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal("must be valid canonical CIDR"),
				}))
			})

			It("should allow specifying eip id", func() {
				ipAllocID := "eip-ufxsdckfgitzcz"
				infrastructureConfig.Networks.Zones[0].NatGateway = &apisalicloud.NatGatewayConfig{
					EIPAllocationID: &ipAllocID,
				}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)
				Expect(errorList).To(BeEmpty())
			})

			It("should allow specifying valid config", func() {
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)
				Expect(errorList).To(BeEmpty())
			})

			It("should allow specifying valid config with podsCIDR=nil and servicesCIDR=nil", func() {
				networking.Pods = nil
				networking.Services = nil
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)
				Expect(errorList).To(BeEmpty())
			})
			It("should forbid if first zone is not in valid zone list", func() {
				infrastructureConfig.Networks.Zones[0].Name = "not-support-enhancenatgateway"
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, validNatGatewayZones)
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":     Equal(field.ErrorTypeNotSupported),
					"Field":    Equal("networks.zones[0]"),
					"BadValue": Equal("not-support-enhancenatgateway"),
					"Detail":   ContainSubstring("supported values"),
				}))
			})

		})
	})

	Describe("#ValidateInfrastructureConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateInfrastructureConfigUpdate(infrastructureConfig, infrastructureConfig)).To(BeEmpty())
		})

		It("should forbid changing the VPC section", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newCIDR := "1.2.3.4/5"
			newInfrastructureConfig.Networks.VPC.CIDR = &newCIDR

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("networks.vpc"),
			}))))
		})

		It("should forbid changing the worker CIRD section", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Networks.Zones[0].Workers = "10.225.3.0/24"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(HaveLen(1))
			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("networks.zones[0]"),
			}))
		})

		It("should forbid removing zone in zones section", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Networks.Zones = newInfrastructureConfig.Networks.Zones[1:]
			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(HaveLen(1))
			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("networks.zones"),
			}))
		})

		It("should allow appending zone in zones section", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Networks.Zones = append(newInfrastructureConfig.Networks.Zones, apisalicloud.Zone{
				Name:    "zone3",
				Workers: "10.250.4.0/24",
			})
			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(BeEmpty())
		})

		It("should allow changing nat gateway by specifying eip id", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			ipAllocID := "eip-ufxsdckfgitzcz"
			newInfrastructureConfig.Networks.Zones[0].NatGateway = &apisalicloud.NatGatewayConfig{
				EIPAllocationID: &ipAllocID,
			}
			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(BeEmpty())
		})
	})
})
