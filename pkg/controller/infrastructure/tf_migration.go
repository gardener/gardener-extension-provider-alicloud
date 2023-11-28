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

package infrastructure

import (
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/terraformer"
	"k8s.io/apimachinery/pkg/runtime"

	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

func migrateTerraformStateToFlowState(rawExtension *runtime.RawExtension, zones []aliapi.Zone) (*infraflow.PersistentState, error) {
	var (
		tfRawState *terraformer.RawState
		tfState    *shared.TerraformState
		err        error
	)

	flowState := infraflow.NewPersistentState()
	// return flowState, nil

	if rawExtension == nil {
		return flowState, nil
	}

	if tfRawState, err = getTerraformerRawState(rawExtension); err != nil {
		return nil, err
	}
	if tfState, err = shared.UnmarshalTerraformStateFromTerraformer(tfRawState); err != nil {
		return nil, err
	}

	if tfState.Outputs == nil {
		return flowState, nil
	}

	vpc_id := tfState.Outputs["vpc_id"].Value
	if vpc_id != "" {
		setFlowStateData(flowState, infraflow.IdentifierVPC, &vpc_id)
	}
	sg_id := tfState.Outputs["sg_id"].Value
	if sg_id != "" {
		setFlowStateData(flowState, infraflow.IdentifierNodesSecurityGroup, &sg_id)
	}

	nat_gateway := tfState.GetManagedResourceInstanceID("alicloud_nat_gateway", "nat_gateway")
	if nat_gateway != nil && *nat_gateway != "" {
		setFlowStateData(flowState, infraflow.IdentifierNatGateway, nat_gateway)
	}
	for i, zone := range zones {
		keyPrefix := infraflow.ChildIdZones + shared.Separator + zone.Name + shared.Separator
		suffix := fmt.Sprintf("z%d", i)
		setFlowStateData(flowState, keyPrefix+infraflow.IdentifierZoneSuffix, &suffix)

		eip := tfState.GetManagedResourceInstanceID("alicloud_eip", "eip_natgw_"+suffix)
		if eip != nil && *eip != "" {
			setFlowStateData(flowState, keyPrefix+infraflow.IdentifierZoneNATGWElasticIP, eip)
		}

		setFlowStateData(flowState, keyPrefix+infraflow.IdentifierZoneVSwitch,
			tfState.GetManagedResourceInstanceID("alicloud_vswitch", "vsw_"+suffix))
	}

	flowState.SetMigratedFromTerraform()

	return flowState, nil
}

func setFlowStateData(state *infraflow.PersistentState, key string, id *string) {
	if id == nil {
		delete(state.Data, key)
	} else {
		state.Data[key] = *id
	}
}

func getTerraformerRawState(state *runtime.RawExtension) (*terraformer.RawState, error) {
	if state == nil {
		return nil, nil
	}
	tfRawState, err := terraformer.UnmarshalRawState(state)
	if err != nil {
		return nil, fmt.Errorf("could not decode terraform raw state: %+v", err)
	}
	return tfRawState, nil
}
