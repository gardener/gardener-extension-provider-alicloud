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

package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
)

// MachineClassKind yields the name of the machine class kind used by Alicloud provider.
func (w *workerDelegate) MachineClassKind() string {
	return "MachineClass"
}

// MachineClass yields a newly initialized MachineClass object.
func (w *workerDelegate) MachineClass() client.Object {
	return &machinev1alpha1.MachineClass{}
}

// MachineClassList yields a newly initialized MachineClassList object.
func (w *workerDelegate) MachineClassList() client.ObjectList {
	return &machinev1alpha1.MachineClassList{}
}

// DeployMachineClasses generates and creates the Alicloud specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	if w.machineClasses == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return err
		}
	}

	// Delete any older version of AlicloudMachineClass CRs.
	if err := w.Client().DeleteAllOf(ctx, &machinev1alpha1.AlicloudMachineClass{}, client.InNamespace(w.worker.Namespace)); err != nil {
		return err
	}

	return w.seedChartApplier.Apply(ctx, filepath.Join(alicloud.InternalChartsPath, "machineclass"), w.worker.Namespace, "machineclass", kubernetes.Values(map[string]interface{}{"machineClasses": w.machineClasses}))
}

// GenerateMachineDeployments generates the configuration for the desired machine deployments.
func (w *workerDelegate) GenerateMachineDeployments(ctx context.Context) (worker.MachineDeployments, error) {
	if w.machineDeployments == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return nil, err
		}
	}
	return w.machineDeployments, nil
}

func (w *workerDelegate) generateMachineConfig(ctx context.Context) error {
	var (
		machineDeployments = worker.MachineDeployments{}
		machineClasses     []map[string]interface{}
		machineImages      []apisalicloud.MachineImage
	)

	infrastructureStatus := &apisalicloud.InfrastructureStatus{}
	if _, _, err := w.Decoder().Decode(w.worker.Spec.InfrastructureProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return err
	}

	nodesSecurityGroup, err := helper.FindSecurityGroupByPurpose(infrastructureStatus.VPC.SecurityGroups, apisalicloud.PurposeNodes)
	if err != nil {
		return err
	}

	for _, pool := range w.worker.Spec.Pools {

		zoneLen := int32(len(pool.Zones))

		workerPoolHash, err := worker.WorkerPoolHash(pool, w.cluster, computeAdditionalHashData(pool)...)
		if err != nil {
			return err
		}

		machineImage, err := w.findMachineImage(pool, infrastructureStatus, w.worker.Spec.Region)
		if err != nil {
			return err
		}

		machineImages = helper.AppendMachineImage(machineImages, *machineImage)

		disks, err := computeDisks(w.worker.Namespace, pool)
		if err != nil {
			return err
		}

		for zoneIndex, zone := range pool.Zones {
			zoneIdx := int32(zoneIndex)
			nodesVSwitch, err := helper.FindVSwitchForPurposeAndZone(infrastructureStatus.VPC.VSwitches, apisalicloud.PurposeNodes, zone)
			if err != nil {
				return err
			}

			machineClassSpec := utils.MergeMaps(map[string]interface{}{
				"imageID":                 machineImage.ID,
				"instanceType":            pool.MachineType,
				"region":                  w.worker.Spec.Region,
				"zoneID":                  zone,
				"securityGroupID":         nodesSecurityGroup.ID,
				"vSwitchID":               nodesVSwitch.ID,
				"instanceChargeType":      "PostPaid",
				"internetChargeType":      "PayByTraffic",
				"internetMaxBandwidthIn":  5,
				"internetMaxBandwidthOut": 5,
				"spotStrategy":            "NoSpot",
				"tags": map[string]string{
					fmt.Sprintf("kubernetes.io/cluster/%s", w.worker.Namespace):     "1",
					fmt.Sprintf("kubernetes.io/role/worker/%s", w.worker.Namespace): "1",
				},
				"secret": map[string]interface{}{
					"userData": string(pool.UserData),
				},
				"credentialsSecretRef": map[string]interface{}{
					"name":      w.worker.Spec.SecretRef.Name,
					"namespace": w.worker.Spec.SecretRef.Namespace,
				},
			}, disks)

			var (
				deploymentName = fmt.Sprintf("%s-%s-%s", w.worker.Namespace, pool.Name, zone)
				className      = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
			)

			machineDeployments = append(machineDeployments, worker.MachineDeployment{
				Name:                 deploymentName,
				ClassName:            className,
				SecretName:           className,
				Minimum:              worker.DistributeOverZones(zoneIdx, pool.Minimum, zoneLen),
				Maximum:              worker.DistributeOverZones(zoneIdx, pool.Maximum, zoneLen),
				MaxSurge:             worker.DistributePositiveIntOrPercent(zoneIdx, pool.MaxSurge, zoneLen, pool.Maximum),
				MaxUnavailable:       worker.DistributePositiveIntOrPercent(zoneIdx, pool.MaxUnavailable, zoneLen, pool.Minimum),
				Labels:               pool.Labels,
				Annotations:          pool.Annotations,
				Taints:               pool.Taints,
				MachineConfiguration: genericworkeractuator.ReadMachineConfiguration(pool),
			})

			machineClassSpec["name"] = className
			machineClassSpec["labels"] = map[string]string{
				v1beta1constants.GardenerPurpose: genericworkeractuator.GardenPurposeMachineClass,
			}

			machineClasses = append(machineClasses, machineClassSpec)
		}
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses
	w.machineImages = machineImages

	return nil
}

func computeDisks(namespace string, pool extensionsv1alpha1.WorkerPool) (map[string]interface{}, error) {
	// handle root disk
	volumeSize, err := worker.DiskSize(pool.Volume.Size)
	if err != nil {
		return nil, err
	}
	systemDisk := map[string]interface{}{
		"size": volumeSize,
	}
	if pool.Volume.Type != nil {
		systemDisk["category"] = *pool.Volume.Type
	}

	disks := map[string]interface{}{
		"systemDisk": systemDisk,
	}

	var dataDisks []map[string]interface{}
	if dataVolumes := pool.DataVolumes; len(dataVolumes) > 0 {
		for _, vol := range pool.DataVolumes {
			volumeSize, err := worker.DiskSize(vol.Size)
			if err != nil {
				return nil, err
			}
			dataDisk := map[string]interface{}{
				"name":               vol.Name,
				"size":               volumeSize,
				"deleteWithInstance": true,
				"description":        fmt.Sprintf("%s-datavol-%s", namespace, vol.Name),
			}
			if vol.Type != nil {
				dataDisk["category"] = *vol.Type
			}
			if vol.Encrypted != nil {
				dataDisk["encrypted"] = *vol.Encrypted
			}
			dataDisks = append(dataDisks, dataDisk)
		}

		disks["dataDisks"] = dataDisks
	}

	return disks, nil
}

func computeAdditionalHashData(pool extensionsv1alpha1.WorkerPool) []string {
	var additionalData []string

	// Volume.Encrypted needs to be included when calculating the hash
	if pool.Volume.Encrypted != nil {
		additionalData = append(additionalData, strconv.FormatBool(*pool.Volume.Encrypted))
	}

	for _, dv := range pool.DataVolumes {
		additionalData = append(additionalData, dv.Size)

		if dv.Type != nil {
			additionalData = append(additionalData, *dv.Type)
		}

		if dv.Encrypted != nil {
			additionalData = append(additionalData, strconv.FormatBool(*dv.Encrypted))
		}
	}

	return additionalData
}
