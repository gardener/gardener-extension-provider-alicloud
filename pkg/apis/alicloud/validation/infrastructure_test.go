// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

var _ = Describe("InfrastructureConfig validation", func() {
	var (
		infrastructureConfig *apisalicloud.InfrastructureConfig

		pods                = "100.96.0.0/11"
		services            = "100.64.0.0/13"
		nodes               = "10.250.0.0/16"
		vpc                 = "10.0.0.0/8"
		invalidCIDR         = "invalid-cidr"
		networking          = core.Networking{}
		shootRegion         = "region1"
		dualStackRegionList []string
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
		dualStackRegionList = make([]string, 2)
		dualStackRegionList = append(dualStackRegionList, "region1")
		dualStackRegionList = append(dualStackRegionList, "region2")

	})

	Describe("#ValidateInfrastructureConfig", func() {
		Context("CIDR", func() {
			It("should forbid invalid VPC CIDRs", func() {
				infrastructureConfig.Networks.VPC.CIDR = &invalidCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.vpc.cidr"),
					"Detail": Equal("invalid CIDR address: invalid-cidr"),
				}))
			})

			It("should forbid invalid workers CIDR", func() {
				infrastructureConfig.Networks.Zones[0].Workers = invalidCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal("invalid CIDR address: invalid-cidr"),
				}))
			})

			It("should forbid workers CIDR which are not in Nodes CIDR", func() {
				infrastructureConfig.Networks.Zones[0].Workers = "1.1.1.1/32"

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal(`must be a subset of "networking.nodes" ("10.250.0.0/16")`),
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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

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
					"Detail": Equal(`must be a subset of "networking.nodes" ("1.1.1.1/32")`),
				}))
			})

			It("should forbid Pod CIDR to overlap with VPC CIDR", func() {
				networking.Pods = ptr.To("10.0.0.1/32")

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Detail": Equal(`must not overlap with "networks.vpc.cidr" ("10.0.0.0/8")`),
				}))
			})

			It("should forbid Services CIDR to overlap with VPC CIDR", func() {
				networking.Services = ptr.To("10.0.0.1/32")

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)

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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)
				Expect(errorList).To(BeEmpty())
			})

			It("should allow specifying valid config", func() {
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)
				Expect(errorList).To(BeEmpty())
			})

			It("should allow specifying valid config with podsCIDR=nil and servicesCIDR=nil", func() {
				networking.Pods = nil
				networking.Services = nil
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)
				Expect(errorList).To(BeEmpty())
			})

			It("should forbid if both provide vpc id and set dualstak enable true", func() {
				vpcID := "vpc-provided"
				infrastructureConfig.DualStack = &apisalicloud.DualStack{
					Enabled: true,
				}
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{
					ID: &vpcID,
				}
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.vpc"),
					"Detail": ContainSubstring("can not set vpc id when DualStack enabled"),
				}))
			})
			It("should forbid if shoot region is not in dualstack region list and set dualstak enable true", func() {
				infrastructureConfig.DualStack = &apisalicloud.DualStack{
					Enabled: true,
				}
				shootRegion = "no_dualstack_region"
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, shootRegion, dualStackRegionList)
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("dualstack.enabled"),
					"Detail": ContainSubstring("can not enable DualStack in target region"),
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

		It("should forbid when change DualStack enable from true to false", func() {
			infrastructureConfig.DualStack = &apisalicloud.DualStack{
				Enabled: true,
			}
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.DualStack = nil
			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(HaveLen(1))
			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeForbidden),
				"Field":  Equal("dualStack.enabled"),
				"Detail": ContainSubstring("field can't be changed from \"true\" to \"false\""),
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
