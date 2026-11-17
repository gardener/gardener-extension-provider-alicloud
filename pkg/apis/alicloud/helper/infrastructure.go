// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package helper

import api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"

// IsManagedNATGatewayConfig reports whether Gardener manages the NAT gateway.
// BYO VSwitches do not use a Gardener-managed NAT gateway. A Gardener-created VPC
// always does; a user-provided VPC requires gardenerManagedNATGateway=true.
func IsManagedNATGatewayConfig(config *api.InfrastructureConfig) bool {
	for _, zone := range config.Networks.Zones {
		if zone.WorkersVSwitchID != nil {
			return false
		}
	}
	return config.Networks.VPC.ID == nil ||
		(config.Networks.VPC.GardenerManagedNATGateway != nil && *config.Networks.VPC.GardenerManagedNATGateway)
}
