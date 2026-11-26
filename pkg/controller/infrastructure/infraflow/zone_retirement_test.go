// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

type stateCheckingRetirementObserver struct {
	flowContext *FlowContext
}

func (o *stateCheckingRetirementObserver) EnsureVSwitchUnused(_ context.Context, _, vSwitchID string) error {
	if vSwitchID != "vsw-retired" {
		return fmt.Errorf("unexpected retirement target %s", vSwitchID)
	}
	if got := o.flowContext.getZoneChild("zone-active").Get(IdentifierZoneVSwitch); got == nil || *got != "vsw-active" {
		return fmt.Errorf("active zone was not reconciled before retirement")
	}
	if got := o.flowContext.state.Get(IdentifierRouteTable); got == nil || *got != "rt-system" {
		return fmt.Errorf("active route table was not reconciled before retirement")
	}
	return nil
}

type rejectVSwitchRetirementObserver struct {
	rejectedID string
}

func (o *rejectVSwitchRetirementObserver) EnsureVSwitchUnused(_ context.Context, _, vSwitchID string) error {
	if vSwitchID == o.rejectedID {
		return fmt.Errorf("VSwitch %s is still used", vSwitchID)
	}
	return nil
}

func retirementFlowConfig() *api.InfrastructureConfig {
	return &api.InfrastructureConfig{Networks: api.Networks{
		VPC:   api.VPC{CIDR: ptr.To("10.0.0.0/16")},
		Zones: []api.Zone{{Name: "zone1", Workers: "10.0.1.0/24"}, {Name: "zone2", Workers: "10.0.2.0/24", Retired: true}},
	}}
}

func retirementFlowState() shared.FlatMap {
	return shared.FlatMap{
		IdentifierVPC: "vpc-123", IdentifierNatGateway: "ngw-123",
		"Zones/zone1/VSwitch": "vsw-active", "Zones/zone1/Suffix": "z0",
		"Zones/zone2/VSwitch": "vsw-retired", "Zones/zone2/Suffix": "z1",
		"Zones/zone2/NATGatewayElasticIP":        "eip-retired",
		"Zones/zone2/NATGatewayElasticIPAddress": "192.0.2.2",
	}
}

func TestActiveAndTrackedVSwitchIDs(t *testing.T) {
	config := retirementFlowConfig()
	config.Networks.Zones[0].Retired, config.Networks.Zones[1].Retired = true, false
	config.Networks.Zones = append(config.Networks.Zones, api.Zone{Name: "zone3"}, api.Zone{Name: "zone4"})
	state := retirementFlowState()
	state["Zones/zone3/VSwitch"] = "vsw-third"
	state["Zones/zone4/VSwitch"] = "<deleted>"
	state["Zones/old-zone/VSwitch"] = "vsw-unknown"
	fc, _ := newBYOTestContext(gomock.NewController(t), config, state)
	if got := fc.getActiveVSwitchIDs(); !reflect.DeepEqual(got, []string{"vsw-retired", "vsw-third"}) {
		t.Fatalf("active IDs: %v", got)
	}
	if got := fc.getTrackedVSwitchIDs(); !reflect.DeepEqual(got, []string{"vsw-unknown", "vsw-active", "vsw-retired", "vsw-third"}) {
		t.Fatalf("tracked IDs: %v", got)
	}
	if !reflect.DeepEqual(fc.ExportState(), state) {
		t.Fatal("ID selection changed state")
	}
	fc.config.Networks.Zones = nil
	if got := fc.getActiveVSwitchIDs(); len(got) != 0 {
		t.Fatalf("empty config: %v", got)
	}
}

func TestReconcileRunsZoneRetirementAfterActiveBYOZones(t *testing.T) {
	ctx := context.Background()
	config := byoConfig(
		byoZone("zone-active", "vsw-active"),
		byoZone("zone-retired", "vsw-retired"),
	)
	config.Networks.Zones[1].Retired = true
	config.Networks.NodesSecurityGroupID = ptr.To("sg-user")
	fc, actor := newBYOTestContext(gomock.NewController(t), config, shared.FlatMap{
		IdentifierVPC:                                "vpc-123",
		"Zones/zone-retired/VSwitch":                 "vsw-retired",
		"Zones/zone-retired/" + IdentifierZoneSuffix: "z1",
	})
	fc.retirementObserver = &stateCheckingRetirementObserver{flowContext: fc}

	actor.EXPECT().GetVpc(gomock.Any(), "vpc-123").Return(&aliclient.VPC{VpcId: "vpc-123"}, nil)
	actor.EXPECT().ListRouteTablesByVPC(gomock.Any(), "vpc-123").Return([]*aliclient.RouteTable{{
		RouteTableId: "rt-system", RouteTableType: "System",
	}}, nil)

	if err := fc.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if !fc.isZoneRetirementCompleted("zone-retired") {
		t.Fatal("retired BYO zone was not retired by the main reconcile graph")
	}
}

func TestRetireZonesChecksAllTargetsBeforeCleanup(t *testing.T) {
	config := byoConfig(
		byoZone("zone-active", "vsw-active"),
		byoZone("zone-retired-a", "vsw-retired-a"),
		byoZone("zone-retired-b", "vsw-retired-b"),
	)
	config.Networks.Zones[1].Retired = true
	config.Networks.Zones[2].Retired = true
	state := shared.FlatMap{
		IdentifierVPC:                  "vpc-123",
		"Zones/zone-retired-a/VSwitch": "vsw-retired-a",
		"Zones/zone-retired-b/VSwitch": "vsw-retired-b",
	}
	fc, _ := newBYOTestContext(gomock.NewController(t), config, state)
	fc.retirementObserver = &rejectVSwitchRetirementObserver{rejectedID: "vsw-retired-b"}

	err := fc.retireZones(context.Background())
	if err == nil || !strings.Contains(err.Error(), "vsw-retired-b is still used") {
		t.Fatalf("unexpected preflight result: %v", err)
	}
	if got := fc.getZoneChild("zone-retired-a").Get(IdentifierZoneVSwitch); got == nil || *got != "vsw-retired-a" {
		t.Fatalf("first target was changed before all preflights passed: %v", got)
	}
	if fc.isZoneRetirementCompleted("zone-retired-a") {
		t.Fatal("first target was completed before the later preflight failed")
	}
}

func TestEnsureVSwitchesDefersRetirement(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name                   string
		canDelete, cloudExists bool
	}{
		{name: "protected", cloudExists: true},
		{name: "generic deletion enabled", canDelete: true, cloudExists: true},
		{name: "already absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := retirementFlowState()
			fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
			fc.canDelete = tc.canDelete
			active := &aliclient.VSwitch{VSwitchId: "vsw-active", ZoneId: "zone1", CidrBlock: "10.0.1.0/24", VpcId: ptr.To("vpc-123")}
			current := []*aliclient.VSwitch{active}
			tagged := current
			if tc.cloudExists {
				retiring := &aliclient.VSwitch{
					VSwitchId: "vsw-retired", ZoneId: "zone2", CidrBlock: "10.0.2.0/24", Name: "test-ns-zone2-vsw", Tags: fc.commonTagsWithSuffix("nodes-z1"),
				}
				current = append(current, retiring)
				tagged = current
			}
			actor.EXPECT().ListVSwitches(ctx, []string{"vsw-active", "vsw-retired"}).Return(current, nil)
			actor.EXPECT().FindVSwitchesByTags(ctx, fc.clusterTags()).Return(tagged, nil)
			actor.EXPECT().FindVSwitchesByVPC(ctx, "vpc-123").Return(current, nil)
			// Active maintenance still runs; no creation, deletion, or mutation of the retired resource is expected.
			actor.EXPECT().CreateTags(ctx, []string{"vsw-active"}, fc.commonTagsWithSuffix("nodes-z0"), "VSWITCH").Return(nil)
			if err := fc.ensureVSwitches(ctx); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(fc.ExportState(), state) {
				t.Fatalf("retained state changed: %v", fc.ExportState())
			}
		})
	}
}

func TestConflictingSharedVPCVSwitchIsNotAdoptedForRetirement(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	delete(state, "Zones/zone2/VSwitch")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	active := &aliclient.VSwitch{VSwitchId: "vsw-active", ZoneId: "zone1", CidrBlock: "10.0.1.0/24"}
	foreign := &aliclient.VSwitch{
		VSwitchId: "vsw-foreign", ZoneId: "zone2", CidrBlock: "10.0.2.0/24", Name: "other-shoot-zone2-vsw",
		Tags: aliclient.Tags{"kubernetes.io/cluster/other-shoot": TagValueCluster},
	}
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-active"}).Return([]*aliclient.VSwitch{active}, nil)
	actor.EXPECT().FindVSwitchesByTags(ctx, fc.clusterTags()).Return([]*aliclient.VSwitch{active}, nil)
	actor.EXPECT().FindVSwitchesByVPC(ctx, "vpc-123").Return([]*aliclient.VSwitch{active, foreign}, nil)
	actor.EXPECT().CreateTags(ctx, []string{"vsw-active"}, fc.commonTagsWithSuffix("nodes-z0"), "VSWITCH").Return(nil)

	if err := fc.ensureVSwitches(ctx); err != nil {
		t.Fatal(err)
	}
	if got := fc.getZoneChild("zone2").Get(IdentifierZoneVSwitch); got != nil {
		t.Fatalf("adopted foreign VSwitch %s", *got)
	}
	// No delete expectation is installed: the foreign VSwitch must be ignored.
}

func TestUnknownVSwitchStillProtected(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	active := &aliclient.VSwitch{VSwitchId: "vsw-active", ZoneId: "zone1", CidrBlock: "10.0.1.0/24"}
	// Sharing a retired zone is insufficient to exempt a resource from deletion protection.
	unknown := &aliclient.VSwitch{VSwitchId: "vsw-unknown", ZoneId: "zone2", CidrBlock: "10.0.9.0/24", Tags: fc.clusterTags()}
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-active", "vsw-retired"}).Return([]*aliclient.VSwitch{active}, nil)
	actor.EXPECT().FindVSwitchesByTags(ctx, fc.clusterTags()).Return([]*aliclient.VSwitch{active, unknown}, nil)
	actor.EXPECT().FindVSwitchesByVPC(ctx, "vpc-123").Return(nil, nil)
	if err := fc.ensureVSwitches(ctx); err == nil || !strings.Contains(err.Error(), "protected:") {
		t.Fatalf("expected protection, got %v", err)
	}
	if !reflect.DeepEqual(fc.ExportState(), state) {
		t.Fatal("protection changed tracked state")
	}
}

func TestNewZoneKeepsSuffixAndIPv6Index(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.DualStack = &api.DualStack{Enabled: true}
	config.Networks.Zones = append(config.Networks.Zones, api.Zone{Name: "zone3", Workers: "10.0.3.0/24"})
	state := retirementFlowState()
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	active := &aliclient.VSwitch{VSwitchId: "vsw-active", ZoneId: "zone1", CidrBlock: "10.0.1.0/24", Tags: fc.commonTagsWithSuffix("nodes-z0")}
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-active", "vsw-retired"}).Return([]*aliclient.VSwitch{active}, nil)
	actor.EXPECT().FindVSwitchesByTags(ctx, fc.clusterTags()).Return(nil, nil)
	actor.EXPECT().FindVSwitchesByVPC(ctx, "vpc-123").Return(nil, nil)
	actor.EXPECT().CreateVSwitch(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, desired *aliclient.VSwitch) (*aliclient.VSwitch, error) {
		if desired.ZoneId != "zone3" || desired.Tags[TagKeyName] != "test-ns-nodes-z2" {
			t.Fatalf("wrong new VSwitch: %+v", desired)
		}
		created := *desired
		created.VSwitchId = "vsw-new"
		return &created, nil
	})
	actor.EXPECT().SetVSwitchIpv6CidrBlock(ctx, "vsw-new", 2).Return(nil)
	actor.EXPECT().GetVSwitchIpv6CidrBlock(ctx, "vsw-new").Return("2001:db8:0:2::/64", nil)
	actor.EXPECT().GetVSwitchIpv6CidrBlock(ctx, "vsw-active").Return("2001:db8::/64", nil)
	if err := fc.ensureVSwitches(ctx); err != nil {
		t.Fatal(err)
	}
	for key, value := range state {
		if fc.ExportState()[key] != value {
			t.Fatalf("changed %s", key)
		}
	}
	if got := fc.getZoneSuffix("zone3"); got != "z2" {
		t.Fatalf("new suffix: %s", got)
	}
}

func TestEnsureZonesOnlyMaintainsActiveEgress(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	// User NGW mode supports retiring the first configured zone.
	config.Networks.VPC = api.VPC{ID: ptr.To("vpc-123")}
	config.Networks.Zones[0], config.Networks.Zones[1] = config.Networks.Zones[1], config.Networks.Zones[0]
	config.Networks.Zones[0].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-retired")}
	config.Networks.Zones[1].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-active")}
	state := retirementFlowState()
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	retiredEIP := &aliclient.EIP{EipId: "eip-retired", InternetChargeType: "PayByBandwidth"}
	eip := &aliclient.EIP{EipId: "eip-active", IpAddress: "192.0.2.1", InternetChargeType: "PayByTraffic", Status: ptr.To("Available")}
	// The retired user EIP is queried only as the Shoot-wide charge type source.
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(retiredEIP, nil)
	actor.EXPECT().GetEIP(gomock.Any(), "eip-active").Return(eip, nil).Times(3)
	actor.EXPECT().AssociateEIP(gomock.Any(), "eip-active", "ngw-123", "Nat").Return(nil)
	actor.EXPECT().GetNatGateway(gomock.Any(), "ngw-123").Return(&aliclient.NatGateway{NatGatewayId: "ngw-123", SNATTableIDs: []string{"snat-table"}}, nil)
	actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return([]*aliclient.SNATEntry{{VSwitchId: "vsw-retired", IpAddress: "192.0.2.2", SnatTableId: "snat-table", SnatEntryId: "snat-retired"}}, nil)
	actor.EXPECT().CreateSNatEntry(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, desired *aliclient.SNATEntry) (*aliclient.SNATEntry, error) {
		if desired.VSwitchId != "vsw-active" || desired.IpAddress != "192.0.2.1" {
			t.Errorf("wrong SNAT: %+v", desired)
		}
		created := *desired
		created.SnatEntryId = "snat-active"
		return &created, nil
	})
	if err := fc.ensureZones(ctx); err != nil {
		t.Fatal(err)
	}
	for key, value := range state {
		if fc.ExportState()[key] != value {
			t.Fatalf("changed %s", key)
		}
	}
	if fc.ExportState()["Zones/zone1/NATGatewayElasticIPAddress"] != "192.0.2.1" {
		t.Fatal("active egress not recorded")
	}
	// No retired-zone EIP/SNAT maintenance or delete call is expected.
}

func TestEIPChargeTypeUsesConfiguredEIPFromRetiredZone(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.Zones[1].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-retired")}
	fc, actor := newBYOTestContext(gomock.NewController(t), config, nil)
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(&aliclient.EIP{
		EipId:              "eip-retired",
		InternetChargeType: "PayByBandwidth",
	}, nil)

	if got := fc.getEipInternetChargeType(ctx); got != "PayByBandwidth" {
		t.Fatalf("charge type = %s, want PayByBandwidth", got)
	}
}

func TestRouteTableOnlyAssociatesActiveVSwitches(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.VPC.UseCustomRouteTable = ptr.To(true)
	state := retirementFlowState()
	state[IdentifierRouteTable] = "rtb-123"
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	actor.EXPECT().GetRouteTable(ctx, "rtb-123").Return(&aliclient.RouteTable{RouteTableId: "rtb-123", Tags: fc.commonTagsWithSuffix("rt")}, nil)
	actor.EXPECT().FindRouteEntryByDest(ctx, "rtb-123", "0.0.0.0/0").Return(&aliclient.RouteEntry{NextHopId: "ngw-123"}, nil)
	actor.EXPECT().AssociateRouteTable(ctx, "rtb-123", "vsw-active").Return(nil)
	if err := fc.ensureRouteTable(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fc.ExportState(), state) {
		t.Fatal("route maintenance changed tracked targets")
	}
}

func TestManagedNATCandidatesFollowActiveConfiguration(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.Zones = append(config.Networks.Zones, api.Zone{Name: "zone3"})
	state := retirementFlowState()
	delete(state, IdentifierNatGateway)
	state["Zones/zone3/VSwitch"] = "vsw-third"
	state["Zones/unknown/VSwitch"] = "vsw-unknown"
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	actor.EXPECT().FindNatGatewayByTags(ctx, fc.commonTagsWithSuffix("natgw")).Return(nil, nil)
	actor.EXPECT().ListNatGatewaysByVSwitchInVPC(ctx, "vpc-123", []string{"vsw-active", "vsw-third"}).Return(nil, nil)
	actor.EXPECT().CreateNatGateway(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, desired *aliclient.NatGateway) (*aliclient.NatGateway, error) {
		if !reflect.DeepEqual(desired.AvailableVSwitches, []string{"vsw-active", "vsw-third"}) {
			t.Fatalf("candidates: %v", desired.AvailableVSwitches)
		}
		created := *desired
		created.NatGatewayId = "ngw-new"
		created.VswitchId = ptr.To("vsw-active")
		return &created, nil
	})
	if err := fc.ensureNatGateway(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestFullDiscoveryKeepsRetiredVSwitches(t *testing.T) {
	ctx := context.Background()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-active", "vsw-retired"}).Return([]*aliclient.VSwitch{{VSwitchId: "vsw-active"}, {VSwitchId: "vsw-retired"}}, nil)
	actor.EXPECT().FindVSwitchesByTags(ctx, fc.clusterTags()).Return(nil, nil)
	current, err := fc.collectExistingVSwitches(ctx)
	if err != nil || len(current) != 2 {
		t.Fatalf("full discovery: %v, %v", current, err)
	}
}

func TestBYORetirementKeepsStateAndRecomputesRoutes(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		tables []*aliclient.RouteTable
		want   string
	}{
		{name: "separate routes", tables: []*aliclient.RouteTable{{RouteTableId: "rt-retired", VSwitchIds: []string{"vsw-retired"}}, {RouteTableId: "rt-active", VSwitchIds: []string{"vsw-active"}}}, want: "rt-active"},
		{name: "shared route", tables: []*aliclient.RouteTable{{RouteTableId: "rt-shared", VSwitchIds: []string{"vsw-retired", "vsw-active"}}}, want: "rt-shared"},
		{name: "unexpected empty result keeps previous route", tables: []*aliclient.RouteTable{{RouteTableId: "rt-retired", VSwitchIds: []string{"vsw-retired"}}}, want: "rt-retired"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := byoConfig(byoZone("zone2", "vsw-retired"), byoZone("zone1", "vsw-active"))
			config.Networks.Zones[0].Retired = true
			state := retirementFlowState()
			state[IdentifierRouteTable] = "rt-retired"
			fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
			actor.EXPECT().ListRouteTablesByVPC(ctx, "vpc-123").Return(tc.tables, nil)
			if err := fc.ensureZones(ctx); err != nil {
				t.Fatal(err)
			}
			expected := retirementFlowState()
			expected[IdentifierRouteTable] = tc.want
			if !reflect.DeepEqual(fc.ExportState(), expected) {
				t.Fatalf("BYO state: %v", fc.ExportState())
			}
			if !reflect.DeepEqual(fc.getActiveVSwitchIDs(), []string{"vsw-active"}) {
				t.Fatal("retired first zone was selected")
			}
		})
	}
}

func TestUserNGWRetirementReusesRecordedGateway(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.VPC = api.VPC{ID: ptr.To("vpc-123")}
	config.Networks.Zones[0], config.Networks.Zones[1] = config.Networks.Zones[1], config.Networks.Zones[0]
	state := retirementFlowState()
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	actor.EXPECT().GetNatGateway(ctx, "ngw-123").Return(&aliclient.NatGateway{NatGatewayId: "ngw-123", VswitchId: ptr.To("vsw-user-home")}, nil)
	if err := fc.ensureNatGateway(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fc.ExportState(), state) {
		t.Fatal("user gateway state changed")
	}
}
