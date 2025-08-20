// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package mutator

import (
	"context"
	encodingjson "encoding/json"
	"fmt"
	"reflect"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	apisalicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
)

const (
	// MutatorPath is the mutator webhook path.
	MutatorPath = "/webhooks/mutate"

	overlayKey = "overlay"
	enabledKey = "enabled"
)

// NewShootMutator returns a new instance of a shoot mutator.
func NewShootMutator(mgr manager.Manager) extensionswebhook.Mutator {
	alicloudclientFactory := alicloudclient.NewClientFactory()
	return NewShootMutatorWithDeps(mgr, alicloudclientFactory)
}

// NewShootMutatorWithDeps with parameter returns a new instance of a shoot mutator.
func NewShootMutatorWithDeps(mgr manager.Manager, alicloudclientFactory alicloudclient.ClientFactory) extensionswebhook.Mutator {
	return &shootMutator{
		client:                mgr.GetClient(),
		apiReader:             mgr.GetAPIReader(),
		codec:                 runtime.NewCodec(json.NewSerializerWithOptions(json.DefaultMetaFactory, mgr.GetScheme(), mgr.GetScheme(), json.SerializerOptions{}), serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder()),
		alicloudClientFactory: alicloudclientFactory,
	}
}

type shootMutator struct {
	client                client.Client
	apiReader             client.Reader
	codec                 runtime.Codec
	alicloudClientFactory alicloudclient.ClientFactory
}

func (s *shootMutator) Mutate(ctx context.Context, newObj, oldObj client.Object) error {
	shoot, ok := newObj.(*corev1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObj)
	}

	// skip validation if it's a workerless Shoot
	if gardencorev1beta1helper.IsWorkerless(shoot) {
		return nil
	}

	if shoot.Spec.Networking != nil && shoot.Spec.Networking.Type != nil {
		err := s.mutateNetworkOverlay(shoot, oldObj)
		if err != nil {
			return err
		}
	}

	if oldObj != nil {
		oldShoot, ok := oldObj.(*corev1beta1.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", oldObj)
		}
		return s.mutateShootUpdate(ctx, shoot, oldShoot)
	} else {
		return s.mutateShootCreation(ctx, shoot)
	}
}

func (s *shootMutator) mutateShootCreation(ctx context.Context, shoot *corev1beta1.Shoot) error {
	logger.Info("Starting Shoot Creation Mutation")

	err := s.mutateControlPlaneConfigForCreate(shoot)
	if err != nil {
		return err
	}

	for _, worker := range shoot.Spec.Provider.Workers {
		if err := s.setDefaultForEncryptedDisk(ctx, shoot, &worker); err != nil {
			return err
		}
	}

	return nil
}

func (s *shootMutator) mutateNetworkOverlay(shoot *corev1beta1.Shoot, old client.Object) error {
	// Skip if shoot is in restore or migration phase
	if wasShootRescheduledToNewSeed(shoot) {
		return nil
	}

	var oldShoot *corev1beta1.Shoot
	var ok bool
	if old != nil {
		oldShoot, ok = old.(*corev1beta1.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T", old)
		}
	}

	if oldShoot != nil && isShootInMigrationOrRestorePhase(shoot) {
		return nil
	}

	// Skip if specs are matching
	if oldShoot != nil && reflect.DeepEqual(shoot.Spec, oldShoot.Spec) {
		return nil
	}

	// Skip if shoot is in deletion phase
	if shoot.DeletionTimestamp != nil || oldShoot != nil && oldShoot.DeletionTimestamp != nil {
		return nil
	}

	if shoot.Spec.Networking != nil {
		networkConfig, err := s.decodeNetworkConfig(shoot.Spec.Networking.ProviderConfig)
		if err != nil {
			return err
		}

		if oldShoot == nil && networkConfig[overlayKey] == nil {
			networkConfig[overlayKey] = map[string]interface{}{enabledKey: false}
		}

		if oldShoot != nil && networkConfig[overlayKey] == nil {
			oldNetworkConfig, err := s.decodeNetworkConfig(oldShoot.Spec.Networking.ProviderConfig)
			if err != nil {
				return err
			}

			if oldNetworkConfig[overlayKey] != nil {
				networkConfig[overlayKey] = oldNetworkConfig[overlayKey]
			}
		}

		if *shoot.Spec.Networking.Type == "calico" {
			if overlay, ok := networkConfig[overlayKey].(map[string]interface{}); ok {
				if !overlay[enabledKey].(bool) {
					networkConfig["snatToUpstreamDNS"] = map[string]interface{}{enabledKey: false}
				}
			}
		}

		modifiedJSON, err := encodingjson.Marshal(networkConfig)
		if err != nil {
			return err
		}
		shoot.Spec.Networking.ProviderConfig = &runtime.RawExtension{
			Raw: modifiedJSON,
		}
	}

	return nil
}

func (s *shootMutator) decodeNetworkConfig(network *runtime.RawExtension) (map[string]interface{}, error) {
	var networkConfig map[string]interface{}
	if network == nil || network.Raw == nil {
		return map[string]interface{}{}, nil
	}
	if err := encodingjson.Unmarshal(network.Raw, &networkConfig); err != nil {
		return nil, err
	}
	return networkConfig, nil
}

func (s *shootMutator) setDefaultForEncryptedDisk(ctx context.Context, shoot *corev1beta1.Shoot, worker *corev1beta1.Worker) error {
	imageName := worker.Machine.Image.Name
	imageVersion := worker.Machine.Image.Version
	logger.Info("Check ImageName: " + imageName + "; ImageVesion: " + *imageVersion)
	if worker.DataVolumes != nil {
		for i := range worker.DataVolumes {
			volume := &worker.DataVolumes[i]
			if volume.Encrypted == nil {
				logger.Info("Set encrypted disk by default for data disk")
				volume.Encrypted = ptr.To(true)
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
			return nil
		}
		logger.Info("Customized Image is used and we set encrypted disk by default for system disk")
		worker.Volume.Encrypted = ptr.To(true)
	}
	return nil
}

func (s *shootMutator) isCustomizedImage(ctx context.Context, shoot *corev1beta1.Shoot, imageName string, imageVersion *string) (bool, error) {
	cloudProfile, err := gardener.GetCloudProfile(ctx, s.client, shoot)
	if err != nil {
		return false, err
	}
	if cloudProfile == nil {
		return false, fmt.Errorf("cloudprofile could not be found")
	}
	region := shoot.Spec.Region
	logger.Info("Checking in cloudProfile", "CloudProfile", client.ObjectKeyFromObject(cloudProfile), "Region", region)
	imageId, err := s.getImageId(ctx, imageName, imageVersion, region, cloudProfile)
	if err != nil || imageId == "" {
		return false, err
	}
	logger.Info("Got ImageID", "ImageID", imageId)
	isOwnedByAli, err := s.isOwnedByAliCloud(ctx, shoot, imageId, region)
	return !isOwnedByAli, err
}

func (s *shootMutator) isOwnedByAliCloud(ctx context.Context, shoot *corev1beta1.Shoot, imageId string, region string) (bool, error) {
	if shoot.Spec.SecretBindingName == nil && shoot.Spec.CredentialsBindingName == nil {
		return false, fmt.Errorf("secretBindingName and credentialsBindingName cannot be both nil")
	}

	var secretKey client.ObjectKey
	if shoot.Spec.SecretBindingName != nil {
		bindingKey := client.ObjectKey{Namespace: shoot.Namespace, Name: *shoot.Spec.SecretBindingName}
		secretBinding := &corev1beta1.SecretBinding{}
		if err := kutil.LookupObject(ctx, s.client, s.apiReader, bindingKey, secretBinding); err != nil {
			return false, err
		}
		secretKey = client.ObjectKey{Namespace: secretBinding.SecretRef.Namespace, Name: secretBinding.SecretRef.Name}
	} else {
		bindingKey := client.ObjectKey{Namespace: shoot.Namespace, Name: *shoot.Spec.CredentialsBindingName}
		credentialsBinding := &securityv1alpha1.CredentialsBinding{}
		if err := kutil.LookupObject(ctx, s.client, s.apiReader, bindingKey, credentialsBinding); err != nil {
			return false, err
		}
		secretKey = client.ObjectKey{Namespace: credentialsBinding.CredentialsRef.Namespace, Name: credentialsBinding.CredentialsRef.Name}
	}

	secret := &corev1.Secret{}
	if err := s.apiReader.Get(ctx, secretKey, secret); err != nil {
		return false, err
	}
	accessKeyID, ok := secret.Data[alicloud.AccessKeyID]
	if !ok {
		return false, fmt.Errorf("missing %q field in secret %s", alicloud.AccessKeyID, secret.Name)
	}
	accessKeySecret, ok := secret.Data[alicloud.AccessKeySecret]
	if !ok {
		return false, fmt.Errorf("missing %q field in secret %s", alicloud.AccessKeySecret, secret.Name)
	}
	shootECSClient, err := s.alicloudClientFactory.NewECSClient(region, string(accessKeyID), string(accessKeySecret))
	if err != nil {
		return false, err
	}
	if exist, err := shootECSClient.CheckIfImageExists(imageId); err != nil {
		return false, err
	} else if exist {
		return shootECSClient.CheckIfImageOwnedByAliCloud(imageId)
	}
	return false, nil
}

func (s *shootMutator) getImageId(_ context.Context, imageName string, imageVersion *string, imageRegion string, cloudProfileSpec *corev1beta1.CloudProfile) (string, error) {
	cloudProfileConfig, err := s.getCloudProfileConfig(cloudProfileSpec)
	if err != nil {
		return "", err
	}
	return helper.FindImageForRegionFromCloudProfile(cloudProfileConfig, imageName, *imageVersion, imageRegion)
}

func (s *shootMutator) getCloudProfileConfig(cloudProfile *corev1beta1.CloudProfile) (*api.CloudProfileConfig, error) {
	var cloudProfileConfig = &api.CloudProfileConfig{}
	if _, _, err := s.codec.Decode(cloudProfile.Spec.ProviderConfig.Raw, nil, cloudProfileConfig); err != nil {
		return nil, fmt.Errorf("could not decode providerConfig of cloudProfile for '%s': %w", client.ObjectKeyFromObject(cloudProfile), err)
	}

	return cloudProfileConfig, nil
}

func (s *shootMutator) mutateShootUpdate(ctx context.Context, shoot, oldShoot *corev1beta1.Shoot) error {
	if !equality.Semantic.DeepEqual(shoot.Spec, oldShoot.Spec) {
		if err := s.mutateControlPlaneConfigForUpdate(shoot, oldShoot); err != nil {
			return err
		}

		if err := s.triggerInfraUpdateForNewEncryptedSystemDisk(ctx, shoot, oldShoot); err != nil {
			return err
		}
	}
	if !equality.Semantic.DeepEqual(shoot.Spec, oldShoot.Spec) {
		s.mutateForEncryptedSystemDiskChange(shoot, oldShoot)
	}
	return nil
}

func (s *shootMutator) triggerInfraUpdateForNewEncryptedSystemDisk(ctx context.Context, shoot, oldshoot *corev1beta1.Shoot) error {
	for _, worker := range shoot.Spec.Provider.Workers {
		oldWorker := getWorkerByName(oldshoot, worker.Name)
		if oldWorker == nil {
			logger.Info("Set default value of encrypted disk for newly added worker")
			if err := s.setDefaultForEncryptedDisk(ctx, shoot, &worker); err != nil {
				return err
			}
			continue
		}
		if worker.Volume != nil && worker.Volume.Encrypted == nil && oldWorker.Volume != nil && oldWorker.Volume.Encrypted != nil {
			logger.Info("Encrypted disk flag for system disk is not set, keep old value")
			worker.Volume.Encrypted = oldWorker.Volume.Encrypted
		}
		oldDataVolumes := oldWorker.DataVolumes
		for i := range worker.DataVolumes {
			dataVolume := &worker.DataVolumes[i]
			oldDataVolume := getVolumeByName(oldDataVolumes, dataVolume.Name)
			if oldDataVolume == nil {
				if dataVolume.Encrypted == nil {
					logger.Info("Set encrypted disk by default for newly added data disk")
					dataVolume.Encrypted = ptr.To(true)
				}
				continue
			}
			if dataVolume.Encrypted == nil && oldDataVolume.Encrypted != nil {
				logger.Info("Encrypted disk flag for data disk is not set, keep old value")
				dataVolume.Encrypted = oldDataVolume.Encrypted
			}
		}
	}
	return nil
}

func getWorkerByName(shoot *corev1beta1.Shoot, workerName string) *corev1beta1.Worker {
	for _, worker := range shoot.Spec.Provider.Workers {
		if worker.Name == workerName {
			return &worker
		}
	}
	return nil
}

func getVolumeByName(dataVolumes []corev1beta1.DataVolume, volumeName string) *corev1beta1.DataVolume {
	if dataVolumes == nil {
		return nil
	}
	for _, volume := range dataVolumes {
		if volume.Name == volumeName {
			return &volume
		}
	}
	return nil
}

func (s *shootMutator) mutateForEncryptedSystemDiskChange(shoot, oldShoot *corev1beta1.Shoot) {
	if requireNewEncryptedImage(shoot.Spec.Provider.Workers, oldShoot.Spec.Provider.Workers) {
		logger.Info("Need to reconcile infra as new encrypted system disk found in workers", "name", shoot.Name, "namespace", shoot.Namespace)
		if shoot.Annotations == nil {
			shoot.Annotations = make(map[string]string)
		}

		controllerutils.AddTasks(shoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)
	}
}

// Check encrypted flag in new workers' volumes. If it is changed to be true, check for old workers
// if there is already a volume is set to be encrypted and also the OS version is the same.
func requireNewEncryptedImage(newWorkers, oldWorkers []corev1beta1.Worker) bool {
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

func (s *shootMutator) decodeControlPlaneConfig(provider *corev1beta1.Provider) (*apisalicloudv1alpha1.ControlPlaneConfig, error) {
	cpConfig := &apisalicloudv1alpha1.ControlPlaneConfig{}

	if provider.ControlPlaneConfig != nil {
		if _, _, err := s.codec.Decode(provider.ControlPlaneConfig.Raw, nil, cpConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of controlplane: %w", err)
		}
	}

	return cpConfig, nil
}

func (s *shootMutator) convertToRawExtension(obj runtime.Object) (*runtime.RawExtension, error) {
	if obj == nil {
		return nil, nil
	}

	data, err := runtime.Encode(s.codec, obj)
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: data,
	}, nil
}

func (s *shootMutator) mutateControlPlaneConfigForCreate(shoot *corev1beta1.Shoot) error {
	cpConfig, err := s.decodeControlPlaneConfig(&shoot.Spec.Provider)
	if err != nil {
		return err
	}

	if cpConfig.CSI == nil {
		cpConfig.CSI = &apisalicloudv1alpha1.CSI{
			EnableADController: ptr.To(true),
		}
	} else {
		if cpConfig.CSI.EnableADController == nil {
			cpConfig.CSI.EnableADController = ptr.To(true)
		}
	}

	raw, err := s.convertToRawExtension(cpConfig)
	if err != nil {
		return err
	}

	shoot.Spec.Provider.ControlPlaneConfig = raw

	return nil
}

func (s *shootMutator) mutateControlPlaneConfigForUpdate(newShoot, oldShoot *corev1beta1.Shoot) error {
	oldCPConfig, err := s.decodeControlPlaneConfig(&oldShoot.Spec.Provider)
	if err != nil {
		return err
	}

	newCPConfig, err := s.decodeControlPlaneConfig(&newShoot.Spec.Provider)
	if err != nil {
		return err
	}

	changed := false
	// If EnableADController in new shoot is nil, keep the old value
	if oldCPConfig.CSI != nil {
		if newCPConfig.CSI == nil {
			newCPConfig.CSI = &apisalicloudv1alpha1.CSI{EnableADController: oldCPConfig.CSI.EnableADController}
			changed = true
		} else if newCPConfig.CSI.EnableADController == nil {
			newCPConfig.CSI.EnableADController = oldCPConfig.CSI.EnableADController
			changed = true
		}
	}

	if changed {
		raw, err := s.convertToRawExtension(newCPConfig)
		if err != nil {
			return err
		}
		newShoot.Spec.Provider.ControlPlaneConfig = raw
	}

	return nil
}

// wasShootRescheduledToNewSeed returns true if the shoot.Spec.SeedName has been changed, but the migration operation has not started yet.
func wasShootRescheduledToNewSeed(shoot *corev1beta1.Shoot) bool {
	return shoot.Status.LastOperation != nil &&
		shoot.Status.LastOperation.Type != corev1beta1.LastOperationTypeMigrate &&
		shoot.Spec.SeedName != nil &&
		shoot.Status.SeedName != nil &&
		*shoot.Spec.SeedName != *shoot.Status.SeedName
}

// isShootInMigrationOrRestorePhase returns true if the shoot is currently being migrated or restored.
func isShootInMigrationOrRestorePhase(shoot *corev1beta1.Shoot) bool {
	return shoot.Status.LastOperation != nil &&
		(shoot.Status.LastOperation.Type == corev1beta1.LastOperationTypeRestore &&
			shoot.Status.LastOperation.State != corev1beta1.LastOperationStateSucceeded ||
			shoot.Status.LastOperation.Type == corev1beta1.LastOperationTypeMigrate)
}
