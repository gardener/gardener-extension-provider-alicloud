// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
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

	machineImage, err := helper.FindMachineImage(infraStatus.MachineImages, name, version, encrypted)
	if err != nil {
		opt := "unencrypted"
		if encrypted {
			opt = "encrypted"
		}
		return nil, worker.ErrorMachineImageNotFound(name, version, opt)
	}

	return machineImage, nil
}
