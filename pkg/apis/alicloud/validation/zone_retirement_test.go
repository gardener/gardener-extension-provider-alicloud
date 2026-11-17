// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

func retirementConfig() *api.InfrastructureConfig {
	return &api.InfrastructureConfig{Networks: api.Networks{
		VPC:   api.VPC{CIDR: ptr.To("10.0.0.0/16")},
		Zones: []api.Zone{{Name: "zone1", Workers: "10.0.1.0/24"}, {Name: "zone2", Workers: "10.0.2.0/24"}},
	}}
}

var _ = Describe("Zone retirement validation", func() {
	DescribeTable("first zone retirement follows the NAT gateway mode",
		func(vpc api.VPC, byo, allowed bool) {
			config := retirementConfig()
			config.Networks.VPC = vpc
			config.Networks.Zones[0].Retired = true
			if byo {
				for i := range config.Networks.Zones {
					config.Networks.Zones[i].Workers = ""
					config.Networks.Zones[i].WorkersVSwitchID = ptr.To("vsw-" + config.Networks.Zones[i].Name)
				}
			}
			errs := ValidateInfrastructureConfig(config, nil, "cn-hangzhou")
			updateErrs := ValidateInfrastructureConfigUpdate(config.DeepCopy(), config, nil, nil)
			if allowed {
				Expect(errs).To(BeEmpty())
				Expect(updateErrs).To(BeEmpty())
			} else {
				Expect(errs.ToAggregate().Error()).To(ContainSubstring("networks.zones[0].retired"))
				Expect(updateErrs.ToAggregate().Error()).To(ContainSubstring("networks.zones[0].retired"))
			}
		},
		Entry("created VPC defaults to managed NAT", api.VPC{CIDR: ptr.To("10.0.0.0/16")}, false, false),
		Entry("created VPC still manages NAT with explicit false", api.VPC{CIDR: ptr.To("10.0.0.0/16"), GardenerManagedNATGateway: ptr.To(false)}, false, false),
		Entry("user VPC with managed NAT", api.VPC{ID: ptr.To("vpc-user"), GardenerManagedNATGateway: ptr.To(true)}, false, false),
		Entry("user NAT with omitted flag", api.VPC{ID: ptr.To("vpc-user")}, false, true),
		Entry("user NAT with explicit false", api.VPC{ID: ptr.To("vpc-user"), GardenerManagedNATGateway: ptr.To(false)}, false, true),
		Entry("BYO", api.VPC{ID: ptr.To("vpc-user")}, true, true),
	)

	It("allows retiring a non-first managed zone", func() {
		old := retirementConfig()
		config := old.DeepCopy()
		config.Networks.Zones[1].Retired = true
		Expect(ValidateInfrastructureConfig(config, nil, "cn-hangzhou")).To(BeEmpty())
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).To(BeEmpty())
	})

	It("requires an active zone even with user NAT", func() {
		old := retirementConfig()
		old.Networks.VPC = api.VPC{ID: ptr.To("vpc-user")}
		config := old.DeepCopy()
		for i := range config.Networks.Zones {
			config.Networks.Zones[i].Retired = true
		}
		Expect(ValidateInfrastructureConfig(config, nil, "cn-hangzhou").ToAggregate().Error()).To(ContainSubstring("at least one zone must remain active"))
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil).ToAggregate().Error()).To(ContainSubstring("at least one zone must remain active"))
	})

	It("does not allow restoring a retired zone", func() {
		old := retirementConfig()
		old.Networks.Zones[1].Retired = true
		config := old.DeepCopy()
		config.Networks.Zones[1].Retired = false
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil).ToAggregate().Error()).To(ContainSubstring("cannot be restored"))
	})

	DescribeTable("freezes resource fields when requesting and after requesting retirement",
		func(mutate func(*api.Zone)) {
			for _, alreadyRetired := range []bool{false, true} {
				old := retirementConfig()
				old.Networks.Zones[1].Retired = alreadyRetired
				old.Networks.Zones[1].Ipv6CidrBlock = ptr.To(1)
				old.Networks.Zones[1].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-old")}
				config := old.DeepCopy()
				config.Networks.Zones[1].Retired = true
				mutate(&config.Networks.Zones[1])
				errs := ValidateInfrastructureConfigUpdate(old, config, nil, nil)
				Expect(errs).NotTo(BeEmpty())
				Expect(errs.ToAggregate().Error()).To(ContainSubstring("networks.zones[1]"))
			}
		},
		Entry("name", func(z *api.Zone) { z.Name = "zone3" }),
		Entry("workers CIDR", func(z *api.Zone) { z.Workers = "10.0.3.0/24" }),
		Entry("deprecated worker field", func(z *api.Zone) { z.Worker, z.Workers = z.Workers, "" }),
		Entry("switch to BYO", func(z *api.Zone) { z.Workers = ""; z.WorkersVSwitchID = ptr.To("vsw-user") }),
		Entry("IPv6 index", func(z *api.Zone) { z.Ipv6CidrBlock = ptr.To(2) }),
		Entry("remove IPv6 index", func(z *api.Zone) { z.Ipv6CidrBlock = nil }),
		Entry("EIP allocation", func(z *api.Zone) { z.NatGateway.EIPAllocationID = ptr.To("eip-new") }),
		Entry("remove NAT config", func(z *api.Zone) { z.NatGateway = nil }),
	)

	It("freezes a BYO VSwitch even if unreferenced or gone", func() {
		for _, alreadyRetired := range []bool{false, true} {
			old := retirementConfig()
			old.Networks.VPC = api.VPC{ID: ptr.To("vpc-user")}
			for i := range old.Networks.Zones {
				old.Networks.Zones[i].Workers = ""
				old.Networks.Zones[i].WorkersVSwitchID = ptr.To("vsw-" + old.Networks.Zones[i].Name)
			}
			old.Networks.Zones[1].Retired = alreadyRetired
			config := old.DeepCopy()
			config.Networks.Zones[1].Retired = true
			config.Networks.Zones[1].WorkersVSwitchID = ptr.To("vsw-new")
			Expect(ValidateInfrastructureConfigUpdate(old, config, nil, []string{"vsw-zone2"})).NotTo(BeEmpty())
		}
	})

	It("preserves active zone IPv6 and EIP update exceptions", func() {
		old := retirementConfig()
		old.Networks.Zones[0].Ipv6CidrBlock = ptr.To(0)
		config := old.DeepCopy()
		config.Networks.Zones[0].Ipv6CidrBlock = ptr.To(3)
		config.Networks.Zones[0].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-new")}
		config.Networks.Zones[1].Retired = true
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).To(BeEmpty())
	})

	It("keeps retired names reserved when appending zones", func() {
		old := retirementConfig()
		old.Networks.Zones[1].Retired = true
		config := old.DeepCopy()
		config.Networks.Zones = append(config.Networks.Zones, api.Zone{Name: "zone2", Workers: "10.0.3.0/24"})
		Expect(ValidateInfrastructureConfig(config, nil, "cn-hangzhou").ToAggregate().Error()).To(ContainSubstring("networks.zones[2].name"))
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).NotTo(BeEmpty())
		config.Networks.Zones[2].Name = "zone3"
		Expect(ValidateInfrastructureConfig(config, nil, "cn-hangzhou")).To(BeEmpty())
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).To(BeEmpty())
	})

	It("does not allow removing or reordering retired entries", func() {
		old := retirementConfig()
		old.Networks.Zones[1].Retired = true
		config := old.DeepCopy()
		config.Networks.Zones = config.Networks.Zones[:1]
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).NotTo(BeEmpty())
		config = old.DeepCopy()
		config.Networks.Zones[0], config.Networks.Zones[1] = config.Networks.Zones[1], config.Networks.Zones[0]
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).NotTo(BeEmpty())
	})

	It("does not reuse retired IPv4 CIDRs or default IPv6 indices", func() {
		config := retirementConfig()
		config.Networks.Zones[1].Retired = true
		config.Networks.Zones = append(config.Networks.Zones, api.Zone{Name: "zone3", Workers: "10.0.2.0/24"})
		Expect(ValidateInfrastructureConfig(config, nil, "cn-hangzhou")).NotTo(BeEmpty())
		config.Networks.Zones[2].Workers = "10.0.3.0/24"
		config.DualStack = &api.DualStack{Enabled: true}
		config.Networks.Zones[2].Ipv6CidrBlock = ptr.To(1)
		Expect(ValidateInfrastructureConfig(config, nil, "cn-hangzhou").ToAggregate().Error()).To(ContainSubstring("must be unique"))
	})

	It("does not permit route table mode changes along with retirement", func() {
		old := retirementConfig()
		config := old.DeepCopy()
		config.Networks.Zones[1].Retired = true
		config.Networks.VPC.UseCustomRouteTable = ptr.To(true)
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil).ToAggregate().Error()).To(ContainSubstring("useCustomRouteTable"))
		config.Networks.VPC.UseCustomRouteTable = ptr.To(false)
		Expect(ValidateInfrastructureConfigUpdate(old, config, nil, nil)).To(BeEmpty())
	})

	It("requires workers to stop referencing the target zone", func() {
		old := retirementConfig()
		config := old.DeepCopy()
		config.Networks.Zones[1].Retired = true
		workers := []core.Worker{{Name: "pool", Zones: []string{"zone2"}}}
		Expect(ValidateInfrastructureConfigUpdate(old, config, workers, nil).ToAggregate().Error()).To(ContainSubstring("migrate the workers"))
	})

	DescribeTable("worker references are checked even without an infrastructure update",
		func(workerZone string, allowed bool) {
			config := retirementConfig()
			config.Networks.Zones[1].Retired = true
			workers := []core.Worker{{Name: "pool", Zones: []string{workerZone}, Volume: &core.Volume{Type: ptr.To("cloud_essd"), VolumeSize: "40Gi"}}}
			errs := ValidateWorkers(workers, config.Networks.Zones, field.NewPath("spec", "provider", "workers"))
			if allowed {
				Expect(errs).To(BeEmpty())
			} else {
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Field).To(Equal("spec.provider.workers[0].zones[0]"))
				if workerZone == "zone2" {
					Expect(errs[0].Detail).To(ContainSubstring("zone is retired"))
				}
			}
		},
		Entry("active", "zone1", true),
		Entry("retired", "zone2", false),
		Entry("unknown", "zone3", false),
	)
})
