// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"fmt"
	"strings"

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

func getEgressCidrs(terraformState *terraformer.RawState) ([]string, error) {
	tfState, err := shared.UnmarshalTerraformStateFromTerraformer(terraformState)
	if err != nil {
		if strings.Contains(err.Error(), "could not decode terraform state") {
			return nil, nil
		}
		return nil, err
	}
	resources := tfState.FindManagedResourcesByType("alicloud_eip")

	egressCidrs := []string{}
	for _, resource := range resources {
		for _, instance := range resource.Instances {
			rawIpAddress := instance.Attributes["ip_address"]
			ipAddress, ok := rawIpAddress.(string)
			if !ok {
				return nil, fmt.Errorf("error parsing '%v' as IP-address from Terraform state", rawIpAddress)
			}
			egressCidrs = append(egressCidrs, ipAddress+"/32")
		}
	}
	return egressCidrs, nil
}
