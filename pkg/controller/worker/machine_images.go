// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"k8s.io/utils/ptr"

	api "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
)

// UpdateMachineImagesStatus implements genericactuator.WorkerDelegate.
func (w *workerDelegate) UpdateMachineImagesStatus(ctx context.Context) error {
	if w.machineImages == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return fmt.Errorf("unable to generate the machine config: %w", err)
		}
	}

	// Decode the current worker provider status.
	workerStatus, err := w.decodeWorkerProviderStatus()
	if err != nil {
		return fmt.Errorf("unable to decode the worker provider status: %w", err)
	}

	workerStatus.MachineImages = w.machineImages
	if err := w.updateWorkerProviderStatus(ctx, workerStatus); err != nil {
		return fmt.Errorf("unable to update worker provider status: %w", err)
	}

	return nil
}

func (w *workerDelegate) findMachineImage(workerPool extensionsv1alpha1.WorkerPool, infraStatus *api.InfrastructureStatus, region string) (*api.MachineImage, error) {
	name := workerPool.MachineImage.Name
	version := workerPool.MachineImage.Version
	encrypted, err := common.UseEncryptedSystemDisk(workerPool.Volume)
	if err != nil {
		return nil, err
	}

	if !encrypted {
		if len(w.cluster.CloudProfile.Spec.MachineCapabilities) > 0 {
			machineTypeFromCloudProfile := gardencorev1beta1helper.FindMachineTypeByName(w.cluster.CloudProfile.Spec.MachineTypes, workerPool.MachineType)
			if machineTypeFromCloudProfile == nil {
				return nil, fmt.Errorf("machine type %q not found in cloud profile %q", workerPool.MachineType, w.cluster.CloudProfile.Name)
			}

			imageFlavor, err := helper.FindImageInCloudProfileFlavor(w.cloudProfileConfig, name, version, region, machineTypeFromCloudProfile.Capabilities, w.cluster.CloudProfile.Spec.MachineCapabilities)
			if err == nil {
				return &api.MachineImage{
					Name:         name,
					Version:      version,
					ID:           imageFlavor.Regions[0].ID,
					Encrypted:    ptr.To(encrypted),
					Capabilities: imageFlavor.Capabilities,
				}, nil
			}
		} else {
			machineImageID, err := helper.FindImageForRegionFromCloudProfile(w.cloudProfileConfig, name, version, region)
			if err == nil {
				return &api.MachineImage{
					Name:      name,
					Version:   version,
					ID:        machineImageID,
					Encrypted: ptr.To(encrypted),
				}, nil
			}
		}
	}
	machineImage, err := helper.FindMachineImage(infraStatus.MachineImages, name, version, encrypted)
	if err != nil {
		opt := "unencrypted"
		if encrypted {
			opt = "encrypted"
		}
		return nil, worker.ErrorMachineImageNotFound(name, version, opt)
	}
	machineImage = helper.UniformSingleMachineImage(machineImage, w.cluster.CloudProfile.Spec.MachineCapabilities)
	// if len(w.cluster.CloudProfile.Spec.MachineCapabilities) > 0 && machineImage.Capabilities == nil {
	// 	machineImage.Capabilities = gardencorev1beta1.Capabilities{
	// 		v1beta1constants.ArchitectureName: []string{v1beta1constants.ArchitectureAMD64},
	// 	}
	// }

	return machineImage, nil
}
