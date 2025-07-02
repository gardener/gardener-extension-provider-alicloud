package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/terraformer"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/flow"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
)

// TerraformReconciler can manage infrastructure resources using Terraformer.
type TerraformReconciler struct {
	client                     client.Client
	restConfig                 *rest.Config
	log                        logr.Logger
	disableProjectedTokenMount bool
	actuator                   *actuator
	terraformChartOps          TerraformChartOps
	terraformerFactory         terraformer.Factory
}

// NewTerraformReconciler returns a new instance of TerraformReconciler.
func NewTerraformReconciler(client client.Client, restConfig *rest.Config, log logr.Logger, disableProjectedTokenMount bool, actuator *actuator, terraformChartOps TerraformChartOps, terraformerFactory terraformer.Factory) Reconciler {
	return &TerraformReconciler{
		client:                     client,
		restConfig:                 restConfig,
		log:                        log,
		disableProjectedTokenMount: disableProjectedTokenMount,
		actuator:                   actuator,
		terraformChartOps:          terraformChartOps,
		terraformerFactory:         terraformerFactory,
	}
}

// Delete implements Reconciler.
func (t *TerraformReconciler) Delete(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, _ *controller.Cluster) error {
	return util.DetermineError(t.delete(ctx, infra), helper.KnownCodes)
}

func (t *TerraformReconciler) delete(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) error {
	tf, err := common.NewTerraformer(t.log, t.terraformerFactory, t.restConfig, TerraformerPurpose, infra, t.disableProjectedTokenMount)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	// terraform pod from previous reconciliation might still be running, ensure they are gone before doing any operations
	if err := tf.EnsureCleanedUp(ctx); err != nil {
		return err
	}

	// If the Terraform state is empty then we can exit early as we didn't create anything. Though, we clean up potentially
	// created configmaps/secrets related to the Terraformer.
	stateIsEmpty, err := common.IsStateEmpty(ctx, tf)
	if err != nil {
		return err
	}
	if stateIsEmpty {
		t.log.Info("exiting early as infrastructure state is empty or contains no resources - nothing to do")
		return tf.CleanupConfiguration(ctx)
	}

	configExists, err := tf.ConfigExists(ctx)
	if err != nil {
		return err
	}
	if !configExists {
		return nil
	}

	var (
		g = flow.NewGraph("Alicloud infrastructure destruction")

		destroyServiceLoadBalancers = g.Add(flow.Task{
			Name: "Destroying service load balancers",
			Fn: flow.TaskFn(func(ctx context.Context) error {
				return t.actuator.cleanupServiceLoadBalancers(ctx, infra)
			}).RetryUntilTimeout(10*time.Second, 5*time.Minute),
		})

		_ = g.Add(flow.Task{
			Name:         "Destroying Shoot infrastructure",
			Fn:           tf.SetEnvVars(common.TerraformerEnvVars(infra.Spec.SecretRef)...).Destroy,
			Dependencies: flow.NewTaskIDs(destroyServiceLoadBalancers),
		})

		f = g.Compile()
	)

	if err := f.Run(ctx, flow.Opts{}); err != nil {
		return util.DetermineError(flow.Errors(err), helper.KnownCodes)
	}
	return nil
}

// Reconcile implements Reconciler.
func (t *TerraformReconciler) Reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	err := t.reconcile(ctx, infra, cluster, terraformer.StateConfigMapInitializerFunc(terraformer.CreateState))
	return util.DetermineError(err, helper.KnownCodes)
}

// Restore implements Reconciler.
func (t *TerraformReconciler) Restore(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster) error {
	terraformState, err := terraformer.UnmarshalRawState(infra.Status.State)
	if err != nil {
		return err
	}

	return t.reconcile(ctx, infra, cluster, terraformer.CreateOrUpdateState{State: &terraformState.Data})
}

func (t *TerraformReconciler) reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *controller.Cluster, stateInitializer terraformer.StateConfigMapInitializer) error {
	config, credentials, err := t.actuator.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	if err := t.actuator.ensureServiceLinkedRole(ctx, infra, credentials); err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	if err = t.actuator.ensureOldSSHKeyDetached(ctx, t.log, infra); err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	tf, err := common.NewTerraformerWithAuth(t.log, t.terraformerFactory, t.restConfig, TerraformerPurpose, infra, t.disableProjectedTokenMount)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	initializerValues, err := t.getInitializerValues(ctx, tf, infra, config, credentials)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	initializer, err := t.newInitializer(infra, config, cluster.Shoot.Spec.Networking.Pods, initializerValues, stateInitializer)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	if err := tf.InitializeWith(ctx, initializer).Apply(ctx); err != nil {
		return util.DetermineError(fmt.Errorf("failed to apply the terraform config: %w", err), helper.KnownCodes)
	}

	var machineImages []apisalicloud.MachineImage
	if cluster.Shoot != nil {
		machineImages, err = t.actuator.ensureImagesForShootProviderAccount(ctx, t.log, infra, cluster)
		if err != nil {
			return fmt.Errorf("failed to ensure machine images for shoot: %w", err)
		}
	}

	status, err := t.generateStatus(ctx, tf, config, machineImages)
	if err != nil {
		return err
	}

	state, err := tf.GetRawState(ctx)
	if err != nil {
		return err
	}
	stateByte, err := state.Marshal()
	if err != nil {
		return err
	}
	egressCidrs, err := getEgressCidrs(state)
	if err != nil {
		return err
	}

	patch := client.MergeFrom(infra.DeepCopy())
	infra.Status.ProviderStatus = &runtime.RawExtension{Object: status}
	infra.Status.State = &runtime.RawExtension{Raw: stateByte}
	if egressCidrs != nil {
		infra.Status.EgressCIDRs = egressCidrs
	}
	return t.client.Status().Patch(ctx, infra, patch)
}

func (t *TerraformReconciler) newInitializer(infra *extensionsv1alpha1.Infrastructure, config *alicloudv1alpha1.InfrastructureConfig, podCIDR *string, values *InitializerValues, stateInitializer terraformer.StateConfigMapInitializer) (terraformer.Initializer, error) {
	chartValues := t.terraformChartOps.ComputeChartValues(infra, config, podCIDR, values)

	var mainTF bytes.Buffer
	if err := tplMainTF.Execute(&mainTF, chartValues); err != nil {
		return nil, fmt.Errorf("could not render Terraform template: %+v", err)
	}

	return t.terraformerFactory.DefaultInitializer(t.client, mainTF.String(), string(variablesTF), terraformTFVars, stateInitializer), nil
}

func fetchEIPInternetChargeType(ctx context.Context, vpcClient alicloudclient.VPC, tf terraformer.Terraformer) (string, error) {
	stateVariables, err := tf.GetStateOutputVariables(ctx, TerraformerOutputKeyVPCID)
	if err != nil {
		if apierrors.IsNotFound(err) || terraformer.IsVariablesNotFoundError(err) {
			return alicloudclient.DefaultInternetChargeType, nil
		}
		return "", err
	}

	return vpcClient.FetchEIPInternetChargeType(ctx, nil, stateVariables[TerraformerOutputKeyVPCID])
}

func (t *TerraformReconciler) getInitializerValues(
	ctx context.Context,
	tf terraformer.Terraformer,
	infra *extensionsv1alpha1.Infrastructure,
	config *alicloudv1alpha1.InfrastructureConfig,
	credentials *alicloud.Credentials,
) (*InitializerValues, error) {
	vpcClient, err := t.actuator.newClientFactory.NewVPCClient(infra.Spec.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	if config.Networks.VPC.ID == nil {
		internetChargeType, err := fetchEIPInternetChargeType(ctx, vpcClient, tf)
		if err != nil {
			return nil, err
		}

		return t.terraformChartOps.ComputeCreateVPCInitializerValues(config, internetChargeType), nil
	}

	vpcID := *config.Networks.VPC.ID

	vpcInfo := &alicloudclient.VPCInfo{}
	if config.Networks.VPC.GardenerManagedNATGateway != nil && *config.Networks.VPC.GardenerManagedNATGateway {
		vpc, err := vpcClient.GetVPCWithID(ctx, vpcID)
		if err != nil {
			return nil, err
		}

		vpcInfo.CIDR = vpc[0].CidrBlock
		vpcInfo.InternetChargeType = alicloudclient.DefaultInternetChargeType
	} else {
		vpcInfo, err = vpcClient.GetVPCInfo(ctx, vpcID)
		if err != nil {
			return nil, err
		}
	}

	return t.terraformChartOps.ComputeUseVPCInitializerValues(config, vpcInfo), nil
}

func computeProviderStatusVSwitches(infrastructure *alicloudv1alpha1.InfrastructureConfig, values map[string]string) ([]alicloudv1alpha1.VSwitch, error) {
	var vswitchesToReturn []alicloudv1alpha1.VSwitch

	for key, value := range values {
		var (
			prefix  string
			purpose alicloudv1alpha1.Purpose
		)

		if strings.HasPrefix(key, TerraformerOutputKeyVSwitchNodesPrefix) {
			prefix = TerraformerOutputKeyVSwitchNodesPrefix
			purpose = alicloudv1alpha1.PurposeNodes
		}

		if len(prefix) == 0 {
			continue
		}

		zoneID, err := strconv.Atoi(strings.TrimPrefix(key, prefix))
		if err != nil {
			return nil, err
		}
		vswitchesToReturn = append(vswitchesToReturn, alicloudv1alpha1.VSwitch{
			ID:      value,
			Purpose: purpose,
			Zone:    infrastructure.Networks.Zones[zoneID].Name,
		})
	}

	return vswitchesToReturn, nil
}

func (t *TerraformReconciler) generateStatus(ctx context.Context, tf terraformer.Terraformer, infraConfig *alicloudv1alpha1.InfrastructureConfig, machineImages []apisalicloud.MachineImage) (*alicloudv1alpha1.InfrastructureStatus, error) {
	outputVarKeys := []string{
		TerraformerOutputKeyVPCID,
		TerraformerOutputKeyVPCCIDR,
		TerraformerOutputKeySecurityGroupID,
	}

	for zoneIndex := range infraConfig.Networks.Zones {
		outputVarKeys = append(outputVarKeys, fmt.Sprintf("%s%d", TerraformerOutputKeyVSwitchNodesPrefix, zoneIndex))
	}

	vars, err := tf.GetStateOutputVariables(ctx, outputVarKeys...)
	if err != nil {
		return nil, err
	}

	vswitches, err := computeProviderStatusVSwitches(infraConfig, vars)
	if err != nil {
		return nil, err
	}

	machineImagesV1alpha1, err := t.actuator.convertImageListToV1alpha1(machineImages)
	if err != nil {
		return nil, err
	}

	return &alicloudv1alpha1.InfrastructureStatus{
		TypeMeta: StatusTypeMeta,
		VPC: alicloudv1alpha1.VPCStatus{
			ID:        vars[TerraformerOutputKeyVPCID],
			VSwitches: vswitches,
			SecurityGroups: []alicloudv1alpha1.SecurityGroup{
				{
					Purpose: alicloudv1alpha1.PurposeNodes,
					ID:      vars[TerraformerOutputKeySecurityGroupID],
				},
			},
		},
		MachineImages: machineImagesV1alpha1,
	}, nil
}
