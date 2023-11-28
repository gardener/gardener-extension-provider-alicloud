// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aliclient

import (
	"context"
	"encoding/json"
	"reflect"
)

// Updater is used for reconcile based with flow
type Updater interface {
	UpdateVpc(ctx context.Context, desired, current *VPC) (modified bool, err error)
	UpdateVSwitch(ctx context.Context, desired, current *VSwitch) (modified bool, err error)
	UpdateNatgateway(ctx context.Context, desired, current *NatGateway) (modified bool, err error)
	UpdateEIP(ctx context.Context, desired, current *EIP) (modified bool, err error)
	UpdateSNATEntry(ctx context.Context, desired, current *SNATEntry) (modified bool, err error)
	UpdateSecurityGroup(ctx context.Context, desired, current *SecurityGroup) (modified bool, err error)
}

type updater struct {
	actor Actor
}

var _ Updater = &updater{}

// NewUpdater creates a new updater instance.
func NewUpdater(actor Actor) Updater {
	return &updater{
		actor: actor,
	}
}

func (u *updater) UpdateSecurityGroup(ctx context.Context, desired, current *SecurityGroup) (modified bool, err error) {

	modified, err = u.updateTags(ctx, current.SecurityGroupId, desired.Tags, current.Tags, "securitygroup")
	return
}

func (u *updater) UpdateSNATEntry(_ context.Context, _, _ *SNATEntry) (modified bool, err error) {
	return
}

func (u *updater) UpdateEIP(ctx context.Context, desired, current *EIP) (modified bool, err error) {
	if desired.Bandwidth != current.Bandwidth {
		err = u.actor.ModifyEIP(ctx, current.EipId, desired)
		if err != nil {
			return
		}
		modified = true
	}
	tagModified, err := u.updateTags(ctx, current.EipId, desired.Tags, current.Tags, "EIP")
	if err != nil {
		return
	}
	modified = modified || tagModified

	return
}

func (u *updater) UpdateNatgateway(ctx context.Context, desired, current *NatGateway) (modified bool, err error) {

	modified, err = u.updateTags(ctx, current.NatGatewayId, desired.Tags, current.Tags, "NATGATEWAY")
	return
}

func (u *updater) UpdateVSwitch(ctx context.Context, desired, current *VSwitch) (modified bool, err error) {
	modified, err = u.updateTags(ctx, current.VSwitchId, desired.Tags, current.Tags, "VSWITCH")
	return

}

func (u *updater) UpdateVpc(ctx context.Context, desired, current *VPC) (modified bool, err error) {
	modified, err = u.updateTags(ctx, current.VpcId, desired.Tags, current.Tags, "VPC")
	return
}

func (u *updater) equalJSON(a, b string) (bool, error) {
	ma := map[string]any{}
	mb := map[string]any{}
	if err := json.Unmarshal([]byte(a), &ma); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(b), &mb); err != nil {
		return false, err
	}
	return reflect.DeepEqual(ma, mb), nil
}

func (u *updater) updateTags(ctx context.Context, id string, desired, current Tags, resourceType string) (bool, error) {
	modified := false
	toBeDeleted := Tags{}
	toBeCreated := Tags{}
	toBeIgnored := Tags{}
	for k, v := range current {
		if dv, ok := desired[k]; ok {
			if dv != v {
				toBeDeleted[k] = v
				toBeCreated[k] = dv
			}
		} else if u.ignoreTag(k) {
			toBeIgnored[k] = v
		} else {
			toBeDeleted[k] = v
		}
	}
	for k, v := range desired {
		if _, ok := current[k]; !ok && !u.ignoreTag(k) {
			toBeCreated[k] = v
		}
	}

	if len(toBeDeleted) > 0 {
		if err := u.actor.DeleteTags(ctx, []string{id}, toBeDeleted, resourceType); err != nil {
			return false, err
		}
		modified = true
	}
	if len(toBeCreated) > 0 {
		if err := u.actor.CreateTags(ctx, []string{id}, toBeCreated, resourceType); err != nil {
			return false, err
		}
		modified = true
	}

	return modified, nil
}

func (u *updater) ignoreTag(_ string) bool {

	return false
}
