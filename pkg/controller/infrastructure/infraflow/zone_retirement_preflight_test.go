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

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

type recordingRetirementObserver struct {
	calls []string
	err   error
}

func (o *recordingRetirementObserver) EnsureVSwitchUnused(_ context.Context, namespace, vSwitchID string) error {
	o.calls = append(o.calls, namespace+"/"+vSwitchID)
	return o.err
}

func TestBuildZoneRetirementTargetsRequiresObserverOnlyForRetirement(t *testing.T) {
	fc, _ := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())

	targets, err := fc.buildZoneRetirementTargets(context.Background())
	if err == nil || !strings.Contains(err.Error(), "zone retirement observer is nil") || targets != nil {
		t.Fatalf("targets=%v err=%v", targets, err)
	}

	fc.config.Networks.Zones[1].Retired = false
	targets, err = fc.buildZoneRetirementTargets(context.Background())
	if err != nil || targets != nil {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
}

func TestBuildManagedZoneRetirementTarget(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	observer := &recordingRetirementObserver{}
	fc.retirementObserver = observer

	status := "Available"
	ownedTags := fc.commonTagsWithSuffix("nodes-z1")
	ownedTags[fc.tagKeyCluster()] = "owned"
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-retired"}).Return([]*aliclient.VSwitch{{
		VSwitchId: "vsw-retired", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24", Status: &status,
		Tags: ownedTags,
	}}, nil)
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(&aliclient.EIP{
		EipId: "eip-retired", Tags: ownedTags,
	}, nil)
	actor.EXPECT().GetNatGateway(ctx, "ngw-123").Return(&aliclient.NatGateway{
		NatGatewayId: "ngw-123", VswitchId: ptr.To("vsw-active"),
	}, nil)

	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets: %+v", targets)
	}
	target := targets[0]
	if target.ZoneName != "zone2" || target.VSwitchID != "vsw-retired" || !target.ManagedVSwitch {
		t.Fatalf("target: %+v", target)
	}
	if target.ManagedEIPID != "eip-retired" {
		t.Fatalf("egress target: %+v", target)
	}
	if !reflect.DeepEqual(observer.calls, []string{"test-ns/vsw-retired"}) {
		t.Fatalf("observer calls: %v", observer.calls)
	}
	if !reflect.DeepEqual(fc.ExportState(), state) {
		t.Fatal("target building changed state")
	}
}

func TestBuildZoneRetirementTargetRecoversOwnedVSwitchWithoutStateID(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	delete(state, "Zones/zone2/VSwitch")
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	observer := &recordingRetirementObserver{}
	fc.retirementObserver = observer

	actor.EXPECT().FindVSwitchesByVPC(ctx, "vpc-123").Return([]*aliclient.VSwitch{{
		VSwitchId: "vsw-recovered", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24",
		Name: "test-ns-zone2-vsw",
	}}, nil)
	actor.EXPECT().GetNatGateway(ctx, "ngw-123").Return(&aliclient.NatGateway{
		NatGatewayId: "ngw-123", VswitchId: ptr.To("vsw-active"),
	}, nil)

	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].VSwitchID != "vsw-recovered" {
		t.Fatalf("targets: %+v", targets)
	}
	if got := fc.getZoneChild("zone2").Get(IdentifierZoneVSwitch); got != nil {
		t.Fatalf("target discovery wrote state: %s", *got)
	}
}

func TestBuildZoneRetirementTargetContinuesWhenVSwitchAndStateIDAreAbsent(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	delete(state, "Zones/zone2/VSwitch")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	observer := &recordingRetirementObserver{}
	fc.retirementObserver = observer

	actor.EXPECT().FindVSwitchesByVPC(ctx, "vpc-123").Return(nil, nil)
	actor.EXPECT().GetEIP(ctx, "eip-retired").Return(&aliclient.EIP{
		EipId: "eip-retired", Tags: fc.clusterTags(),
	}, nil)
	actor.EXPECT().GetNatGateway(ctx, "ngw-123").Return(&aliclient.NatGateway{
		NatGatewayId: "ngw-123", VswitchId: ptr.To("vsw-active"),
	}, nil)

	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].VSwitchID != "" || targets[0].ManagedEIPID != "eip-retired" {
		t.Fatalf("targets: %+v", targets)
	}
	if len(observer.calls) != 0 {
		t.Fatalf("observer calls: %v", observer.calls)
	}
}

func TestBuildZoneRetirementTargetRejectsForeignVSwitch(t *testing.T) {
	ctx := context.Background()
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	fc.retirementObserver = &recordingRetirementObserver{}
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-retired"}).Return([]*aliclient.VSwitch{{
		VSwitchId: "vsw-retired", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24",
		Tags: aliclient.Tags{"kubernetes.io/cluster/other-shoot": TagValueCluster},
	}}, nil)

	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err == nil || !strings.Contains(err.Error(), "ownership cannot be verified") {
		t.Fatalf("expected ownership error, got targets=%v err=%v", targets, err)
	}
	if targets != nil {
		t.Fatalf("failed preflight returned executable targets: %+v", targets)
	}
}

func TestBuildZoneRetirementTargetBlocksManagedNatGatewayLocation(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	fc.retirementObserver = &recordingRetirementObserver{}
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-retired"}).Return([]*aliclient.VSwitch{{
		VSwitchId: "vsw-retired", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24",
		Tags: fc.clusterTags(),
	}}, nil)
	actor.EXPECT().GetNatGateway(ctx, "ngw-123").Return(&aliclient.NatGateway{
		NatGatewayId: "ngw-123", VswitchId: ptr.To("vsw-retired"),
	}, nil)

	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err == nil || !strings.Contains(err.Error(), "managed NatGateway ngw-123 is located in target VSwitch") {
		t.Fatalf("expected NatGateway location error, got targets=%v err=%v", targets, err)
	}
}

func TestUserNatGatewayAndBYORetirementTargetsDoNotApplyManagedLocationCheck(t *testing.T) {
	ctx := context.Background()
	t.Run("user NatGateway", func(t *testing.T) {
		config := retirementFlowConfig()
		config.Networks.VPC.ID = ptr.To("vpc-123")
		config.Networks.Zones[1].NatGateway = &api.NatGatewayConfig{EIPAllocationID: ptr.To("eip-user")}
		state := retirementFlowState()
		fc, actor := newBYOTestContext(gomock.NewController(t), config, state)
		fc.retirementObserver = &recordingRetirementObserver{}
		actor.EXPECT().ListVSwitches(ctx, []string{"vsw-retired"}).Return([]*aliclient.VSwitch{{
			VSwitchId: "vsw-retired", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24", Tags: fc.clusterTags(),
		}}, nil)
		actor.EXPECT().GetEIP(ctx, "eip-retired").Return(&aliclient.EIP{
			EipId: "eip-retired", Name: "test-ns-eip-natgw-z1",
		}, nil)

		targets, err := fc.buildZoneRetirementTargets(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 || targets[0].UserEIPID != "eip-user" || targets[0].ManagedEIPID != "eip-retired" {
			t.Fatalf("targets: %+v", targets)
		}
	})

	t.Run("BYO VSwitch", func(t *testing.T) {
		config := byoConfig(byoZone("zone1", "vsw-active"), byoZone("zone2", "vsw-retired"))
		config.Networks.Zones[1].Retired = true
		state := shared.FlatMap{
			IdentifierVPC: "vpc-123", "Zones/zone1/VSwitch": "vsw-active", "Zones/zone2/VSwitch": "vsw-retired",
		}
		fc, _ := newBYOTestContext(gomock.NewController(t), config, state)
		observer := &recordingRetirementObserver{}
		fc.retirementObserver = observer

		targets, err := fc.buildZoneRetirementTargets(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 || targets[0].ManagedVSwitch || targets[0].VSwitchID != "vsw-retired" {
			t.Fatalf("targets: %+v", targets)
		}
	})
}

func TestBuildZoneRetirementTargetsRejectsObserverFailure(t *testing.T) {
	ctx := context.Background()
	state := retirementFlowState()
	delete(state, "Zones/zone2/NATGatewayElasticIP")
	fc, actor := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), state)
	fc.retirementObserver = &recordingRetirementObserver{err: fmt.Errorf("Machine still exists")}
	actor.EXPECT().ListVSwitches(ctx, []string{"vsw-retired"}).Return([]*aliclient.VSwitch{{
		VSwitchId: "vsw-retired", VpcId: ptr.To("vpc-123"), ZoneId: "zone2", CidrBlock: "10.0.2.0/24", Tags: fc.clusterTags(),
	}}, nil)
	targets, err := fc.buildZoneRetirementTargets(ctx)
	if err == nil || !strings.Contains(err.Error(), "Machine still exists") || targets != nil {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
}

func TestWorkerSpecStillReferencingRetiringZoneIsRejected(t *testing.T) {
	fc, _ := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	fc.cluster = &extensioncontroller.Cluster{Shoot: &gardencorev1beta1.Shoot{
		Spec: gardencorev1beta1.ShootSpec{
			Provider: gardencorev1beta1.Provider{Workers: []gardencorev1beta1.Worker{{Name: "old-pool", Zones: []string{"zone2"}}}},
		},
	}}
	err := fc.ensureWorkerSpecDoesNotUseZone("zone2")
	if err == nil || !strings.Contains(err.Error(), "worker pool old-pool still references retiring zone zone2") {
		t.Fatalf("expected worker reference error, got %v", err)
	}
}

func TestWorkerSpecCheckProtectsNilShoot(t *testing.T) {
	fc, _ := newBYOTestContext(gomock.NewController(t), retirementFlowConfig(), retirementFlowState())
	fc.cluster.Shoot = nil

	err := fc.ensureWorkerSpecDoesNotUseZone("zone2")
	if err == nil || !strings.Contains(err.Error(), "cluster or Shoot is nil") {
		t.Fatalf("expected nil protection error, got %v", err)
	}
}
