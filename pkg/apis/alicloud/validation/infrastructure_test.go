// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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

		pods        = "100.96.0.0/11"
		services    = "100.64.0.0/13"
		nodes       = "10.250.0.0/16"
		vpc         = "10.0.0.0/8"
		invalidCIDR = "invalid-cidr"
		networking  = core.Networking{}
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
	})

	Describe("#ValidateInfrastructureConfig", func() {
		Context("CIDR", func() {
			It("should forbid invalid VPC CIDRs", func() {
				infrastructureConfig.Networks.VPC.CIDR = &invalidCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.vpc.cidr"),
					"Detail": Equal("invalid CIDR address: invalid-cidr"),
				}))
			})

			It("should forbid invalid workers CIDR", func() {
				infrastructureConfig.Networks.Zones[0].Workers = invalidCIDR

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].workers"),
					"Detail": Equal("invalid CIDR address: invalid-cidr"),
				}))
			})

			It("should forbid workers CIDR which are not in Nodes CIDR", func() {
				infrastructureConfig.Networks.Zones[0].Workers = "1.1.1.1/32"

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Detail": Equal(`must not overlap with "networks.vpc.cidr" ("10.0.0.0/8")`),
				}))
			})

			It("should forbid Services CIDR to overlap with VPC CIDR", func() {
				networking.Services = ptr.To("10.0.0.1/32")

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

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

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")
				Expect(errorList).To(BeEmpty())
			})

			It("should allow specifying valid config", func() {
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")
				Expect(errorList).To(BeEmpty())
			})

			It("should allow specifying valid config with podsCIDR=nil and servicesCIDR=nil", func() {
				networking.Pods = nil
				networking.Services = nil
				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")
				Expect(errorList).To(BeEmpty())
			})

		})

		Context("dualStack", func() {
			var vpcID = "vpc-12345678"

			Context("Gardener-managed VPC (no VPC.ID)", func() {
				It("should pass when all zones omit ipv6CidrBlock (defaults to zone index)", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					// no Ipv6CidrBlock set; defaults are 0 and 1 — no conflict

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})

				It("should pass when some zones omit ipv6CidrBlock and there is no conflict with explicit values", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr5 := 5
					// zone[0] explicit=5, zone[1] nil → default index=1; no conflict
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr5

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})

				It("should pass when all zones have valid ipv6CidrBlock", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr0 := 0
					cidr1 := 1
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr0
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr1

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})

				It("should forbid when default index conflicts with an explicit value (zone[1] explicit=0 clashes with zone[0] default=0)", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr0 := 0
					// zone[0] nil → default 0; zone[1] explicit=0 → conflict
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr0

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[1].ipv6CidrBlock"),
						"Detail": ContainSubstring("must be unique across zones"),
					}))))
				})

				It("should forbid when default index conflicts with explicit value on a later zone (zone[0] explicit=1 clashes with zone[1] default=1)", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr1 := 1
					// zone[0] explicit=1; zone[1] nil → default index=1 → conflict
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr1

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[1].ipv6CidrBlock"),
						"Detail": ContainSubstring("default ipv6CidrBlock"),
					}))))
				})

				It("should forbid ipv6CidrBlock out of range (>255)", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr0 := 256
					cidr1 := 1
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr0
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr1

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[0].ipv6CidrBlock"),
						"Detail": ContainSubstring("must be in range 0-255"),
					}))))
				})

				It("should forbid ipv6CidrBlock out of range (<0)", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidrNeg := -1
					cidr1 := 1
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidrNeg
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr1

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[0].ipv6CidrBlock"),
						"Detail": ContainSubstring("must be in range 0-255"),
					}))))
				})

				It("should forbid duplicate ipv6CidrBlock across zones", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr := 5
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[1].ipv6CidrBlock"),
						"Detail": ContainSubstring("must be unique across zones"),
					}))))
				})
			})

			Context("user-provided VPC (VPC.ID set)", func() {
				BeforeEach(func() {
					infrastructureConfig.Networks.VPC = apisalicloud.VPC{ID: &vpcID}
				})

				// Note: with a bare VPC.ID and no CIDR, ValidateInfrastructureConfig will produce
				// unrelated CIDR/zone errors. Assertions below are scoped to the specific field
				// under test to avoid false failures from those unrelated errors.

				It("should require ipv6CidrBlock for every zone when dualStack.enabled=true and user-provided VPC", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					// no Ipv6CidrBlock set

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("networks.zones[0].ipv6CidrBlock"),
					}))))
					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("networks.zones[1].ipv6CidrBlock"),
					}))))
				})

				It("should pass when zones have valid ipv6CidrBlock", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr0 := 10
					cidr1 := 20
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr0
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr1

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})

				It("should forbid out-of-range ipv6CidrBlock even with user VPC", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidrBig := 300
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidrBig

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[0].ipv6CidrBlock"),
						"Detail": ContainSubstring("must be in range 0-255"),
					}))))
				})

				It("should forbid duplicate ipv6CidrBlock even with user VPC", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					cidr := 7
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":   Equal(field.ErrorTypeInvalid),
						"Field":  Equal("networks.zones[1].ipv6CidrBlock"),
						"Detail": ContainSubstring("must be unique across zones"),
					}))))
				})
			})

			Context("DualStack disabled", func() {
				It("should pass without ipv6CidrBlock when dualStack is nil", func() {
					// infrastructureConfig.DualStack is nil by default
					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")
					Expect(errorList).To(BeEmpty())
				})

				It("should pass without ipv6CidrBlock when dualStack.enabled=false", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: false}

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})

				It("should pass with ipv6CidrBlock set even when dualStack is disabled", func() {
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: false}
					cidr := 5
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr

					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})
			})

			Context("NLB region check", func() {
				BeforeEach(func() {
					cidr0 := 0
					cidr1 := 1
					infrastructureConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					infrastructureConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr0
					infrastructureConfig.Networks.Zones[1].Ipv6CidrBlock = &cidr1
				})

				It("should forbid dualStack in a region that does not support NLB", func() {
					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "ap-southeast-99")

					Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("region"),
					}))))
				})

				It("should allow dualStack in a region that supports NLB", func() {
					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

					Expect(errorList).To(BeEmpty())
				})

				It("should allow dualStack in another supported NLB region", func() {
					errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "eu-central-1")

					Expect(errorList).To(BeEmpty())
				})
			})
		})

		Context("useCustomRouteTable", func() {
			var vpcID = "vpc-12345678"

			// Note: VPC is replaced wholesale in these tests, which may produce unrelated
			// CIDR/zone validation errors. Assertions are scoped to the useCustomRouteTable
			// field only so that unrelated errors do not cause false failures.

			It("should allow useCustomRouteTable=true when vpc.id is not set", func() {
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{
					CIDR:                &vpc,
					UseCustomRouteTable: ptr.To(true),
				}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).NotTo(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Field": Equal("networks.vpc.useCustomRouteTable"),
				}))))
			})

			It("should allow useCustomRouteTable=true when vpc.id is set and gardenerManagedNATGateway=true", func() {
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{
					ID:                        &vpcID,
					UseCustomRouteTable:       ptr.To(true),
					GardenerManagedNATGateway: ptr.To(true),
				}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).NotTo(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Field": Equal("networks.vpc.gardenerManagedNATGateway"),
				}))))
			})

			It("should forbid useCustomRouteTable=true when vpc.id is set and gardenerManagedNATGateway is not set", func() {
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{
					ID:                  &vpcID,
					UseCustomRouteTable: ptr.To(true),
				}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("networks.vpc.gardenerManagedNATGateway"),
					"Detail": ContainSubstring("gardenerManagedNATGateway must be true when useCustomRouteTable is enabled with a user-provided VPC"),
				}))))
			})

			It("should forbid useCustomRouteTable=true when vpc.id is set and gardenerManagedNATGateway=false", func() {
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{
					ID:                        &vpcID,
					UseCustomRouteTable:       ptr.To(true),
					GardenerManagedNATGateway: ptr.To(false),
				}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("networks.vpc.gardenerManagedNATGateway"),
					"Detail": ContainSubstring("gardenerManagedNATGateway must be true when useCustomRouteTable is enabled with a user-provided VPC"),
				}))))
			})

			It("should allow useCustomRouteTable=false when vpc.id is not set", func() {
				infrastructureConfig.Networks.VPC = apisalicloud.VPC{
					CIDR:                &vpc,
					UseCustomRouteTable: ptr.To(false),
				}

				errorList := ValidateInfrastructureConfig(infrastructureConfig, &networking, "cn-hangzhou")

				Expect(errorList).NotTo(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Field": Equal("networks.vpc.useCustomRouteTable"),
				}))))
			})
		})
	})

	Describe("#ValidateInfrastructureConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateInfrastructureConfigUpdate(infrastructureConfig, infrastructureConfig)).To(BeEmpty())
		})

		It("should return no errors for migrate zone worker to workers", func() {
			oldInfrastructureConfig := infrastructureConfig.DeepCopy()
			tmpvalue := oldInfrastructureConfig.Networks.Zones[0].Worker
			oldInfrastructureConfig.Networks.Zones[0].Worker = oldInfrastructureConfig.Networks.Zones[0].Workers
			oldInfrastructureConfig.Networks.Zones[0].Workers = tmpvalue
			Expect(ValidateInfrastructureConfigUpdate(oldInfrastructureConfig, infrastructureConfig)).To(BeEmpty())
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

		Context("dualStack immutability", func() {
			It("should allow enabling dualStack (false -> true)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.DualStack = &apisalicloud.DualStack{Enabled: false}

				newConfig := oldConfig.DeepCopy()
				newConfig.DualStack = &apisalicloud.DualStack{Enabled: true}

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(BeEmpty())
			})

			It("should allow enabling dualStack (nil -> true)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				// DualStack is nil by default

				newConfig := oldConfig.DeepCopy()
				newConfig.DualStack = &apisalicloud.DualStack{Enabled: true}

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid disabling dualStack (true -> false)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.DualStack = &apisalicloud.DualStack{Enabled: true}

				newConfig := oldConfig.DeepCopy()
				newConfig.DualStack = &apisalicloud.DualStack{Enabled: false}

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("dualStack"),
				}))))
			})

			It("should forbid disabling dualStack (true -> nil)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.DualStack = &apisalicloud.DualStack{Enabled: true}

				newConfig := oldConfig.DeepCopy()
				newConfig.DualStack = nil

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("dualStack"),
				}))))
			})
		})

		Context("ipv6CidrBlock mutability", func() {
			It("should allow setting ipv6CidrBlock for the first time (nil -> value)", func() {
				newConfig := infrastructureConfig.DeepCopy()
				cidr := 5
				newConfig.Networks.Zones[0].Ipv6CidrBlock = &cidr

				errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newConfig)

				Expect(errorList).To(BeEmpty())
			})

			It("should allow changing ipv6CidrBlock once set (value -> new value)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				cidrOld := 1
				oldConfig.Networks.Zones[0].Ipv6CidrBlock = &cidrOld

				newConfig := oldConfig.DeepCopy()
				cidrNew := 2
				newConfig.Networks.Zones[0].Ipv6CidrBlock = &cidrNew

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid removing ipv6CidrBlock once set (value -> nil)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				cidrOld := 3
				oldConfig.Networks.Zones[0].Ipv6CidrBlock = &cidrOld

				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.Zones[0].Ipv6CidrBlock = nil

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("networks.zones[0].ipv6CidrBlock"),
					"Detail": ContainSubstring("cannot be removed once set"),
				}))))
			})
		})

		Context("useCustomRouteTable immutability", func() {
			var vpcID = "vpc-12345678"

			It("should forbid changing useCustomRouteTable from nil to true", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.VPC = apisalicloud.VPC{ID: &vpcID}

				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.VPC.UseCustomRouteTable = ptr.To(true)

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("networks.vpc.useCustomRouteTable"),
					"Detail": ContainSubstring("useCustomRouteTable can only be set at shoot creation time and cannot be changed afterwards"),
				}))))
			})

			It("should forbid changing useCustomRouteTable from false to true", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.VPC = apisalicloud.VPC{ID: &vpcID, UseCustomRouteTable: ptr.To(false)}

				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.VPC.UseCustomRouteTable = ptr.To(true)

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("networks.vpc.useCustomRouteTable"),
					"Detail": ContainSubstring("useCustomRouteTable can only be set at shoot creation time and cannot be changed afterwards"),
				}))))
			})

			It("should forbid changing useCustomRouteTable from true to false", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.VPC = apisalicloud.VPC{ID: &vpcID, UseCustomRouteTable: ptr.To(true)}

				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.VPC.UseCustomRouteTable = ptr.To(false)

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("networks.vpc.useCustomRouteTable"),
					"Detail": ContainSubstring("useCustomRouteTable can only be set at shoot creation time and cannot be changed afterwards"),
				}))))
			})

			It("should forbid changing useCustomRouteTable from true to nil", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.VPC = apisalicloud.VPC{ID: &vpcID, UseCustomRouteTable: ptr.To(true)}

				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.VPC.UseCustomRouteTable = nil

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("networks.vpc.useCustomRouteTable"),
					"Detail": ContainSubstring("useCustomRouteTable can only be set at shoot creation time and cannot be changed afterwards"),
				}))))
			})

			It("should allow nil to false transition (semantically equivalent)", func() {
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.VPC = apisalicloud.VPC{ID: &vpcID}

				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.VPC.UseCustomRouteTable = ptr.To(false)

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(BeEmpty())
			})
		})
	})

	Describe("BYO (Bring Your Own Infrastructure)", func() {
		var vpcID = "vpc-byo12345"

		Context("#ValidateInfrastructureConfig — BYO VSwitch", func() {
			var byoConfig *apisalicloud.InfrastructureConfig

			BeforeEach(func() {
				vswID0 := "vsw-aaa"
				vswID1 := "vsw-bbb"
				byoConfig = &apisalicloud.InfrastructureConfig{
					Networks: apisalicloud.Networks{
						VPC: apisalicloud.VPC{ID: &vpcID},
						Zones: []apisalicloud.Zone{
							{Name: "zone1", WorkersVSwitchID: &vswID0},
							{Name: "zone2", WorkersVSwitchID: &vswID1},
						},
					},
				}
			})

			It("should allow valid BYO VSwitch config", func() {
				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")
				Expect(errorList).To(BeEmpty())
			})

			It("should allow BYO VSwitch with nodesSecurityGroupID", func() {
				sgID := "sg-abc"
				byoConfig.Networks.NodesSecurityGroupID = &sgID
				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")
				Expect(errorList).To(BeEmpty())
			})

			It("should require vpc.id when workersVSwitchID is set", func() {
				byoConfig.Networks.VPC = apisalicloud.VPC{CIDR: &vpc}

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("networks.vpc.id"),
				}))))
			})

			It("should forbid gardenerManagedNATGateway when workersVSwitchID is set", func() {
				byoConfig.Networks.VPC.GardenerManagedNATGateway = ptr.To(true)

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("networks.vpc.gardenerManagedNATGateway"),
				}))))
			})

			It("should forbid useCustomRouteTable when workersVSwitchID is set", func() {
				byoConfig.Networks.VPC.UseCustomRouteTable = ptr.To(true)

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("networks.vpc.useCustomRouteTable"),
				}))))
			})

			It("should forbid natGateway on a BYO zone", func() {
				eipID := "eip-abc"
				byoConfig.Networks.Zones[0].NatGateway = &apisalicloud.NatGatewayConfig{EIPAllocationID: &eipID}

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("networks.zones[0].natGateway"),
				}))))
			})

			It("should forbid mixing BYO and CIDR zones", func() {
				byoConfig.Networks.Zones[1].WorkersVSwitchID = nil
				byoConfig.Networks.Zones[1].Workers = "10.250.4.0/24"

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("networks.zones"),
				}))))
			})

			It("should forbid setting both workers CIDR and workersVSwitchID on same zone", func() {
				byoConfig.Networks.Zones[0].Workers = "10.250.3.0/24"

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("networks.zones[0]"),
				}))))
			})

			It("should forbid zone with neither workers CIDR nor workersVSwitchID", func() {
				byoConfig.Networks.Zones[0].WorkersVSwitchID = nil

				errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("networks.zones[0]"),
				}))))
			})

			Context("dualStack + BYO VSwitch", func() {
				It("should allow omitting ipv6CidrBlock for BYO zones (user pre-configures IPv6 on VSwitch)", func() {
					byoConfig.DualStack = &apisalicloud.DualStack{Enabled: true}
					// no Ipv6CidrBlock set — should be optional for BYO zones

					errorList := ValidateInfrastructureConfig(byoConfig, &networking, "cn-hangzhou")

					Expect(errorList).NotTo(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": ContainSubstring("ipv6CidrBlock"),
					}))))
				})
			})
		})

		Context("#ValidateInfrastructureConfig — nodesSecurityGroupID", func() {
			It("should allow nodesSecurityGroupID with vpc.id", func() {
				sgID := "sg-abc"
				config := &apisalicloud.InfrastructureConfig{
					Networks: apisalicloud.Networks{
						VPC:                  apisalicloud.VPC{ID: &vpcID},
						NodesSecurityGroupID: &sgID,
						Zones: []apisalicloud.Zone{
							{Name: "zone1", Workers: "10.250.3.0/24"},
						},
					},
				}
				errorList := ValidateInfrastructureConfig(config, &networking, "cn-hangzhou")
				Expect(errorList).To(BeEmpty())
			})

			It("should require vpc.id when nodesSecurityGroupID is set", func() {
				sgID := "sg-abc"
				config := &apisalicloud.InfrastructureConfig{
					Networks: apisalicloud.Networks{
						VPC:                  apisalicloud.VPC{CIDR: &vpc},
						NodesSecurityGroupID: &sgID,
						Zones: []apisalicloud.Zone{
							{Name: "zone1", Workers: "10.250.3.0/24"},
						},
					},
				}
				errorList := ValidateInfrastructureConfig(config, &networking, "cn-hangzhou")

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("networks.vpc.id"),
				}))))
			})
		})

		Context("#ValidateInfrastructureConfigUpdate — BYO immutability", func() {
			It("should forbid changing workersVSwitchID", func() {
				vswID := "vsw-aaa"
				vswIDNew := "vsw-bbb"
				oldConfig := &apisalicloud.InfrastructureConfig{
					Networks: apisalicloud.Networks{
						VPC: apisalicloud.VPC{ID: &vpcID},
						Zones: []apisalicloud.Zone{
							{Name: "zone1", WorkersVSwitchID: &vswID},
						},
					},
				}
				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.Zones[0].WorkersVSwitchID = &vswIDNew

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("networks.zones[0].workersVSwitchID"),
				}))))
			})

			It("should forbid switching from CIDR to workersVSwitchID", func() {
				vswID := "vsw-aaa"
				oldConfig := infrastructureConfig.DeepCopy()
				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.Zones[0].Workers = ""
				newConfig.Networks.Zones[0].WorkersVSwitchID = &vswID

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("networks.zones[0]"),
				}))))
			})

			It("should forbid switching from workersVSwitchID to CIDR", func() {
				vswID := "vsw-aaa"
				oldConfig := &apisalicloud.InfrastructureConfig{
					Networks: apisalicloud.Networks{
						VPC: apisalicloud.VPC{ID: &vpcID},
						Zones: []apisalicloud.Zone{
							{Name: "zone1", WorkersVSwitchID: &vswID},
						},
					},
				}
				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.Zones[0].WorkersVSwitchID = nil
				newConfig.Networks.Zones[0].Workers = "10.250.3.0/24"

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("networks.zones[0]"),
				}))))
			})

			It("should allow workersVSwitchID unchanged", func() {
				vswID := "vsw-aaa"
				config := &apisalicloud.InfrastructureConfig{
					Networks: apisalicloud.Networks{
						VPC: apisalicloud.VPC{ID: &vpcID},
						Zones: []apisalicloud.Zone{
							{Name: "zone1", WorkersVSwitchID: &vswID},
						},
					},
				}
				errorList := ValidateInfrastructureConfigUpdate(config, config.DeepCopy())
				Expect(errorList).To(BeEmpty())
			})

			It("should forbid changing nodesSecurityGroupID", func() {
				sgID := "sg-old"
				sgIDNew := "sg-new"
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.NodesSecurityGroupID = &sgID
				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.NodesSecurityGroupID = &sgIDNew

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("networks.nodesSecurityGroupID"),
				}))))
			})

			It("should forbid removing nodesSecurityGroupID once set", func() {
				sgID := "sg-abc"
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.NodesSecurityGroupID = &sgID
				newConfig := oldConfig.DeepCopy()
				newConfig.Networks.NodesSecurityGroupID = nil

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("networks.nodesSecurityGroupID"),
				}))))
			})

			It("should allow nodesSecurityGroupID unchanged", func() {
				sgID := "sg-abc"
				oldConfig := infrastructureConfig.DeepCopy()
				oldConfig.Networks.NodesSecurityGroupID = &sgID
				newConfig := oldConfig.DeepCopy()

				errorList := ValidateInfrastructureConfigUpdate(oldConfig, newConfig)

				Expect(errorList).To(BeEmpty())
			})

			It("should allow nodesSecurityGroupID nil -> nil", func() {
				errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, infrastructureConfig.DeepCopy())
				Expect(errorList).To(BeEmpty())
			})
		})
	})
})
