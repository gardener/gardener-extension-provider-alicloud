// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package mutator

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ShootMutatorName is the shoots mutator webhook name.
	ShootMutatorName = "shoots.mutator"
	// MutatorPath is the mutator webhook path.
	MutatorPath = "/webhooks/mutate"
)

// NewShootMutator returns a new instance of a shoot validator.
func NewShootMutator(virtualGardenclient client.Client, apiReader client.Reader, decoder runtime.Decoder) extensionswebhook.Mutator {

	alicloudclientFactory := alicloudclient.NewClientFactory()
	return &shootMutator{virtualGardenclient: virtualGardenclient, apiReader: apiReader, decoder: decoder, alicloudClientFactory: alicloudclientFactory}
}

type shootMutator struct {
	virtualGardenclient   client.Client
	apiReader             client.Reader
	decoder               runtime.Decoder
	alicloudClientFactory alicloudclient.ClientFactory
}

func (s *shootMutator) Mutate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*corev1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if old != nil {
		oldShoot, ok := old.(*corev1beta1.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", old)
		}
		return s.mutateShootUpdate(oldShoot, shoot)
	} else {
		return s.mutateShootCreation(ctx, shoot)
	}

}
func (s *shootMutator) mutateShootCreation(ctx context.Context, shoot *corev1beta1.Shoot) error {
	logger.Info("Starting Shoot Creation Mutation")
	for _, worker := range shoot.Spec.Provider.Workers {
		imageName := worker.Machine.Image.Name
		imageVersion := worker.Machine.Image.Version
		logger.Info("Check ImageName: " + imageName + "; ImageVesion: " + *imageVersion)
		if worker.DataVolumes != nil {
			for i := range worker.DataVolumes {
				volume := &worker.DataVolumes[i]
				if volume.Encrypted == nil {
					logger.Info("set encrypted disk by default for data disk")
					encrypted := true
					volume.Encrypted = &encrypted
				}
			}
		}
		if worker.Volume != nil && worker.Volume.Encrypted == nil {
			//don't set encrypted disk by default if image is system image
			isCustomizeImage, err := s.isCustomizedImage(ctx, shoot, imageName, imageVersion)
			if err != nil {
				return err
			}
			if !isCustomizeImage {
				continue
			}
			logger.Info("Customized Image is used and we set encrypted disk by default for system disk")
			encrypted := true
			worker.Volume.Encrypted = &encrypted
		}

	}
	return nil
}
func (s *shootMutator) isCustomizedImage(ctx context.Context, shoot *corev1beta1.Shoot, imageName string, imageVersion *string) (bool, error) {
	cloudProfile := shoot.Spec.CloudProfileName
	region := shoot.Spec.Region
	logger.Info("Checking in cloudProfie", "CloudProfile", cloudProfile, "Region", region)
	imageId, err := s.getImageId(ctx, imageName, imageVersion, region, cloudProfile)
	if err != nil || imageId == "" {
		return false, fmt.Errorf("can't find imageID")
	}
	logger.Info("Got ImageID", "ImageID", imageId)
	s.isOwnedbyAliCloud(ctx, shoot, imageId, region)
	return true, nil
}
func (s *shootMutator) isOwnedbyAliCloud(ctx context.Context, shoot *corev1beta1.Shoot, imageId string, region string) (bool, error) {

	var (
		secretBinding    = &corev1beta1.SecretBinding{}
		secretBindingKey = kutil.Key(shoot.Namespace, shoot.Spec.SecretBindingName)
	)
	if err := kutil.LookupObject(ctx, s.virtualGardenclient, s.apiReader, secretBindingKey, secretBinding); err != nil {
		return false, err
	}

	var (
		secret    = &corev1.Secret{}
		secretRef = secretBinding.SecretRef.Name
		secretKey = kutil.Key(secretBinding.SecretRef.Namespace, secretRef)
	)
	// Explicitly use the client.Reader to prevent controller-runtime to start Informer for Secrets
	// under the hood. The latter increases the memory usage of the component.
	if err := s.apiReader.Get(ctx, secretKey, secret); err != nil {
		return false, err
	}
	accessKeyID, ok := secret.Data[alicloud.AccessKeyID]
	if !ok {
		return false, fmt.Errorf("missing %q field in secret %s", alicloud.AccessKeyID, secretRef)
	}
	accessKeySecret, ok := secret.Data[alicloud.AccessKeySecret]
	if !ok {
		return false, fmt.Errorf("missing %q field in secret %s", alicloud.AccessKeySecret, secretRef)
	}
	shootECSClient, err := s.alicloudClientFactory.NewECSClient(region, string(accessKeyID), string(accessKeySecret))
	if err != nil {
		return false, err
	}
	if exist, err := shootECSClient.CheckIfImageExists(ctx, imageId); err != nil {
		return false, err
	} else if exist {
		if ownedByAliCloud, err := shootECSClient.CheckIfImageOwnedByAliCloud(imageId); err != nil {
			return false, err
		} else if ownedByAliCloud {
			return true, nil
		}
	}
	return false, nil
}
func (s *shootMutator) getImageId(ctx context.Context, imageName string, imageVersion *string, imageRegion string, cloudProfileName string) (string, error) {
	var (
		cloudProfile    = &corev1beta1.CloudProfile{}
		cloudProfileKey = kutil.Key(cloudProfileName)
	)
	imageId := ""
	if err := kutil.LookupObject(ctx, s.virtualGardenclient, s.apiReader, cloudProfileKey, cloudProfile); err != nil {
		return imageId, err
	}
	//logger.Info("Got CloudProfile", "profile Detail", cloudProfile)
	cloudProfileConfig, err := s.getCloudProfileConfig(cloudProfile)
	if err != nil {
		return imageId, err
	}
	//	logger.Info("Got CloudProfileConfig", "Config", cloudProfileConfig)
	//	logger.Info("Images in Config", "Images", cloudProfileConfig.MachineImages)
	for _, machineImage := range cloudProfileConfig.MachineImages {
		name := machineImage.Name
		if imageName == name {
			versions := machineImage.Versions
			for _, version := range versions {
				if version.Version == *imageVersion {
					regions := version.Regions
					for _, region := range regions {
						if region.Name == imageRegion {
							imageId = region.ID
						}
					}
				}
			}
		}
	}
	return imageId, nil
}
func (s *shootMutator) getCloudProfileConfig(cloudProfile *corev1beta1.CloudProfile) (*api.CloudProfileConfig, error) {
	var cloudProfileConfig *api.CloudProfileConfig = &api.CloudProfileConfig{}
	//	logger.Info("ProvideConfig", "Raw", cloudProfile.Spec.ProviderConfig.Raw)
	if _, _, err := s.decoder.Decode(cloudProfile.Spec.ProviderConfig.Raw, nil, cloudProfileConfig); err != nil {
		return nil, fmt.Errorf("could not decode providerConfig of cloudProfile for '%s': %w", kutil.ObjectName(cloudProfile), err)
	}

	return cloudProfileConfig, nil
}
func (s *shootMutator) mutateShootUpdate(oldShoot, shoot *corev1beta1.Shoot) error {
	if !equality.Semantic.DeepEqual(oldShoot.Spec, shoot.Spec) {
		s.mutateForEncryptedSystemDiskChange(oldShoot, shoot)
	}

	return nil
}

func (s *shootMutator) mutateForEncryptedSystemDiskChange(oldShoot, shoot *corev1beta1.Shoot) {
	if requireNewEncryptedImage(oldShoot.Spec.Provider.Workers, shoot.Spec.Provider.Workers) {
		logger.Info("Need to reconcile infra as new encrypted system disk found in workers", "name", shoot.Name, "namespace", shoot.Namespace)
		if shoot.Annotations == nil {
			shoot.Annotations = make(map[string]string)
		}

		controllerutils.AddTasks(shoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)
	}
}

// Check encrypted flag in new workers' volumes. If it is changed to be true, check for old workers
// if there is already a volume is set to be encrypted and also the OS version is the same.
func requireNewEncryptedImage(oldWorkers, newWorkers []corev1beta1.Worker) bool {
	var imagesEncrypted []*corev1beta1.ShootMachineImage
	for _, w := range oldWorkers {
		if w.Volume != nil && w.Volume.Encrypted != nil && *w.Volume.Encrypted {
			if w.Machine.Image != nil {
				imagesEncrypted = append(imagesEncrypted, w.Machine.Image)
			}
		}
	}

	for _, w := range newWorkers {
		if w.Volume != nil && w.Volume.Encrypted != nil && *w.Volume.Encrypted {
			if w.Machine.Image != nil {
				found := false
				for _, image := range imagesEncrypted {
					if w.Machine.Image.Name == image.Name && reflect.DeepEqual(w.Machine.Image.Version, image.Version) {
						found = true
						break
					}
				}

				if !found {
					return true
				}
			}
		}
	}

	return false
}
