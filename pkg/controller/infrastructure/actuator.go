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
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	"github.com/gardener/gardener/extensions/pkg/terraformer"
	"github.com/gardener/gardener/extensions/pkg/util"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/flow"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
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
func NewActuator(mgr manager.Manager, machineImageOwnerSecretRef *corev1.SecretReference, toBeSharedImageIDs []string, disableProjectedTokenMount bool) (infrastructure.Actuator, error) {
	return NewActuatorWithDeps(
		mgr,
		alicloudclient.NewClientFactory(),
		terraformer.DefaultFactory(),
		DefaultTerraformOps(),
		machineImageOwnerSecretRef,
		toBeSharedImageIDs,
		disableProjectedTokenMount,
	)
}

// NewActuatorWithDeps instantiates an actuator with the given dependencies.
func NewActuatorWithDeps(
	mgr manager.Manager,
	newClientFactory alicloudclient.ClientFactory,
	terraformerFactory terraformer.Factory,
	terraformChartOps TerraformChartOps,
	machineImageOwnerSecretRef *corev1.SecretReference,
	toBeSharedImageIDs []string,
	disableProjectedTokenMount bool,
) (infrastructure.Actuator, error) {
	a := &actuator{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		restConfig: mgr.GetConfig(),
		decoder:    serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),

		newClientFactory:           newClientFactory,
		terraformerFactory:         terraformerFactory,
		terraformChartOps:          terraformChartOps,
		machineImageOwnerSecretRef: machineImageOwnerSecretRef,
		toBeSharedImageIDs:         toBeSharedImageIDs,
		disableProjectedTokenMount: disableProjectedTokenMount,
	}

	if a.machineImageOwnerSecretRef != nil {
		machineImageOwnerSecret := &corev1.Secret{}
		err := mgr.GetAPIReader().Get(context.Background(), client.ObjectKey{
			Name:      a.machineImageOwnerSecretRef.Name,
			Namespace: a.machineImageOwnerSecretRef.Namespace,
		}, machineImageOwnerSecret)
		if err != nil {
			return nil, err
		}
		seedCloudProviderCredentials, err := alicloud.ReadSecretCredentials(machineImageOwnerSecret, false)
		if err != nil {
			return nil, err
		}
		a.alicloudECSClient, err = a.newClientFactory.NewECSClient("", seedCloudProviderCredentials.AccessKeyID, seedCloudProviderCredentials.AccessKeySecret)
		return nil, err
	}

	return a, nil
}

type actuator struct {
	client     client.Client
	scheme     *runtime.Scheme
	decoder    runtime.Decoder
	restConfig *rest.Config

	alicloudECSClient  alicloudclient.ECS
	newClientFactory   alicloudclient.ClientFactory
	terraformerFactory terraformer.Factory
	terraformChartOps  TerraformChartOps

	machineImageOwnerSecretRef *corev1.SecretReference
	toBeSharedImageIDs         []string
	disableProjectedTokenMount bool
}

func (a *actuator) getConfigAndCredentialsForInfra(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) (*alicloudv1alpha1.InfrastructureConfig, *alicloud.Credentials, error) {
	config := &alicloudv1alpha1.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infra.Spec.ProviderConfig.Raw, nil, config); err != nil {
		return nil, nil, err
	}

	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, a.client, &infra.Spec.SecretRef)
	if err != nil {
		return nil, nil, err
	}

	return config, credentials, nil
}

func (a *actuator) fetchEIPInternetChargeType(ctx context.Context, vpcClient alicloudclient.VPC, tf terraformer.Terraformer) (string, error) {
	stateVariables, err := tf.GetStateOutputVariables(ctx, TerraformerOutputKeyVPCID)
	if err != nil {
		if apierrors.IsNotFound(err) || terraformer.IsVariablesNotFoundError(err) {
			return alicloudclient.DefaultInternetChargeType, nil
		}
		return "", err
	}

	return vpcClient.FetchEIPInternetChargeType(ctx, nil, stateVariables[TerraformerOutputKeyVPCID])
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
		internetChargeType, err := a.fetchEIPInternetChargeType(ctx, vpcClient, tf)
		if err != nil {
			return nil, err
		}

		return a.terraformChartOps.ComputeCreateVPCInitializerValues(config, internetChargeType), nil
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

	return a.terraformChartOps.ComputeUseVPCInitializerValues(config, vpcInfo), nil
}

func (a *actuator) newInitializer(infra *extensionsv1alpha1.Infrastructure, config *alicloudv1alpha1.InfrastructureConfig, podCIDR *string, values *InitializerValues, stateInitializer terraformer.StateConfigMapInitializer) (terraformer.Initializer, error) {
	chartValues := a.terraformChartOps.ComputeChartValues(infra, config, podCIDR, values)

	var mainTF bytes.Buffer
	if err := tplMainTF.Execute(&mainTF, chartValues); err != nil {
		return nil, fmt.Errorf("could not render Terraform template: %+v", err)
	}

	return a.terraformerFactory.DefaultInitializer(a.client, mainTF.String(), string(variablesTF), terraformTFVars, stateInitializer), nil
}

func (a *actuator) convertImageListToV1alpha1(machineImages []apisalicloud.MachineImage) ([]alicloudv1alpha1.MachineImage, error) {
	var result []alicloudv1alpha1.MachineImage
	for _, image := range machineImages {
		converted := &alicloudv1alpha1.MachineImage{}
		if err := a.scheme.Convert(&image, converted, nil); err != nil {
			return nil, err
		}

		result = append(result, *converted)
	}

	return result, nil
}

func (a *actuator) generateStatus(ctx context.Context, tf terraformer.Terraformer, infraConfig *alicloudv1alpha1.InfrastructureConfig, machineImages []apisalicloud.MachineImage) (*alicloudv1alpha1.InfrastructureStatus, error) {
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

	machineImagesV1alpha1, err := a.convertImageListToV1alpha1(machineImages)
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

func (a *actuator) toBeShared(imageID string) bool {
	if a.toBeSharedImageIDs != nil {
		for _, toBeSharedImageID := range a.toBeSharedImageIDs {
			if imageID == toBeSharedImageID {
				return true
			}
		}
	}
	return false
}

// ensureOldSSHKeyDetached ensures the compatibility when ssh key is generated in the current cluster via terraform.
func (a *actuator) ensureOldSSHKeyDetached(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure) error {
	if infra.Status.ProviderStatus == nil {
		return nil
	}

	infrastructureStatus := &alicloudv1alpha1.InfrastructureStatus{}
	if _, _, err := a.decoder.Decode(infra.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return err
	}

	// nolint
	if infrastructureStatus.KeyPairName == "" {
		return nil
	}

	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	shootAlicloudECSClient, err := a.newClientFactory.NewECSClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return err
	}

	// nolint
	log.V(2).Info("Detaching ssh key pair from ECS instances", "keypair", infrastructureStatus.KeyPairName)
	// nolint
	err = shootAlicloudECSClient.DetachECSInstancesFromSSHKeyPair(infrastructureStatus.KeyPairName)
	// nolint
	log.V(2).Info("Finished detaching ssh key pair from ECS instances", "keypair", infrastructureStatus.KeyPairName)

	return err
}

// ensureImagesForShootProviderAccount does following things
// 1. If worker needs an encrypted image, this method will ensure an corresponding encrypted image is copied.
// 2. If worker needs a plain image, this method will make the corresponding image is visible to shoot's provider account.
// The list of images that workers use will be returned.
func (a *actuator) ensureImagesForShootProviderAccount(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) ([]apisalicloud.MachineImage, error) {
	var (
		machineImages []apisalicloud.MachineImage
	)

	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return nil, err
	}

	shootAlicloudECSClient, err := a.newClientFactory.NewECSClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	shootAlicloudROSClient, err := a.newClientFactory.NewROSClient(infra.Spec.Region, shootCloudProviderCredentials.AccessKeyID, shootCloudProviderCredentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}

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

	log.Info("Preparing virtual machine images for Shoot's Alicloud account", "infrastructure", infra.Name)
	for _, worker := range cluster.Shoot.Spec.Provider.Workers {
		var machineImage *apisalicloud.MachineImage
		useEncrytedDisk, err := common.UseEncryptedSystemDisk(worker.Volume)
		if err != nil {
			return nil, err
		}
		if useEncrytedDisk {
			if machineImage, err = a.ensureEncryptedImageForShootProviderAccount(ctx, log, cloudProfileConfig, worker, infra, shootAlicloudROSClient, shootAlicloudECSClient, shootCloudProviderAccountID); err != nil {
				return nil, err
			}
		} else {
			if machineImage, err = a.ensurePlainImageForShootProviderAccount(ctx, log, cloudProfileConfig, worker, infra, shootAlicloudECSClient, shootCloudProviderAccountID); err != nil {
				return nil, err
			}
		}
		machineImages = helper.AppendMachineImage(machineImages, *machineImage)
	}
	log.Info("Finish preparing virtual machine images for Shoot's Alicloud account", "infrastructure", infra.Name)

	return machineImages, nil
}

func (a *actuator) ensureEncryptedImageForShootProviderAccount(
	ctx context.Context,
	log logr.Logger,
	cloudProfileConfig *apisalicloud.CloudProfileConfig,
	worker gardencorev1beta1.Worker,
	infra *extensionsv1alpha1.Infrastructure,
	shootROSClient alicloudclient.ROS,
	shootECSClient alicloudclient.ECS,
	shootCloudProviderAccountID string) (*apisalicloud.MachineImage, error) {
	infrastructureStatus := &apisalicloud.InfrastructureStatus{}
	if infra.Status.ProviderStatus != nil {
		if _, _, err := a.decoder.Decode(infra.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return nil, fmt.Errorf("could not decode infrastructure status of infrastructure '%s': %w", kutil.ObjectName(infra), err)
		}
	}

	if machineImage, err := helper.FindMachineImage(infrastructureStatus.MachineImages, worker.Machine.Image.Name, *worker.Machine.Image.Version, true); err == nil {
		return machineImage, nil
	}

	// Encrypted image is not found
	// Find from cloud profile first, if not found then from status
	imageID, err := helper.FindImageForRegionFromCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region)
	if err != nil {
		if machineImage, err := helper.FindMachineImage(infrastructureStatus.MachineImages, worker.Machine.Image.Name, *worker.Machine.Image.Version, false); err != nil {
			return nil, err
		} else {
			imageID = machineImage.ID
		}
	}

	// If it is a custom image, it need to be shared with shoot account
	if err = a.makeImageVisibleForShoot(ctx, log, shootECSClient, infra.Spec.Region, imageID, shootCloudProviderAccountID); err != nil {
		return nil, err
	}

	if exist, err := shootECSClient.CheckIfImageExists(imageID); err != nil {
		return nil, err
	} else if exist {
		// Check if image is provided by AliCloud (OwnerAlias is System).
		if ownedByAliCloud, err := shootECSClient.CheckIfImageOwnedByAliCloud(imageID); err != nil {
			return nil, err
		} else if ownedByAliCloud {
			return nil, fmt.Errorf("image (%s-%s/%s) is owned by AliCloud. An encrypted image can't be created from this image for the shoot", worker.Machine.Image.Name, *worker.Machine.Image.Version, imageID)
		}
	}
	// else {} it is private shared

	// It may block 10 minutes
	log.Info("Preparing encrypted image for shoot account", "name", worker.Machine.Image.Name, "version", *worker.Machine.Image.Version)
	encryptor := common.NewImageEncryptor(shootROSClient, infra.Spec.Region, worker.Machine.Image.Name, *worker.Machine.Image.Version, imageID)
	encryptedImageID, err := encryptor.TryToGetEncryptedImageID(ctx, 15*time.Minute, 10*time.Second)
	if err != nil {
		return nil, err
	}

	return &apisalicloud.MachineImage{
		Name:      worker.Machine.Image.Name,
		Version:   *worker.Machine.Image.Version,
		ID:        encryptedImageID,
		Encrypted: pointer.Bool(true),
	}, nil
}

func (a *actuator) ensurePlainImageForShootProviderAccount(ctx context.Context, log logr.Logger, cloudProfileConfig *apisalicloud.CloudProfileConfig, worker gardencorev1beta1.Worker, infra *extensionsv1alpha1.Infrastructure, shootECSClient alicloudclient.ECS, shootCloudProviderAccountID string) (*apisalicloud.MachineImage, error) {
	imageID, err := helper.FindImageForRegionFromCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region)
	if err != nil {
		if providerStatus := infra.Status.ProviderStatus; providerStatus != nil {
			infrastructureStatus := &apisalicloud.InfrastructureStatus{}
			if _, _, err := a.decoder.Decode(providerStatus.Raw, nil, infrastructureStatus); err != nil {
				return nil, fmt.Errorf("could not decode infrastructure status of infrastructure '%s': %w", kutil.ObjectName(infra), err)
			}
			if machineImage, err := helper.FindMachineImage(infrastructureStatus.MachineImages, worker.Machine.Image.Name, *worker.Machine.Image.Version, false); err != nil {
				return nil, err
			} else {
				imageID = machineImage.ID
			}
		} else {
			return nil, err
		}
	}

	if err = a.makeImageVisibleForShoot(ctx, log, shootECSClient, infra.Spec.Region, imageID, shootCloudProviderAccountID); err != nil {
		return nil, err
	}

	return &apisalicloud.MachineImage{
		Name:    worker.Machine.Image.Name,
		Version: *worker.Machine.Image.Version,
		ID:      imageID,
	}, nil
}

func (a *actuator) makeImageVisibleForShoot(ctx context.Context, log logr.Logger, shootECSClient alicloudclient.ECS, region, imageID, shootAccountID string) error {
	// if this is a whitelisted machine image, we no longer need to check if it exists in cloud provider account, and
	// we don't need to share the image to that account either.
	log.Info("Sharing customized image with Shoot's Alicloud account from Seed", "imageID", imageID)
	if !a.toBeShared(imageID) {
		log.Info("Skip image sharing as it is not in the ToBeSharedImageIDs", "imageID", imageID)
		return nil
	}

	exists, err := shootECSClient.CheckIfImageExists(imageID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if a.alicloudECSClient == nil {
		return fmt.Errorf("image sharing is not enabled or configured correctly and Alicloud ECS client is not instantiated in Seed. Please contact Gardener administrator")
	}

	return a.alicloudECSClient.ShareImageToAccount(ctx, region, imageID, shootAccountID)
}

// Reconcile implements infrastructure.Actuator.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	flowState, err := a.getFlowStateFromInfraStatus(infra)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}
	if flowState == nil {
		if a.shouldUseFlow(infra, cluster) {
			flowState, err = a.migrateFlowStateFromTerraformerState(ctx, log, infra)
			if err != nil {
				return util.DetermineError(err, helper.KnownCodes)
			}
		}
	}
	if flowState != nil {
		err = a.reconcileWithFlow(ctx, log, infra, cluster, flowState)
		return util.DetermineError(err, helper.KnownCodes)
	}
	return a.reconcile(ctx, log, infra, cluster, terraformer.StateConfigMapInitializerFunc(terraformer.CreateState))
}

// Restore implements infrastructure.Actuator.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster) error {
	flowState, err := a.getFlowStateFromInfraStatus(infra)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}
	if flowState == nil {
		if a.shouldUseFlow(infra, cluster) {
			flowState, err = a.migrateFlowStateFromTerraformerState(ctx, log, infra)
			if err != nil {
				return util.DetermineError(err, helper.KnownCodes)
			}
		}
	}
	if flowState != nil {
		err = a.reconcileWithFlow(ctx, log, infra, cluster, flowState)
		return util.DetermineError(err, helper.KnownCodes)
	}
	terraformState, err := terraformer.UnmarshalRawState(infra.Status.State)
	if err != nil {
		return err
	}
	return a.reconcile(ctx, log, infra, cluster, terraformer.CreateOrUpdateState{State: &terraformState.Data})
}

func (a *actuator) reconcile(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, cluster *extensioncontroller.Cluster, stateInitializer terraformer.StateConfigMapInitializer) error {
	config, credentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

	if err := a.ensureServiceLinkedRole(ctx, infra, credentials); err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	if err = a.ensureOldSSHKeyDetached(ctx, log, infra); err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	tf, err := common.NewTerraformerWithAuth(log, a.terraformerFactory, a.restConfig, TerraformerPurpose, infra, a.disableProjectedTokenMount)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	initializerValues, err := a.getInitializerValues(ctx, tf, infra, config, credentials)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	initializer, err := a.newInitializer(infra, config, cluster.Shoot.Spec.Networking.Pods, initializerValues, stateInitializer)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	if err := tf.InitializeWith(ctx, initializer).Apply(ctx); err != nil {
		return util.DetermineError(fmt.Errorf("failed to apply the terraform config: %w", err), helper.KnownCodes)
	}

	var machineImages []apisalicloud.MachineImage
	if cluster.Shoot != nil {
		machineImages, err = a.ensureImagesForShootProviderAccount(ctx, log, infra, cluster)
		if err != nil {
			return fmt.Errorf("failed to ensure machine images for shoot: %w", err)
		}
	}

	status, err := a.generateStatus(ctx, tf, config, machineImages)
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

	patch := client.MergeFrom(infra.DeepCopy())
	infra.Status.ProviderStatus = &runtime.RawExtension{Object: status}
	infra.Status.State = &runtime.RawExtension{Raw: stateByte}
	return a.client.Status().Patch(ctx, infra, patch)
}

func (a *actuator) cleanupServiceLoadBalancers(ctx context.Context, infra *extensionsv1alpha1.Infrastructure) error {
	_, shootCloudProviderCredentials, err := a.getConfigAndCredentialsForInfra(ctx, infra)
	if err != nil {
		return err
	}

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
			err = shootAlicloudSLBClient.SetLoadBalancerDeleteProtection(ctx, infra.Spec.Region, loadBalancerID, false)
			if err != nil {
				return err
			}
			err = shootAlicloudSLBClient.DeleteLoadBalancer(ctx, infra.Spec.Region, loadBalancerID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete implements infrastructure.Actuator.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, _ *extensioncontroller.Cluster) error {
	flowState, err := a.getFlowStateFromInfraStatus(infra)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}
	if flowState != nil {
		err = a.deleteWithFlow(ctx, log, infra, flowState)
		return util.DetermineError(err, helper.KnownCodes)
	}

	return a.delete(ctx, log, infra)
}

func (a *actuator) delete(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure) error {
	tf, err := common.NewTerraformer(log, a.terraformerFactory, a.restConfig, TerraformerPurpose, infra, a.disableProjectedTokenMount)
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
		log.Info("exiting early as infrastructure state is empty or contains no resources - nothing to do")
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
				return a.cleanupServiceLoadBalancers(ctx, infra)
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

// ForceDelete implements infrastructure.Actuator.
func (a *actuator) ForceDelete(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.Infrastructure, _ *extensioncontroller.Cluster) error {
	return nil
}

// Migrate implements infrastructure.Actuator.
func (a *actuator) Migrate(ctx context.Context, log logr.Logger, infra *extensionsv1alpha1.Infrastructure, _ *extensioncontroller.Cluster) error {
	flowState, err := a.getFlowStateFromInfraStatus(infra)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}
	if flowState != nil {
		return nil // nothing to do if already using new flow without Terraformer
	}
	tf, err := common.NewTerraformer(log, a.terraformerFactory, a.restConfig, TerraformerPurpose, infra, a.disableProjectedTokenMount)
	if err != nil {
		return util.DetermineError(err, helper.KnownCodes)
	}

	if err := tf.CleanupConfiguration(ctx); err != nil {
		return err
	}
	return tf.RemoveTerraformerFinalizerFromConfig(ctx)
}

// ensureServiceLinkedRole is to check if service linked role exists, if not create one.
func (a *actuator) ensureServiceLinkedRole(_ context.Context, infra *extensionsv1alpha1.Infrastructure, credentials *alicloud.Credentials) error {
	client, err := a.newClientFactory.NewRAMClient(infra.Spec.Region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return err
	}

	serviceLinkedRole, err := client.GetServiceLinkedRole(alicloud.ServiceLinkedRoleForNATGateway)
	if err != nil {
		return err
	}

	if serviceLinkedRole == nil {
		if err := client.CreateServiceLinkedRole(infra.Spec.Region, alicloud.ServiceForNATGateway); err != nil {
			return err
		}
	}

	return nil
}
