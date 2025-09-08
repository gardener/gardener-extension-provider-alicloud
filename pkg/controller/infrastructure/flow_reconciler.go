// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	aliv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/shared"
)

// FlowReconciler can manage infrastructure resources using Flow.
type FlowReconciler struct {
	client                     client.Client
	restConfig                 *rest.Config
	log                        logr.Logger
	disableProjectedTokenMount bool
	actuator                   *actuator
}

// NewFlowReconciler creates a new flow reconciler.
func NewFlowReconciler(client client.Client, restConfig *rest.Config, log logr.Logger, projToken bool, actuator *actuator) Reconciler {
	return &FlowReconciler{
		client:                     client,
		restConfig:                 restConfig,
		log:                        log,
		disableProjectedTokenMount: projToken,
		actuator:                   actuator,
	}
}

// Delete implements Reconciler.
func (f *FlowReconciler) Delete(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, _ *extensioncontroller.Cluster) error {
	f.log.Info("cleanupServiceLoadBalancers")
	err := f.actuator.cleanupServiceLoadBalancers(ctx, infra)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	f.log.Info("getFlowStateFromInfraStatus")
	flowState, err := f.getFlowStateFromInfraStatus(infra)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}
	err = f.deleteWithFlow(ctx, infra, flowState)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}
	return util.DetermineError(f.actuator.cleanupTerraformerResources(ctx, f.log, infra), helper.KnownCodes)
}

// Reconcile implements Reconciler.
func (f *FlowReconciler) Reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	var (
		flowState *infraflow.PersistentState
	)
	fsOk, err := hasFlowState(infra.Status.State)
	if err != nil {
		return err
	}
	if fsOk {
		flowState, err = f.getFlowStateFromInfraStatus(infra)
		if err != nil {
			return util.DetermineError(err, helper.KnownCodes)
		}
	} else {
		flowState, err = f.migrateFlowStateFromTerraformerState(ctx, infra)
		if err != nil {
			return util.DetermineError(err, helper.KnownCodes)
		}
	}

	return util.DetermineError(f.reconcileWithFlow(ctx, infra, cluster, flowState), helper.KnownCodes)
}

// Restore implements Reconciler.
func (f *FlowReconciler) Restore(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	return f.Reconcile(ctx, infra, cluster)
}

func (f *FlowReconciler) reconcileWithFlow(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster, oldState *infraflow.PersistentState) error {
	f.log.Info("reconcileWithFlow")

	var (
		machineImages []aliapi.MachineImage
		err           error
	)

	_, credentials, err := f.actuator.getConfigAndCredentialsForInfra(ctx, infrastructure)
	if err != nil {
		return err
	}

	if err := f.actuator.ensureServiceLinkedRole(ctx, infrastructure, credentials); err != nil {
		return err
	}

	if err = f.actuator.ensureOldSSHKeyDetached(ctx, f.log, infrastructure); err != nil {
		return err
	}

	if cluster.Shoot != nil {
		machineImages, err = f.actuator.ensureImagesForShootProviderAccount(ctx, f.log, infrastructure, cluster)
		if err != nil {
			return fmt.Errorf("failed to ensure machine images for shoot: %w", err)
		}
	}

	flowContext, err := f.createFlowContext(ctx, infrastructure, cluster, oldState, false)
	if err != nil {
		return err
	}
	if err = flowContext.Reconcile(ctx); err != nil {
		_ = f.updateStatusProvider(ctx, infrastructure, machineImages, flowContext.ExportState())
		return err
	}
	return f.updateStatusProvider(ctx, infrastructure, machineImages, flowContext.ExportState())
}

func (f *FlowReconciler) migrateFlowStateFromTerraformerState(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure) (*infraflow.PersistentState, error) {
	f.log.Info("starting terraform state migration")
	infrastructureConfig, err := f.decodeInfrastructureConfig(infrastructure)
	if err != nil {
		return nil, err
	}

	state, err := migrateTerraformStateToFlowState(infrastructure.Status.State, infrastructureConfig.Networks.Zones)
	if err != nil {
		return nil, fmt.Errorf("migration from terraform state failed: %w", err)
	}

	if err := f.updateStatusState(ctx, infrastructure, state); err != nil {
		return nil, fmt.Errorf("updating status state failed: %w", err)
	}
	f.log.Info("terraform state migrated successfully")

	return state, nil
}

func (f *FlowReconciler) updateStatusState(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, state *infraflow.PersistentState) error {
	stateBytes, err := state.ToJSON()
	if err != nil {
		return err
	}

	patch := client.MergeFrom(infra.DeepCopy())
	infra.Status.State = &runtime.RawExtension{Raw: stateBytes}
	return f.client.Status().Patch(ctx, infra, patch)
}

func (f *FlowReconciler) updateStatusProvider(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, machineImages []aliapi.MachineImage, flatState shared.FlatMap) error {
	infrastructureConfig, err := f.decodeInfrastructureConfig(infra)
	if err != nil {
		return err
	}

	state := infraflow.NewPersistentStateFromFlatMap(flatState)
	infrastructureStatus, err := computeProviderStatusFromFlowState(infrastructureConfig, state)

	if err != nil {
		return err
	}

	machineImagesV1alpha1, err := f.actuator.convertImageListToV1alpha1(machineImages)
	if err != nil {
		return err
	}

	infrastructureStatus.MachineImages = machineImagesV1alpha1

	patch := client.MergeFrom(infra.DeepCopy())
	infra.Status.ProviderStatus = &runtime.RawExtension{Object: infrastructureStatus}
	egressCidrs := getEgressIpCidrs(state)
	if egressCidrs != nil {
		infra.Status.EgressCIDRs = egressCidrs
	}
	return f.client.Status().Patch(ctx, infra, patch)
}

func getEgressIpCidrs(state *infraflow.PersistentState) []string {
	if len(state.Data) == 0 {
		return nil
	}

	cidrs := []string{}
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
			if parts[2] == infraflow.ZoneNATGWElasticIPAddress {
				cidrs = append(cidrs, v+"/32")
			}
		}
	}
	if len(cidrs) == 0 {
		return nil
	}

	return cidrs
}

func computeProviderStatusFromFlowState(config *aliapi.InfrastructureConfig, state *infraflow.PersistentState) (*aliv1alpha1.InfrastructureStatus, error) {
	if len(state.Data) == 0 {
		return nil, nil
	}

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

func (f *FlowReconciler) decodeInfrastructureConfig(infrastructure *extensionsv1alpha1.Infrastructure) (*aliapi.InfrastructureConfig, error) {
	infrastructureConfig := &aliapi.InfrastructureConfig{}
	if _, _, err := f.actuator.decoder.Decode(infrastructure.Spec.ProviderConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, fmt.Errorf("could not decode provider config: %w", err)
	}
	return infrastructureConfig, nil
}

func (f *FlowReconciler) createFlowContext(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster, oldState *infraflow.PersistentState, fromDelete bool) (*infraflow.FlowContext, error) {
	infrastructureConfig, err := f.decodeInfrastructureConfig(infrastructure)
	if err != nil {
		return nil, err
	}
	_, shootCloudProviderCredentials, err := f.actuator.getConfigAndCredentialsForInfra(ctx, infrastructure)
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
		if err := f.client.Get(ctx, infraObjectKey, infra); err != nil {
			return err
		}
		return f.updateStatusState(ctx, infra, state)
	}

	var oldFlatState shared.FlatMap
	if oldState != nil {
		if valid, err := oldState.HasValidVersion(); !valid {
			return nil, err
		}
		oldFlatState = oldState.ToFlatMap()
	}

	return infraflow.NewFlowContext(f.log, shootCloudProviderCredentials, infrastructure, infrastructureConfig, oldFlatState, persistor, cluster, fromDelete)
}

func (f *FlowReconciler) getFlowStateFromInfraStatus(infrastructure *extensionsv1alpha1.Infrastructure) (*infraflow.PersistentState, error) {
	if infrastructure.Status.State != nil {
		return infraflow.NewPersistentStateFromJSON(infrastructure.Status.State.Raw)
	}
	return nil, nil
}

func (f *FlowReconciler) deleteWithFlow(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, oldState *infraflow.PersistentState) error {
	f.log.Info("deleteWithFlow")

	flowContext, err := f.createFlowContext(ctx, infrastructure, nil, oldState, true)
	if err != nil {
		return err
	}
	if err = flowContext.Delete(ctx); err != nil {
		_ = flowContext.PersistState(ctx, true)
		return err
	}
	return flowContext.PersistState(ctx, true)
}
