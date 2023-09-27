// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"context"
	"fmt"
	"strings"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	aliv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

// shouldUseFlow checks if flow reconciliation should be used, by any of these conditions:
// - annotation `alicloud.provider.extensions.gardener.cloud/use-flow=true` on infrastructure resource
// - annotation `alicloud.provider.extensions.gardener.cloud/use-flow=true` on shoot resource
// - label `alicloud.provider.extensions.gardener.cloud/use-flow=true` on seed resource (label instead of annotation, as only labels are transported from managedseed to seed object)
func (a *actuator) shouldUseFlow(infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) bool {
	return strings.EqualFold(infrastructure.Annotations[aliapi.AnnotationKeyUseFlow], "true") ||
		(cluster.Shoot != nil && strings.EqualFold(cluster.Shoot.Annotations[aliapi.AnnotationKeyUseFlow], "true")) ||
		(cluster.Seed != nil && strings.EqualFold(cluster.Seed.Labels[aliapi.SeedLabelKeyUseFlow], "true"))
}

func (a *actuator) reconcileWithFlow(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	log.Info("reconcileWithFlow")

	oldState, err := a.getFlowStateFromInfraStatus(infrastructure)
	if err != nil {
		return err
	}
	if oldState == nil {

		oldState, err = a.migrateFlowStateFromTerraformerState(ctx, log, infrastructure)
		if err != nil {
			return err
		}
	}

	flowContext, err := a.createFlowContext(ctx, log, infrastructure, cluster, oldState)
	if err != nil {
		return err
	}
	if err = flowContext.Reconcile(ctx); err != nil {
		_ = flowContext.PersistState(ctx, true)
		return util.DetermineError(err, helper.KnownCodes)
	}
	return flowContext.PersistState(ctx, true)
}

func (a *actuator) migrateFlowStateFromTerraformerState(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure) (*infraflow.PersistentState, error) {
	log.Info("starting terraform state migration")
	infrastructureConfig, err := a.decodeInfrastructureConfig(infrastructure)
	if err != nil {
		return nil, err
	}
	// state := infraflow.NewPersistentState()
	state, err := migrateTerraformStateToFlowState(infrastructure.Status.State, infrastructureConfig.Networks.Zones)
	if err != nil {
		return nil, fmt.Errorf("migration from terraform state failed: %w", err)
	}

	if err := a.updateStatusState(ctx, infrastructure, state); err != nil {
		return nil, fmt.Errorf("updating status state failed: %w", err)
	}
	log.Info("terraform state migrated successfully")

	return state, nil
}

func (a *actuator) updateStatusState(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, state *infraflow.PersistentState) error {
	infrastructureConfig, err := a.decodeInfrastructureConfig(infra)
	if err != nil {
		return err
	}

	infrastructureStatus, err := computeProviderStatusFromFlowState(infrastructureConfig, state)
	if err != nil {
		return err
	}

	stateBytes, err := state.ToJSON()
	if err != nil {
		return err
	}

	return updateProviderStatus(ctx, a.client, infra, infrastructureStatus, stateBytes)
}

func updateProviderStatus(ctx context.Context, c client.Client, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureStatus *aliv1alpha1.InfrastructureStatus, stateBytes []byte) error {

	patch := client.MergeFrom(infrastructure.DeepCopy())
	infrastructure.Status.ProviderStatus = &runtime.RawExtension{Object: infrastructureStatus}
	infrastructure.Status.State = &runtime.RawExtension{Raw: stateBytes}
	return c.Status().Patch(ctx, infrastructure, patch)
}

func computeProviderStatusFromFlowState(config *aliapi.InfrastructureConfig, state *infraflow.PersistentState) (*aliv1alpha1.InfrastructureStatus, error) {
	if len(state.Data) == 0 {
		return nil, nil
	}

	// return &aliv1alpha1.InfrastructureStatus{
	// 	TypeMeta: StatusTypeMeta,
	// 	VPC: aliv1alpha1.VPCStatus{
	// 		ID:        vars[TerraformerOutputKeyVPCID],
	// 		VSwitches: vswitches,
	// 		SecurityGroups: []aliv1alpha1.SecurityGroup{
	// 			{
	// 				Purpose: aliv1alpha1.PurposeNodes,
	// 				ID:      vars[TerraformerOutputKeySecurityGroupID],
	// 			},
	// 		},
	// 	},
	// 	MachineImages: machineImagesV1alpha1,
	// }, nil

	status := &aliv1alpha1.InfrastructureStatus{
		TypeMeta: StatusTypeMeta,
	}

	vpcID := ""
	if config.Networks.VPC.ID != nil {
		vpcID = *config.Networks.VPC.ID
	} else {
		vpcID = state.Data[infraflow.IdentifierVPC]
		if !shared.IsValidValue(vpcID) {
			vpcID = ""
		}
	}
	if vpcID != "" {
		var vswitches []aliv1alpha1.VSwitch
		prefix := infraflow.ChildIdZones + shared.Separator
		for k, v := range state.Data {
			if !shared.IsValidValue(v) {
				continue
			}
			if strings.HasPrefix(k, prefix) {
				parts := strings.Split(k, shared.Separator)
				if len(parts) != 3 {
					continue
				}
				if parts[2] == infraflow.IdentifierZoneVSwitch {
					vswitches = append(vswitches, aliv1alpha1.VSwitch{
						ID:      v,
						Purpose: aliv1alpha1.PurposeNodes,
						Zone:    parts[1],
					})
				}
			}
		}
		status.VPC = aliv1alpha1.VPCStatus{
			ID:        vpcID,
			VSwitches: vswitches,
		}
		if groupID := state.Data[infraflow.IdentifierNodesSecurityGroup]; shared.IsValidValue(groupID) {
			status.VPC.SecurityGroups = []aliv1alpha1.SecurityGroup{
				{
					Purpose: aliv1alpha1.PurposeNodes,
					ID:      groupID,
				},
			}
		}

	}

	return status, nil

}

func (a *actuator) decodeInfrastructureConfig(infrastructure *extensionsv1alpha1.Infrastructure) (*aliapi.InfrastructureConfig, error) {
	infrastructureConfig := &aliapi.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infrastructure.Spec.ProviderConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, fmt.Errorf("could not decode provider config: %w", err)
	}
	return infrastructureConfig, nil
}

func (a *actuator) createFlowContext(ctx context.Context, log logr.Logger,
	infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster, oldState *infraflow.PersistentState) (*infraflow.FlowContext, error) {
	if oldState.MigratedFromTerraform() && !oldState.TerraformCleanedUp() {
		err := a.cleanupTerraformerResources(ctx, log, infrastructure)
		if err != nil {
			return nil, fmt.Errorf("cleaning up terraformer resources failed: %w", err)
		}
		oldState.SetTerraformCleanedUp()
		if err := a.updateStatusState(ctx, infrastructure, oldState); err != nil {
			return nil, fmt.Errorf("updating status state failed: %w", err)
		}
	}

	infrastructureConfig, err := a.decodeInfrastructureConfig(infrastructure)
	if err != nil {
		return nil, err
	}
	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infrastructure)
	if err != nil {
		return nil, fmt.Errorf("failed to get shoot credentials: %w", err)
	}

	infraObjectKey := client.ObjectKey{
		Namespace: infrastructure.Namespace,
		Name:      infrastructure.Name,
	}
	persistor := func(ctx context.Context, flatState shared.FlatMap) error {
		state := infraflow.NewPersistentStateFromFlatMap(flatState)
		infra := &extensionsv1alpha1.Infrastructure{}
		if err := a.client.Get(ctx, infraObjectKey, infra); err != nil {
			return err
		}
		return a.updateStatusState(ctx, infra, state)
	}

	var oldFlatState shared.FlatMap
	if oldState != nil {
		if valid, err := oldState.HasValidVersion(); !valid {
			return nil, err
		}
		oldFlatState = oldState.ToFlatMap()
	}

	return infraflow.NewFlowContext(log, shootCloudProviderCredentials, infrastructure, infrastructureConfig, oldFlatState, persistor, cluster)
}

func (a *actuator) cleanupTerraformerResources(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure) error {

	tf, err := common.NewTerraformer(log, a.terraformerFactory, a.restConfig, TerraformerPurpose, infrastructure, a.disableProjectedTokenMount)
	if err != nil {
		return fmt.Errorf("could not create terraformer object: %w", err)
	}

	if err := tf.CleanupConfiguration(ctx); err != nil {
		return err
	}
	return tf.RemoveTerraformerFinalizerFromConfig(ctx)
}

func (a *actuator) getFlowStateFromInfraStatus(infrastructure *extensionsv1alpha1.Infrastructure) (*infraflow.PersistentState, error) {
	if infrastructure.Status.State != nil {
		return infraflow.NewPersistentStateFromJSON(infrastructure.Status.State.Raw)
	}
	return nil, nil
}

func (a *actuator) deleteWithFlow(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure) error {
	log.Info("deleteWithFlow")

	oldState, err := a.getFlowStateFromInfraStatus(infrastructure)
	if err != nil {
		return err
	}
	if oldState == nil {

		oldState, err = a.migrateFlowStateFromTerraformerState(ctx, log, infrastructure)
		if err != nil {
			return err
		}
	}
	flowContext, err := a.createFlowContext(ctx, log, infrastructure, nil, oldState)
	if err != nil {
		return err
	}
	if err = flowContext.Delete(ctx); err != nil {
		_ = flowContext.PersistState(ctx, true)
		return util.DetermineError(err, helper.KnownCodes)
	}
	return flowContext.PersistState(ctx, true)

}
