// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package helper_test

import (
	"testing"

	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
)

func TestIsManagedNATGatewayConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		vpc  api.VPC
		byo  bool
		want bool
	}{
		{name: "created VPC", want: true},
		{name: "created VPC ignores false flag", vpc: api.VPC{GardenerManagedNATGateway: ptr.To(false)}, want: true},
		{name: "user VPC default", vpc: api.VPC{ID: ptr.To("vpc-user")}},
		{name: "user VPC explicit false", vpc: api.VPC{ID: ptr.To("vpc-user"), GardenerManagedNATGateway: ptr.To(false)}},
		{name: "user VPC managed NAT", vpc: api.VPC{ID: ptr.To("vpc-user"), GardenerManagedNATGateway: ptr.To(true)}, want: true},
		{name: "BYO", vpc: api.VPC{ID: ptr.To("vpc-user")}, byo: true},
		{name: "BYO does not manage NAT even with invalid true flag", vpc: api.VPC{ID: ptr.To("vpc-user"), GardenerManagedNATGateway: ptr.To(true)}, byo: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &api.InfrastructureConfig{Networks: api.Networks{VPC: tc.vpc, Zones: []api.Zone{{Name: "zone1", Retired: true}, {Name: "zone2"}}}}
			if tc.byo {
				for i := range config.Networks.Zones {
					config.Networks.Zones[i].WorkersVSwitchID = ptr.To("vsw-user")
				}
			}
			if got := helper.IsManagedNATGatewayConfig(config); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
