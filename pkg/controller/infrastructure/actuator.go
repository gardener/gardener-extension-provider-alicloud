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
	"strconv"
	"strings"
	"time"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	commonext "github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	"github.com/gardener/gardener/extensions/pkg/terraformer"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionschartrenderer "github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils/flow"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StatusTypeMeta is the TypeMeta of InfrastructureStatus.
var StatusTypeMeta = func() metav1.TypeMeta {
	apiVersion, kind := alicloudv1alpha1.SchemeGroupVersion.WithKind(extensioncontroller.UnsafeGuessKind(&alicloudv1alpha1.InfrastructureStatus{})).ToAPIVersionAndKind()
	return metav1.TypeMeta{
		APIVersion: apiVersion,
		Kind:       kind,
	}
}()

// NewActuator instantiates an actuator with the default dependencies.
func NewActuator(machineImageOwnerSecretRef *corev1.SecretReference, whitelistedImageIDs []string) infrastructure.Actuator {
	return NewActuatorWithDeps(
		log.Log.WithName("infrastructure-actuator"),
		alicloudclient.NewClientFactory(),
		terraformer.DefaultFactory(),
		extensionschartrenderer.DefaultFactory(),
		DefaultTerraformOps(),
		machineImageOwnerSecretRef,
		whitelistedImageIDs,
	)
}

// NewActuatorWithDeps instantiates an actuator with the given dependencies.
func NewActuatorWithDeps(
	logger logr.Logger,
	newClientFactory alicloudclient.ClientFactory,
	terraformerFactory terraformer.Factory,
	chartRendererFactory extensionschartrenderer.Factory,
	terraformChartOps TerraformChartOps,
	machineImageOwnerSecretRef *corev1.SecretReference,
	whitelistedImageIDs []string,
) infrastructure.Actuator {
	a := &actuator{
		logger:                     logger,
		ChartRendererContext:       commonext.NewChartRendererContext(chartRendererFactory),
		newClientFactory:           newClientFactory,
		terraformerFactory:         terraformerFactory,
		terraformChartOps:          terraformChartOps,
		machineImageOwnerSecretRef: machineImageOwnerSecretRef,
		whitelistedImageIDs:        whitelistedImageIDs,
	}

	return a
}

type actuator struct {
	logger logr.Logger
	commonext.ChartRendererContext

	alicloudECSClient  alicloudclient.ECS
	newClientFactory   alicloudclient.ClientFactory
	terraformerFactory terraformer.Factory
	terraformChartOps  TerraformChartOps

	machineImageOwnerSecretRef *corev1.SecretReference
	whitelistedImageIDs        []string
}

// InjectAPIReader implements inject.APIReader and instantiates actuator.alicloudECSClient.
func (a *actuator) InjectAPIReader(reader client.Reader) error {
	if a.machineImageOwnerSecretRef != nil {
		machineImageOwnerSecret := &corev1.Secret{}
		err := reader.Get(context.Background(), client.ObjectKey{
			Name:      a.machineImageOwnerSecretRef.Name,
			Namespace: a.machineImageOwnerSecretRef.Namespace,
		}, machineImageOwnerSecret)
		if err != nil {
			return err
		}
		seedCloudProviderCredentials, err := alicloud.ReadSecretCredentials(machineImageOwnerSecret)
		if err != nil {
			return err
		}
		a.alicloudECSClient, err = a.newClientFactory.NewECSClient("", seedCloudProviderCredentials.AccessKeyID, seedCloudProviderCredentials.AccessKeySecret)
		return err
	}
	return nil
}

func (a *actuator) getConfigAndCredentialsForInfra(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) (*alicloudv1alpha1.InfrastructureConfig, *alicloud.Credentials, error) {
	config := &alicloudv1alpha1.InfrastructureConfig{}
	if _, _, err := a.Decoder().Decode(infra.Spec.ProviderConfig.Raw, nil, config); err != nil {
		return nil, nil, err
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.Client(), &infra.Spec.SecretRef)
	if err != nil {
		return nil, nil, err
	}

	return config, credentials, nil
}

func (a *actuator) fetchEIPInternetChargeType(vpcClient alicloudclient.VPC, tf terraformer.Terraformer) (string, error) {
	stateVariables, err := tf.GetStateOutputVariables(TerraformerOutputKeyVPCID)
	if err != nil {
		if apierrors.IsNotFound(err) || terraformer.IsVariablesNotFoundError(err) {
			return alicloudclient.DefaultInternetChargeType, nil
		}
		return "", err
	}

	return FetchEIPInternetChargeType(vpcClient, stateVariables[TerraformerOutputKeyVPCID])
}

func (a *actuator) getInitializerValues(
	ctx context.Context,
	tf terraformer.Terraformer,
	infra *extensionsv1alpha1.Infrastructure,
	config *alicloudv1alpha1.InfrastructureConfig,
	credentials *alicloud.Credentials,
) (*InitializerValues, error) {
	vpcClient, err := a.newClientFactory.NewVPCClient(infra.Spec.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	if config.Networks.VPC.ID == nil {
		internetChargeType, err := a.fetchEIPInternetChargeType(vpcClient, tf)
		if err != nil {
			return nil, err
		}

		return a.terraformChartOps.ComputeCreateVPCInitializerValues(config, internetChargeType), nil
	}

	vpcID := *config.Networks.VPC.ID

	vpcInfo, err := GetVPCInfo(vpcClient, vpcID)
	if err != nil {
		return nil, err
	}

	return a.terraformChartOps.ComputeUseVPCInitializerValues(config, vpcInfo), nil
}

func (a *actuator) newInitializer(infra *extensionsv1alpha1.Infrastructure, config *alicloudv1alpha1.InfrastructureConfig, values *InitializerValues, stateInitializer terraformer.StateConfigMapInitializer) (terraformer.Initializer, error) {
	chartValues := a.terraformChartOps.ComputeChartValues(infra, config, values)
	release, err := a.ChartRenderer().Render(alicloud.InfraChartPath, alicloud.InfraRelease, infra.Namespace, chartValues)
	if err != nil {
		return nil, err
	}

	files, err := terraformer.ExtractTerraformFiles(release)
	if err != nil {
		return nil, err
	}

	return a.terraformerFactory.DefaultInitializer(a.Client(), files.Main, files.Variables, files.TFVars, stateInitializer), nil
}

func (a *actuator) newTerraformer(infra *extensionsv1alpha1.Infrastructure, credentials *alicloud.Credentials) (terraformer.Terraformer, error) {
	return common.NewTerraformerWithAuth(a.terraformerFactory, a.RESTConfig(), TerraformerPurpose, infra.Namespace, infra.Name, credentials)
}

func (a *actuator) extractStatus(tf terraformer.Terraformer, infraConfig *alicloudv1alpha1.InfrastructureConfig, machineImages []alicloudv1alpha1.MachineImage) (*alicloudv1alpha1.InfrastructureStatus, error) {
	outputVarKeys := []string{
		TerraformerOutputKeyVPCID,
		TerraformerOutputKeyVPCCIDR,
		TerraformerOutputKeySecurityGroupID,
		TerraformerOutputKeyKeyPairName,
	}

	for zoneIndex := range infraConfig.Networks.Zones {
		outputVarKeys = append(outputVarKeys, fmt.Sprintf("%s%d", TerraformerOutputKeyVSwitchNodesPrefix, zoneIndex))
	}

	vars, err := tf.GetStateOutputVariables(outputVarKeys...)
	if err != nil {
		return nil, err
	}

	vswitches, err := computeProviderStatusVSwitches(infraConfig, vars)
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
		KeyPairName:   vars[TerraformerOutputKeyKeyPairName],
		MachineImages: machineImages,
	}, nil
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

// findMachineImage takes a list of machine images and tries to find the first entry
// whose name and version matches with the given name and version. If no such entry is
// found then an error will be returned.
func findMachineImage(machineImages []alicloudv1alpha1.MachineImage, name, version string) (*alicloudv1alpha1.MachineImage, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == name && machineImage.Version == version {
			return &machineImage, nil
		}
	}
	return nil, fmt.Errorf("no machine image name %q in version %q found", name, version)
}

func appendMachineImage(machineImages []alicloudv1alpha1.MachineImage, machineImage alicloudv1alpha1.MachineImage) []alicloudv1alpha1.MachineImage {
	if _, err := findMachineImage(machineImages, machineImage.Name, machineImage.Version); err != nil {
		return append(machineImages, machineImage)
	}
	return machineImages
}

func (a *actuator) isWhitelistedImageID(imageID string) bool {
	if a.whitelistedImageIDs != nil {
		for _, whitelistedImageID := range a.whitelistedImageIDs {
			if imageID == whitelistedImageID {
				return true
			}
		}
	}
	return false
}

// shareCustomizedImages checks whether Shoot's Alicloud account has permissions to use the customized images. If it can't
// access them, these images will be shared with it from Seed's Alicloud account. The list of images that worker use will be
// returned.
func (a *actuator) shareCustomizedImages(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) ([]alicloudv1alpha1.MachineImage, error) {
	var (
		machineImages []alicloudv1alpha1.MachineImage
	)

	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return nil, err
	}
	a.logger.Info("Creating Alicloud ECS client for Shoot", "infrastructure", infra.Name)
	shootAlicloudECSClient, err := a.newClientFactory.NewECSClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	a.logger.Info("Creating Alicloud STS client for Shoot", "infrastructure", infra.Name)
	shootAlicloudSTSClient, err := a.newClientFactory.NewSTSClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	shootCloudProviderAccountID, err := shootAlicloudSTSClient.GetAccountIDFromCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	a.logger.Info("Sharing customized image with Shoot's Alicloud account from Seed", "infrastructure", infra.Name)
	for _, worker := range cluster.Shoot.Spec.Provider.Workers {
		imageID, err := helper.FindImageForRegionFromCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region)
		if err != nil {
			if providerStatus := infra.Status.ProviderStatus; providerStatus != nil {
				infrastructureStatus := &apisalicloud.InfrastructureStatus{}
				if _, _, err := a.Decoder().Decode(providerStatus.Raw, nil, infrastructureStatus); err != nil {
					return nil, errors.Wrapf(err, "could not decode infrastructure status of infrastructure '%s'", kutil.ObjectName(infra))
				}

				machineImage, err := helper.FindMachineImage(infrastructureStatus.MachineImages, worker.Machine.Image.Name, *worker.Machine.Image.Version)
				if err != nil {
					return nil, err
				}
				imageID = machineImage.ID
			} else {
				return nil, err
			}
		}
		machineImages = appendMachineImage(machineImages, alicloudv1alpha1.MachineImage{
			Name:    worker.Machine.Image.Name,
			Version: *worker.Machine.Image.Version,
			ID:      imageID,
		})

		// if this is a whitelisted machine image, we no longer need to check if it exists in cloud provider account, and
		// we don't need to share the image to that account either.
		if a.isWhitelistedImageID(imageID) {
			a.logger.Info("Skipping image sharing for whitelisted image", "imageID", imageID)
			continue
		}

		isPublic, err := shootAlicloudECSClient.CheckIfImagePublic(ctx, imageID)
		if err != nil {
			return nil, err
		}
		if isPublic {
			continue
		}

		exists, err := shootAlicloudECSClient.CheckIfImageExists(ctx, imageID)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		if a.alicloudECSClient == nil {
			return nil, fmt.Errorf("image sharing is not enabled or configured correctly and Alicloud ECS client is not instantiated in Seed. Please contact Gardener administrator")
		}
		if err := a.alicloudECSClient.ShareImageToAccount(ctx, infra.Spec.Region, imageID, shootCloudProviderAccountID); err != nil {
			return nil, err
		}
	}

	return machineImages, nil
}

// Reconcile implements infrastructure.Actuator.
func (a *actuator) Reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	return a.reconcile(ctx, infra, cluster, terraformer.StateConfigMapInitializerFunc(terraformer.CreateState))
}

// Restore implements infrastructure.Actuator.
func (a *actuator) Restore(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	terraformState, err := terraformer.UnmarshalRawState(infra.Status.State)
	if err != nil {
		return err
	}
	return a.reconcile(ctx, infra, cluster, terraformer.CreateOrUpdateState{State: &terraformState.Data})
}

func (a *actuator) reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster, stateInitializer terraformer.StateConfigMapInitializer) error {
	config, credentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	tf, err := a.newTerraformer(infra, credentials)
	if err != nil {
		return err
	}

	initializerValues, err := a.getInitializerValues(ctx, tf, infra, config, credentials)
	if err != nil {
		return err
	}

	initializer, err := a.newInitializer(infra, config, initializerValues, stateInitializer)
	if err != nil {
		return err
	}

	if err := tf.InitializeWith(initializer).Apply(); err != nil {
		return errors.Wrapf(err, "failed to apply the terraform config")
	}

	var machineImages []alicloudv1alpha1.MachineImage
	if cluster.Shoot != nil {
		machineImages, err = a.shareCustomizedImages(ctx, infra, cluster)
		if err != nil {
			return errors.Wrapf(err, "failed to share the machine images")
		}
	}

	status, err := a.extractStatus(tf, config, machineImages)
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

	return extensioncontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.Client(), infra, func() error {
		infra.Status.ProviderStatus = &runtime.RawExtension{Object: status}
		infra.Status.State = &runtime.RawExtension{Raw: stateByte}
		return nil
	})
}

func (a *actuator) cleanupServiceLoadBalancers(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) error {
	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}
	a.logger.Info("Creating Alicloud SLB client for Shoot", "infrastructure", infra.Name)
	shootAlicloudSLBClient, err := a.newClientFactory.NewSLBClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return err
	}

	loadBalancerIDs, err := shootAlicloudSLBClient.GetLoadBalancerIDs(ctx, infra.Spec.Region)
	if err != nil {
		return err
	}
	// SLBs created by Alicloud CCM do not have association with VPCs, so can only be iterated to check
	// if one SLB is related to this specific Shoot.
	for _, loadBalancerID := range loadBalancerIDs {
		vServerGroupName, err := shootAlicloudSLBClient.GetFirstVServerGroupName(ctx, infra.Spec.Region, loadBalancerID)
		if err != nil {
			return err
		}
		if vServerGroupName == "" {
			continue
		}

		// Get the last slice of VServerGroupName string divided by '/' which is the clusterid.
		slices := strings.Split(vServerGroupName, "/")
		clusterID := slices[len(slices)-1]
		if clusterID == infra.Namespace {
			err = shootAlicloudSLBClient.DeleteLoadBalancer(ctx, infra.Spec.Region, loadBalancerID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete implements infrastructure.Actuator.
func (a *actuator) Delete(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	tf, err := common.NewTerraformer(a.terraformerFactory, a.RESTConfig(), TerraformerPurpose, infra.Namespace, infra.Name)
	if err != nil {
		return err
	}

	// terraform pod from previous reconciliation might still be running, ensure they are gone before doing any operations
	if err := tf.EnsureCleanedUp(ctx); err != nil {
		return err
	}

	// If the Terraform state is empty then we can exit early as we didn't create anything. Though, we clean up potentially
	// created configmaps/secrets related to the Terraformer.
	stateIsEmpty, err := common.IsStateEmpty(tf)
	if err != nil {
		return err
	}
	if stateIsEmpty {
		a.logger.Info("exiting early as infrastructure state is empty or contains no resources - nothing to do")
		return tf.CleanupConfiguration(ctx)
	}

	_, credentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	configExists, err := tf.ConfigExists()
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
				return a.cleanupServiceLoadBalancers(ctx, infra)
			}).RetryUntilTimeout(10*time.Second, 5*time.Minute),
		})

		_ = g.Add(flow.Task{
			Name:         "Destroying Shoot infrastructure",
			Fn:           flow.SimpleTaskFn(tf.SetVariablesEnvironment(common.TerraformVariablesEnvironmentFromCredentials(credentials)).Destroy),
			Dependencies: flow.NewTaskIDs(destroyServiceLoadBalancers),
		})

		f = g.Compile()
	)

	if err := f.Run(flow.Opts{Context: ctx}); err != nil {
		return flow.Causes(err)
	}
	return nil
}

// Migrate implements infrastructure.Actuator.
func (a *actuator) Migrate(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	tf, err := common.NewTerraformer(a.terraformerFactory, a.RESTConfig(), TerraformerPurpose, infra.Namespace, infra.Name)
	if err != nil {
		return err
	}

	return tf.CleanupConfiguration(ctx)
}
