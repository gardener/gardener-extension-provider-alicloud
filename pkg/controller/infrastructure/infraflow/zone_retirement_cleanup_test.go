// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infraflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

func TestZoneRetirementCleansManagedResourcesInOrder(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.VPC.UseCustomRouteTable = ptr.To(true)
	state := retirementFlowState()
	state[IdentifierRouteTable] = "rtb-123"
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	var persisted shared.FlatMap
	setRetirementTestPersistor(fc, func(saved shared.FlatMap) { persisted = saved })

	vswitch := managedRetirementVSwitch(fc)
	inUse := &aliclient.EIP{
		EipId: "eip-retired", Name: "test-ns-eip-natgw-z1", IpAddress: "192.0.2.2", Status: ptr.To("InUse"),
		InstanceId: ptr.To("ngw-123"), InstanceType: ptr.To("Nat"),
	}
	available := &aliclient.EIP{
		EipId: "eip-retired", IpAddress: "192.0.2.2", Status: ptr.To("Available"),
	}
	entry := &aliclient.SNATEntry{VSwitchId: "vsw-retired", SnatEntryId: "snat-retired", SnatTableId: "snat-table"}

	gomock.InOrder(
		actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return([]*aliclient.SNATEntry{entry}, nil),
		actor.EXPECT().DeleteSNatEntry(gomock.Any(), "snat-retired", "snat-table").Return(nil),
		actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(vswitch, nil),
		actor.EXPECT().GetRouteTable(gomock.Any(), "rtb-123").Return(&aliclient.RouteTable{RouteTableId: "rtb-123", VSwitchIds: []string{"vsw-retired"}}, nil),
		actor.EXPECT().UnassociateRouteTable(gomock.Any(), "rtb-123", "vsw-retired").Return(nil),
		actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(vswitch, nil),
		actor.EXPECT().DeleteVSwitch(gomock.Any(), "vsw-retired").Return(nil),
		actor.EXPECT().GetEIP(gomock.Any(), "eip-retired").Return(inUse, nil),
		actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil),
		actor.EXPECT().UnAssociateEIP(gomock.Any(), inUse).Return(nil),
		actor.EXPECT().GetEIP(gomock.Any(), "eip-retired").Return(available, nil),
		actor.EXPECT().DeleteEIP(gomock.Any(), "eip-retired").Return(nil),
	)

	if err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{
		ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true, ManagedEIPID: "eip-retired",
	}}); err != nil {
		t.Fatal(err)
	}
	if persisted["Zones/zone2/RetirementCompleted"] != ZoneRetirementCompletedValue {
		t.Fatalf("persisted state: %v", persisted)
	}
	if persisted["Zones/zone2/VSwitch"] != "<deleted>" || persisted["Zones/zone2/NATGatewayElasticIP"] != "<deleted>" {
		t.Fatalf("resources were not marked deleted: %v", persisted)
	}
}

func TestZoneRetirementResumesFromOriginalStateAndCloudState(t *testing.T) {
	ctx := context.Background()
	target := zoneRetirementTarget{ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true}
	state := retirementFlowState()
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	var persisted shared.FlatMap
	setRetirementTestPersistor(fc, func(saved shared.FlatMap) { persisted = saved })
	actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return([]*aliclient.SNATEntry{{
		VSwitchId: "vsw-retired", SnatEntryId: "snat-retired", SnatTableId: "snat-table",
	}}, nil)
	actor.EXPECT().DeleteSNatEntry(gomock.Any(), "snat-retired", "snat-table").Return(errors.New("connection closed after request"))

	if err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{target}); err == nil {
		t.Fatal("expected the first cleanup attempt to fail")
	}
	if persisted["Zones/zone2/VSwitch"] != "vsw-retired" || persisted["Zones/zone2/RetirementCompleted"] != "" {
		t.Fatalf("original state was not retained after the failed deletion: %v", persisted)
	}

	resumed, resumedActor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), persisted)
	var final shared.FlatMap
	setRetirementTestPersistor(resumed, func(saved shared.FlatMap) { final = saved })
	resumedActor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil)
	resumedActor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(nil, nil)

	if err := resumed.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{target}); err != nil {
		t.Fatal(err)
	}
	if final["Zones/zone2/RetirementCompleted"] != ZoneRetirementCompletedValue {
		t.Fatalf("resumed state: %v", final)
	}
}

func TestZoneRetirementHandlesUserAndHistoricalManagedEIPSeparately(t *testing.T) {
	ctx := context.Background()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	setRetirementTestPersistor(fc, nil)
	managed := &aliclient.EIP{EipId: "eip-retired", Status: ptr.To("Available")}
	user := &aliclient.EIP{
		EipId: "eip-user", IpAddress: "192.0.2.9", Status: ptr.To("InUse"),
		InstanceId: ptr.To("ngw-123"), InstanceType: ptr.To("Nat"),
	}

	gomock.InOrder(
		actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil),
		actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(nil, nil),
		actor.EXPECT().GetEIP(gomock.Any(), "eip-retired").Return(managed, nil),
		actor.EXPECT().GetEIP(gomock.Any(), "eip-user").Return(user, nil),
		actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil),
		actor.EXPECT().UnAssociateEIP(gomock.Any(), user).Return(nil),
		actor.EXPECT().GetEIP(gomock.Any(), "eip-retired").Return(managed, nil),
		actor.EXPECT().DeleteEIP(gomock.Any(), "eip-retired").Return(nil),
	)

	if err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{
		ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true,
		ManagedEIPID: "eip-retired", UserEIPID: "eip-user",
	}}); err != nil {
		t.Fatal(err)
	}
}

func TestZoneRetirementKeepsEIPReferencedByActiveZone(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.Zones[0].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-retired")}
	fc, _ := newBYOTestContext(gomock.NewController(t), config, retirementFlowState())

	if err := fc.releaseRetirementManagedEIP(zoneRetirementTarget{ZoneName: "zone2", ManagedEIPID: "eip-retired"})(ctx); err != nil {
		t.Fatal(err)
	}
	if got := fc.getZoneChild("zone2").Get(IdentifierZoneNATGWElasticIP); got != nil {
		t.Fatalf("retired zone still references shared EIP: %s", *got)
	}
	if got := config.Networks.Zones[0].NatGateway.EIPAllocationID; got == nil || *got != "eip-retired" {
		t.Fatalf("active zone configuration changed: %v", got)
	}
}

func TestZoneRetirementDoesNotDeleteVSwitchAfterRouteFailure(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.VPC.UseCustomRouteTable = ptr.To(true)
	state := retirementFlowState()
	state[IdentifierRouteTable] = "rtb-123"
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	setRetirementTestPersistor(fc, nil)
	actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil)
	actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(managedRetirementVSwitch(fc), nil)
	actor.EXPECT().GetRouteTable(gomock.Any(), "rtb-123").Return(&aliclient.RouteTable{VSwitchIds: []string{"vsw-retired"}}, nil)
	actor.EXPECT().UnassociateRouteTable(gomock.Any(), "rtb-123", "vsw-retired").Return(errors.New("route table busy"))

	err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{
		ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true,
	}})
	if err == nil || !strings.Contains(err.Error(), "route table busy") {
		t.Fatalf("expected route failure, got %v", err)
	}
	if fc.isZoneRetirementCompleted("zone2") {
		t.Fatal("failed retirement was marked completed")
	}
}

func TestZoneRetirementKeepsVSwitchStateWhenVSwitchIsStillInUse(t *testing.T) {
	ctx := context.Background()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	setRetirementTestPersistor(fc, nil)
	actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil)
	actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(managedRetirementVSwitch(fc), nil)
	actor.EXPECT().DeleteVSwitch(gomock.Any(), "vsw-retired").Return(errors.New("ResourceInUse"))

	err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{
		ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true,
	}})
	if err == nil || !strings.Contains(err.Error(), "ResourceInUse") {
		t.Fatalf("expected VSwitch usage error, got %v", err)
	}
	if fc.isZoneRetirementCompleted("zone2") {
		t.Fatal("failed retirement was marked completed")
	}
	if got := fc.getZoneChild("zone2").Get(IdentifierZoneVSwitch); got == nil || *got != "vsw-retired" {
		t.Fatalf("VSwitch state was removed: %v", got)
	}
}

func TestZoneRetirementDeletesVSwitchBeforeReportingEIPFailure(t *testing.T) {
	ctx := context.Background()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	setRetirementTestPersistor(fc, nil)
	gomock.InOrder(
		actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil),
		actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(nil, nil),
		actor.EXPECT().GetEIP(gomock.Any(), "eip-retired").Return(nil, errors.New("EIP API unavailable")),
	)

	err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{
		ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true, ManagedEIPID: "eip-retired",
	}})
	if err == nil || !strings.Contains(err.Error(), "EIP API unavailable") {
		t.Fatalf("expected EIP failure, got %v", err)
	}
	if !fc.getZoneChild("zone2").IsAlreadyDeleted(IdentifierZoneVSwitch) {
		t.Fatal("VSwitch progress was not retained before EIP cleanup")
	}
	if fc.isZoneRetirementCompleted("zone2") {
		t.Fatal("retirement was marked completed despite EIP failure")
	}
}

func TestZoneRetirementRecoversWhenCompletedCannotBePersisted(t *testing.T) {
	ctx := context.Background()
	target := zoneRetirementTarget{ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: true}
	state := retirementFlowState()
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	var persisted shared.FlatMap
	fc.BasicFlowContext = *shared.NewBasicFlowContext(logr.Discard(), fc.state, func(_ context.Context, state shared.FlatMap) error {
		if state["Zones/zone2/RetirementCompleted"] == ZoneRetirementCompletedValue {
			return errors.New("cannot persist Completed")
		}
		persisted = cloneFlatMap(state)
		return nil
	})
	actor.EXPECT().FindSNatEntriesByNatGateway(gomock.Any(), "ngw-123").Return(nil, nil)
	actor.EXPECT().GetVSwitch(gomock.Any(), "vsw-retired").Return(nil, nil)

	err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{target})
	if err == nil || !strings.Contains(err.Error(), "cannot persist Completed") {
		t.Fatalf("expected Completed persistence error, got %v", err)
	}
	if persisted["Zones/zone2/VSwitch"] != "<deleted>" || persisted["Zones/zone2/RetirementCompleted"] != "" {
		t.Fatalf("resource progress was not persisted independently: %v", persisted)
	}

	resumed, _ := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), persisted)
	var final shared.FlatMap
	setRetirementTestPersistor(resumed, func(saved shared.FlatMap) { final = saved })
	if err := resumed.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{ZoneName: "zone2", ManagedVSwitch: true}}); err != nil {
		t.Fatal(err)
	}
	if final["Zones/zone2/RetirementCompleted"] != ZoneRetirementCompletedValue {
		t.Fatalf("resumed state: %v", final)
	}
}

func TestZoneRetirementProcessesZonesSerially(t *testing.T) {
	ctx := context.Background()
	config := byoConfig(
		byoZone("zone1", "vsw-active"),
		byoZone("zone2", "vsw-second"),
		byoZone("zone3", "vsw-third"),
	)
	config.Networks.Zones[1].Retired = true
	config.Networks.Zones[2].Retired = true
	state := shared.FlatMap{
		IdentifierVPC: "vpc-123", "Zones/zone2/VSwitch": "vsw-second", "Zones/zone3/VSwitch": "vsw-third",
	}
	fc, _ := newBYOTestContext(gomock.NewController(t), config, state)
	var snapshots []shared.FlatMap
	setRetirementTestPersistor(fc, func(saved shared.FlatMap) { snapshots = append(snapshots, saved) })

	if err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{
		{ZoneName: "zone2", VSwitchID: "vsw-second"},
		{ZoneName: "zone3", VSwitchID: "vsw-third"},
	}); err != nil {
		t.Fatal(err)
	}
	firstZoneCompletedAlone := false
	for _, snapshot := range snapshots {
		if snapshot["Zones/zone2/RetirementCompleted"] == ZoneRetirementCompletedValue && snapshot["Zones/zone3/RetirementCompleted"] == "" {
			firstZoneCompletedAlone = true
			break
		}
	}
	if !firstZoneCompletedAlone {
		t.Fatalf("zone3 started before zone2 completed: %v", snapshots)
	}
	if !fc.isZoneRetirementCompleted("zone2") || !fc.isZoneRetirementCompleted("zone3") {
		t.Fatal("not all zones completed")
	}
}

func TestBYOZoneRetirementOnlyClearsShootReference(t *testing.T) {
	ctx := context.Background()
	config := byoConfig(byoZone("zone1", "vsw-active"), byoZone("zone2", "vsw-retired"))
	config.Networks.Zones[1].Retired = true
	state := shared.FlatMap{IdentifierVPC: "vpc-123", "Zones/zone2/VSwitch": "vsw-retired"}
	fc, _ := newBYOTestContext(gomock.NewController(t), config, state)
	setRetirementTestPersistor(fc, nil)

	if err := fc.runZoneRetirementCleanup(ctx, []zoneRetirementTarget{{
		ZoneName: "zone2", VSwitchID: "vsw-retired", ManagedVSwitch: false,
	}}); err != nil {
		t.Fatal(err)
	}
	if !fc.getZoneChild("zone2").IsAlreadyDeleted(IdentifierZoneVSwitch) {
		t.Fatal("BYO VSwitch reference was not cleared")
	}
	if !fc.isZoneRetirementCompleted("zone2") {
		t.Fatal("BYO retirement was not marked completed")
	}
}

func TestCompletedZoneIsNotDecomposedAgain(t *testing.T) {
	state := retirementFlowState()
	state["Zones/zone2/RetirementCompleted"] = ZoneRetirementCompletedValue
	fc, _ := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)

	targets, err := fc.buildZoneRetirementTargets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 0 {
		t.Fatalf("completed zone was decomposed again: %+v", targets)
	}
}

func TestDeletedVSwitchStateIsNotRediscoveredDuringRecovery(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	state["Zones/zone2/VSwitch"] = "<deleted>"
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	fc.retirementObserver = &recordingRetirementObserver{}
	actor.EXPECT().GetNatGateway(ctx, "ngw-123").Return(&aliclient.NatGateway{
		NatGatewayId: "ngw-123", VswitchId: ptr.To("vsw-active"),
	}, nil)

	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].VSwitchID != "" {
		t.Fatalf("deleted VSwitch was rediscovered: %+v", targets)
	}
}

func TestZoneRetirementRejectsEIPWithoutStatus(t *testing.T) {
	ctx := context.Background()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(&aliclient.EIP{EipId: "eip-retired"}, nil)

	err := fc.cleanupRetirementEIPAssociation(ctx, zoneRetirementTarget{ZoneName: "zone2"}, "eip-retired", true)
	if err == nil || !strings.Contains(err.Error(), "has no status") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestZoneRetirementHandlesUnknownSNATUsingEIPByOwnership(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name    string
		managed bool
		wantErr bool
	}{
		{name: "managed EIP blocks retirement", managed: true, wantErr: true},
		{name: "user EIP remains associated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			eip := &aliclient.EIP{
				EipId: "eip-retired", IpAddress: "192.0.2.2", Status: ptr.To("InUse"),
				InstanceId: ptr.To("ngw-123"), InstanceType: ptr.To("Nat"),
			}
			fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
			actor.EXPECT().GetEIP(ctx, "eip-retired").Return(eip, nil)
			actor.EXPECT().FindSNatEntriesByNatGateway(ctx, "ngw-123").Return([]*aliclient.SNATEntry{{
				SnatEntryId: "snat-unknown", IpAddress: "192.0.2.2",
			}}, nil)

			err := fc.cleanupRetirementEIPAssociation(ctx, zoneRetirementTarget{ZoneName: "zone2"}, "eip-retired", test.managed)
			if test.wantErr {
				if err == nil || !strings.Contains(err.Error(), "managed EIP eip-retired is still used by SNAT entry snat-unknown") {
					t.Fatalf("expected managed EIP usage error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeleteRetiringZoneResidualsCleansManagedEIPAfterRetirementDeletedVSwitch(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	state["Zones/zone2/VSwitch"] = "<deleted>"
	delete(state, "Zones/zone1/VSwitch")
	delete(state, "Zones/zone1/Suffix")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	available := &aliclient.EIP{EipId: "eip-retired", Status: ptr.To("Available")}

	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(available, nil)
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(available, nil)
	actor.EXPECT().DeleteEIP(ctx, "eip-retired").Return(nil)

	if err := fc.deleteRetiringZoneResiduals(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if fc.state.GetChild(ChildIdZones).HasChild("zone2") {
		t.Fatal("retired zone state was retained after whole-Shoot deletion")
	}
}

func TestDeleteRetiringZoneResidualsKeepsEIPReferencedByActiveZone(t *testing.T) {
	config := retirementFlowConfig()
	config.Networks.Zones[0].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-retired")}
	state := retirementFlowState()
	state["Zones/zone2/VSwitch"] = "<deleted>"
	fc, _ := newBYOTestContext(gomock.NewController(t), config, state)

	if err := fc.deleteRetiringZoneResiduals(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if fc.state.GetChild(ChildIdZones).HasChild("zone2") {
		t.Fatal("retired zone state was retained after the active EIP reference was handed over")
	}
	if got := config.Networks.Zones[0].NatGateway.EIPAllocationID; got == nil || *got != "eip-retired" {
		t.Fatalf("active EIP reference changed: %v", got)
	}
}

func TestDeleteRetiringZoneResidualsHandlesUnpersistedVSwitchDeletion(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	delete(state, "Zones/zone1/VSwitch")
	delete(state, "Zones/zone1/Suffix")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	available := &aliclient.EIP{EipId: "eip-retired", Status: ptr.To("Available")}

	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(available, nil)
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(available, nil)
	actor.EXPECT().DeleteEIP(ctx, "eip-retired").Return(nil)

	if err := fc.deleteRetiringZoneResiduals(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if fc.state.GetChild(ChildIdZones).HasChild("zone2") {
		t.Fatal("zone with an absent cloud VSwitch retained residual state")
	}
}

func TestDeleteRetiringZoneResidualsUnassociatesUserEIP(t *testing.T) {
	ctx := context.Background()
	config := retirementFlowConfig()
	config.Networks.Zones[1].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-user")}
	state := retirementFlowState()
	state["Zones/zone2/VSwitch"] = "<deleted>"
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
	eip := &aliclient.EIP{
		EipId: "eip-user", IpAddress: "192.0.2.9", Status: ptr.To("InUse"),
		InstanceId: ptr.To("ngw-123"), InstanceType: ptr.To("Nat"),
	}

	actor.EXPECT().GetEIP(ctx, "eip-user").Return(eip, nil)
	actor.EXPECT().FindSNatEntriesByNatGateway(ctx, "ngw-123").Return(nil, nil)
	actor.EXPECT().UnAssociateEIP(ctx, eip).Return(nil)

	if err := fc.deleteRetiringZoneResiduals(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if fc.state.GetChild(ChildIdZones).HasChild("zone2") {
		t.Fatal("retired zone state was retained after user EIP cleanup")
	}
	if got := config.Networks.Zones[1].NatGateway.EIPAllocationID; got == nil || *got != "eip-user" {
		t.Fatalf("user EIP allocation changed: %v", got)
	}
}

func setRetirementTestPersistor(fc *FlowContext, record func(shared.FlatMap)) {
	fc.BasicFlowContext = *shared.NewBasicFlowContext(logr.Discard(), fc.state, func(_ context.Context, state shared.FlatMap) error {
		if record != nil {
			record(cloneFlatMap(state))
		}
		return nil
	})
}

func cloneFlatMap(state shared.FlatMap) shared.FlatMap {
	saved := make(shared.FlatMap, len(state))
	for key, value := range state {
		saved[key] = value
	}
	return saved
}

func managedRetirementVSwitch(fc *FlowContext) *aliclient.VSwitch {
	return &aliclient.VSwitch{
		VSwitchId: "vsw-retired", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24",
		Name: "test-ns-zone2-vsw", Tags: fc.clusterTags(),
	}
}
