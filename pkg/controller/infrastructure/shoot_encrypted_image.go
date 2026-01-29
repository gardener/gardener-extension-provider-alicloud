// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"time"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
)

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
			if machineImage, err = a.ensureEncryptedImageForShootProviderAccount(ctx, log, cloudProfileConfig, worker, infra, shootAlicloudROSClient, shootAlicloudECSClient, shootCloudProviderAccountID, cluster); err != nil {
				return nil, err
			}
		} else {
			if machineImage, err = a.ensurePlainImageForShootProviderAccount(ctx, log, cloudProfileConfig, worker, infra, shootAlicloudECSClient, shootCloudProviderAccountID, cluster); err != nil {
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
	shootCloudProviderAccountID string,
	cluster *extensioncontroller.Cluster) (*apisalicloud.MachineImage, error) {
	infrastructureStatus := &apisalicloud.InfrastructureStatus{}
	if infra.Status.ProviderStatus != nil {
		if _, _, err := a.decoder.Decode(infra.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return nil, fmt.Errorf("could not decode infrastructure status of infrastructure '%s': %w", client.ObjectKeyFromObject(infra), err)
		}
	}

	if machineImage, err := helper.FindMachineImage(infrastructureStatus.MachineImages, worker.Machine.Image.Name, *worker.Machine.Image.Version, true); err == nil {
		return machineImage, nil
	}

	// Encrypted image is not found
	// Find from cloud profile first, if not found then from status
	var imageID string
	var err error
	var capabilitySet *apisalicloud.MachineImageFlavor
	if len(cluster.CloudProfile.Spec.MachineCapabilities) > 0 {
		machineTypeFromCloudProfile := gardencorev1beta1helper.FindMachineTypeByName(cluster.CloudProfile.Spec.MachineTypes, worker.Machine.Type)
		if machineTypeFromCloudProfile == nil {
			return nil, fmt.Errorf("machine type %q not found in cloud profile %q", worker.Machine.Type, cluster.CloudProfile.Name)
		}
		capabilitySet, err = helper.FindImageInCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region, machineTypeFromCloudProfile.Capabilities, cluster.CloudProfile.Spec.MachineCapabilities)
		if err == nil {
			imageID = capabilitySet.Regions[0].ID
		}
	} else {
		imageID, err = helper.FindImageForRegionFromCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region)
	}
	if err != nil {
		capabilitySet = &apisalicloud.MachineImageFlavor{}
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
		Name:         worker.Machine.Image.Name,
		Version:      *worker.Machine.Image.Version,
		ID:           encryptedImageID,
		Encrypted:    ptr.To(true),
		Capabilities: capabilitySet.Capabilities,
	}, nil
}

func (a *actuator) ensurePlainImageForShootProviderAccount(ctx context.Context, log logr.Logger, cloudProfileConfig *apisalicloud.CloudProfileConfig, worker gardencorev1beta1.Worker, infra *extensionsv1alpha1.Infrastructure, shootECSClient alicloudclient.ECS, shootCloudProviderAccountID string, cluster *extensioncontroller.Cluster) (*apisalicloud.MachineImage, error) {
	machineTypeFromCloudProfile := gardencorev1beta1helper.FindMachineTypeByName(cluster.CloudProfile.Spec.MachineTypes, worker.Machine.Type)
	if machineTypeFromCloudProfile == nil {
		return nil, fmt.Errorf("machine type %q not found in cloud profile %q", worker.Machine.Type, cluster.CloudProfile.Name)
	}
	var imageID string
	var err error
	var capabilitySet *apisalicloud.MachineImageFlavor
	if len(cluster.CloudProfile.Spec.MachineCapabilities) > 0 {
		capabilitySet, err := helper.FindImageInCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region, machineTypeFromCloudProfile.Capabilities, cluster.CloudProfile.Spec.MachineCapabilities)
		if err == nil {
			imageID = capabilitySet.Regions[0].ID
		}
	} else {
		imageID, err = helper.FindImageForRegionFromCloudProfile(cloudProfileConfig, worker.Machine.Image.Name, *worker.Machine.Image.Version, infra.Spec.Region)
	}
	if err != nil {
		capabilitySet = &apisalicloud.MachineImageFlavor{}
		providerStatus := infra.Status.ProviderStatus
		if providerStatus == nil {
			return nil, err
		}
		infrastructureStatus := &apisalicloud.InfrastructureStatus{}
		if _, _, err := a.decoder.Decode(providerStatus.Raw, nil, infrastructureStatus); err != nil {
			return nil, fmt.Errorf("could not decode infrastructure status of infrastructure '%s': %w", client.ObjectKeyFromObject(infra), err)
		}
		if machineImage, err := helper.FindMachineImage(infrastructureStatus.MachineImages, worker.Machine.Image.Name, *worker.Machine.Image.Version, false); err != nil {
			return nil, err
		} else {
			imageID = machineImage.ID
		}
	}

	if err = a.makeImageVisibleForShoot(ctx, log, shootECSClient, infra.Spec.Region, imageID, shootCloudProviderAccountID); err != nil {
		return nil, err
	}

	return &apisalicloud.MachineImage{
		Name:         worker.Machine.Image.Name,
		Version:      *worker.Machine.Image.Version,
		ID:           imageID,
		Encrypted:    ptr.To(false),
		Capabilities: capabilitySet.Capabilities,
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
