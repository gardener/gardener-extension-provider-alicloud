// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	aliclientmock "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient/mock"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

// newBYOTestContext constructs a FlowContext with a mock actor for BYO unit tests.
// Tests are in package infraflow (white-box) to access unexported fields directly.
func newBYOTestContext(
	ctrl *gomock.Controller,
	config *aliapi.InfrastructureConfig,
	initialState shared.FlatMap,
) (*FlowContext, *aliclientmock.MockActor) {
	mockActor := aliclientmock.NewMockActor(ctrl)
	wb := shared.NewWhiteboard()
	if initialState != nil {
		wb.ImportFromFlatMap(initialState)
	}
	noPersist := func(_ context.Context, _ shared.FlatMap) error { return nil }

	fc := &FlowContext{
		BasicFlowContext: *shared.NewBasicFlowContext(logr.Discard(), wb, noPersist),
		state:            wb,
		namespace:        "test-ns",
		config:           config,
		actor:            mockActor,
		updater:          aliclient.NewUpdater(mockActor),
		commonTags: aliclient.Tags{
			"kubernetes.io/cluster/test-ns": TagValueCluster,
			TagKeyName:                      "test-ns",
		},
	}
	return fc, mockActor
}

// byoConfig builds a minimal InfrastructureConfig for BYO tests.
func byoConfig(zones ...aliapi.Zone) *aliapi.InfrastructureConfig {
	return &aliapi.InfrastructureConfig{
		Networks: aliapi.Networks{
			VPC:   aliapi.VPC{ID: ptr.To("vpc-123")},
			Zones: zones,
		},
	}
}

// byoZone builds a Zone with a pre-existing VSwitch ID.
func byoZone(name, vswID string) aliapi.Zone {
	return aliapi.Zone{Name: name, WorkersVSwitchID: ptr.To(vswID)}
}

var _ = Describe("BYO infrastructure", func() {
	ctx := context.Background()

	Describe("ensureBYOZones", func() {
		It("stores VSwitch ID and discovers custom route table", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			fc, mockActor := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierVPC: "vpc-123",
			})

			mockActor.EXPECT().ListRouteTablesByVPC(ctx, "vpc-123").Return([]*aliclient.RouteTable{
				{RouteTableId: "rt-custom", RouteTableType: "Custom", VSwitchIds: []string{"vsw-aaa"}},
				{RouteTableId: "rt-system", RouteTableType: "System", VSwitchIds: []string{}},
			}, nil)

			Expect(fc.ensureBYOZones(ctx)).To(Succeed())
			Expect(fc.state.GetChild(ChildIdZones).GetChild("cn-hangzhou-a").Get(IdentifierZoneVSwitch)).
				To(Equal(ptr.To("vsw-aaa")))
			Expect(fc.state.Get(IdentifierRouteTable)).To(Equal(ptr.To("rt-custom")))
		})

		It("falls back to System route table when VSwitch has no explicit association", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-b", "vsw-bbb"))
			fc, mockActor := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierVPC: "vpc-123",
			})

			mockActor.EXPECT().ListRouteTablesByVPC(ctx, "vpc-123").Return([]*aliclient.RouteTable{
				// Custom RT does not contain vsw-bbb
				{RouteTableId: "rt-custom", RouteTableType: "Custom", VSwitchIds: []string{"vsw-other"}},
				{RouteTableId: "rt-system", RouteTableType: "System", VSwitchIds: []string{}},
			}, nil)

			Expect(fc.ensureBYOZones(ctx)).To(Succeed())
			Expect(fc.state.Get(IdentifierRouteTable)).To(Equal(ptr.To("rt-system")))
		})

		It("deduplicates route table ID when multiple zones share the same table", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(
				byoZone("cn-hangzhou-a", "vsw-aaa"),
				byoZone("cn-hangzhou-b", "vsw-bbb"),
			)
			fc, mockActor := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierVPC: "vpc-123",
			})

			mockActor.EXPECT().ListRouteTablesByVPC(ctx, "vpc-123").Return([]*aliclient.RouteTable{
				{RouteTableId: "rt-1", RouteTableType: "Custom", VSwitchIds: []string{"vsw-aaa", "vsw-bbb"}},
				{RouteTableId: "rt-system", RouteTableType: "System", VSwitchIds: []string{}},
			}, nil)

			Expect(fc.ensureBYOZones(ctx)).To(Succeed())
			// rt-1 must appear only once despite two zones being associated with it
			Expect(fc.state.Get(IdentifierRouteTable)).To(Equal(ptr.To("rt-1")))
		})

		It("produces comma-separated route table IDs when zones use different tables", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(
				byoZone("cn-hangzhou-a", "vsw-aaa"),
				byoZone("cn-hangzhou-b", "vsw-bbb"),
			)
			fc, mockActor := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierVPC: "vpc-123",
			})

			mockActor.EXPECT().ListRouteTablesByVPC(ctx, "vpc-123").Return([]*aliclient.RouteTable{
				{RouteTableId: "rt-1", RouteTableType: "Custom", VSwitchIds: []string{"vsw-aaa"}},
				{RouteTableId: "rt-2", RouteTableType: "Custom", VSwitchIds: []string{"vsw-bbb"}},
				{RouteTableId: "rt-system", RouteTableType: "System", VSwitchIds: []string{}},
			}, nil)

			Expect(fc.ensureBYOZones(ctx)).To(Succeed())
			Expect(fc.state.Get(IdentifierRouteTable)).To(Equal(ptr.To("rt-1,rt-2")))
		})

		It("returns error when IdentifierVPC is nil", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			// initialState deliberately omits IdentifierVPC
			fc, _ := newBYOTestContext(ctrl, config, nil)

			err := fc.ensureBYOZones(ctx)
			Expect(err).To(MatchError(ContainSubstring("IdentifierVPC is nil")))
		})

		It("wraps error returned by ListRouteTablesByVPC", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			fc, mockActor := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierVPC: "vpc-123",
			})

			mockActor.EXPECT().ListRouteTablesByVPC(ctx, "vpc-123").Return(nil, fmt.Errorf("sdk error"))

			err := fc.ensureBYOZones(ctx)
			Expect(err).To(MatchError(ContainSubstring("sdk error")))
		})
	})

	Describe("deleteZones BYO branch", func() {
		It("clears VSwitch and route table state without calling any delete APIs", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			fc, _ := newBYOTestContext(ctrl, config, shared.FlatMap{
				// Zone VSwitch state stored by ensureBYOZones
				"Zones/cn-hangzhou-a/" + IdentifierZoneVSwitch: "vsw-aaa",
				IdentifierRouteTable:                            "rt-1",
				// VPC state required to satisfy isBYOInfrastructure check (via config)
				IdentifierVPC: "vpc-123",
			})
			// No mock expectations set: any unexpected API call will fail the test.

			Expect(fc.deleteZones(ctx)).To(Succeed())
			Expect(fc.state.GetChild(ChildIdZones).GetChild("cn-hangzhou-a").IsAlreadyDeleted(IdentifierZoneVSwitch)).
				To(BeTrue())
			Expect(fc.state.IsAlreadyDeleted(IdentifierRouteTable)).To(BeTrue())
		})

	})

	Describe("deleteSecurityGroup BYO branch", func() {
		It("marks security group as deleted without calling delete API", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			config.Networks.NodesSecurityGroupID = ptr.To("sg-123")
			fc, _ := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierNodesSecurityGroup: "sg-123",
			})
			// No mock expectations set.

			Expect(fc.deleteSecurityGroup(ctx)).To(Succeed())
			Expect(fc.state.IsAlreadyDeleted(IdentifierNodesSecurityGroup)).To(BeTrue())
		})

		It("returns immediately when security group is already marked deleted", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			config.Networks.NodesSecurityGroupID = ptr.To("sg-123")
			// "<deleted>" is the Whiteboard marker for SetAsDeleted
			fc, _ := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierNodesSecurityGroup: "<deleted>",
			})

			Expect(fc.deleteSecurityGroup(ctx)).To(Succeed())
			// State must not have changed — still deleted, no panic or side-effect.
			Expect(fc.state.IsAlreadyDeleted(IdentifierNodesSecurityGroup)).To(BeTrue())
		})
	})

	Describe("ensureSecurityGroup BYO branch", func() {
		It("writes user-provided security group ID to state without creating a resource", func() {
			ctrl := gomock.NewController(GinkgoT())
			config := byoConfig(byoZone("cn-hangzhou-a", "vsw-aaa"))
			config.Networks.NodesSecurityGroupID = ptr.To("sg-byo-456")
			fc, _ := newBYOTestContext(ctrl, config, shared.FlatMap{
				IdentifierVPC: "vpc-123",
			})
			// No mock expectations set: CreateSecurityGroup / FindSecurityGroupsByTags must not be called.

			Expect(fc.ensureSecurityGroup(ctx)).To(Succeed())
			Expect(fc.state.Get(IdentifierNodesSecurityGroup)).To(Equal(ptr.To("sg-byo-456")))
		})
	})
})
