// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"reflect"
	"sort"
	"testing"

	"k8s.io/utils/ptr"

	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

func TestZoneRetirementProviderOutputs(t *testing.T) {
	config := &aliapi.InfrastructureConfig{Networks: aliapi.Networks{
		VPC: aliapi.VPC{ID: ptr.To("vpc-123")},
		Zones: []aliapi.Zone{
			{Name: "zone-retired", Retired: true},
			{Name: "zone-active", NatGateway: &aliapi.NatGatewayConfig{EIPAllocationID: ptr.To("eip-user")}},
		},
	}}
	state := infraflow.NewPersistentStateFromFlatMap(shared.FlatMap{
		"Zones/zone-retired/VSwitch":                    "vsw-retired",
		"Zones/zone-retired/NATGatewayElasticIPAddress": "192.0.2.1",
		"Zones/zone-active/VSwitch":                     "vsw-active",
		"Zones/zone-active/NATGatewayElasticIPAddress":  "192.0.2.2",
	})

	status, err := computeProviderStatusFromFlowState(config, state)
	if err != nil {
		t.Fatal(err)
	}
	if got := status.VPC.VSwitches; len(got) != 1 || got[0].Zone != "zone-active" {
		t.Fatalf("provider VSwitches do not contain only active zones: %+v", got)
	}
	if got := getEgressIpCidrs(state); !equalStringsIgnoringOrder(got, []string{"192.0.2.1/32", "192.0.2.2/32"}) {
		t.Fatalf("pending retirement egress CIDRs: %v", got)
	}

	state.Data["Zones/zone-retired/"+infraflow.IdentifierZoneRetirementCompleted] = infraflow.ZoneRetirementCompletedValue
	state.Data["Zones/zone-retired/NATGatewayElasticIPAddress"] = "<deleted>"
	if got := getEgressIpCidrs(state); !reflect.DeepEqual(got, []string{"192.0.2.2/32"}) {
		t.Fatalf("completed retirement must retain the active zone's user-provided EIP: %v", got)
	}

	delete(state.Data, "Zones/zone-active/NATGatewayElasticIPAddress")
	if got := getEgressIpCidrs(state); got != nil {
		t.Fatalf("underlying egress collector changed its empty-result contract, got %#v", got)
	}
}

func equalStringsIgnoringOrder(left, right []string) bool {
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}
