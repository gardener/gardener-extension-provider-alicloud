// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package validation

import (
	"fmt"
	"regexp"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/apis/core/validation"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// AliCloudVPCCidr is a IPV4 CIDR used within AliCloud.
// For example: the meta service endpoint is 100.100.100.200.
// This CIDR can be accessed by any machine which is running with AliCloud VPC.
const AliCloudVPCCidr = "100.64.0.0/10"

// ValidateNetworking validates the network settings of a Shoot.
func ValidateNetworking(networking core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if networking.Nodes == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("nodes"), "a nodes CIDR must be provided for AliCloud shoots"))
	} else {
		if cidrvalidation.NetworksIntersect(*networking.Nodes, AliCloudVPCCidr) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("nodes"), *networking.Nodes, "must not overlap with 100.64.0.0/10 because 10.64.0.0/10 is reserved by AliCloud"))
		}
	}
	if cidrvalidation.NetworksIntersect(*networking.Pods, AliCloudVPCCidr) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("pods"), *networking.Pods, "must not overlap with 100.64.0.0/10 because 10.64.0.0/10 is reserved by AliCloud"))
	}
	if cidrvalidation.NetworksIntersect(*networking.Services, AliCloudVPCCidr) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("services"), *networking.Services, "must not overlap with 100.64.0.0/10 because 10.64.0.0/10 is reserved by AliCloud"))
	}

	return allErrs
}

const (
	maxDataDiskCount        = 64
	dataDiskNameFmt  string = `[a-zA-Z][a-zA-Z0-9\.\-_:]+`
)

var dataDiskNameRegexp = regexp.MustCompile("^" + dataDiskNameFmt + "$")

// ValidateWorkers validates the workers of a Shoot.
func ValidateWorkers(workers []core.Worker, zones []apisalicloud.Zone, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	alicloudZones := sets.NewString()
	for _, alicloudZone := range zones {
		alicloudZones.Insert(alicloudZone.Name)
	}

	for i, worker := range workers {
		if worker.Volume == nil {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("volume"), "must not be nil"))
		} else {
			allErrs = append(allErrs, validateVolume(worker.Volume, fldPath.Index(i).Child("volume"))...)
			if worker.Volume.Encrypted != nil {
				allErrs = append(allErrs, field.NotSupported(fldPath.Index(i).Child("volume").Child("encrypted"), *worker.Volume.Encrypted, nil))
			}
		}

		if length := len(worker.DataVolumes); length > maxDataDiskCount {
			allErrs = append(allErrs, field.TooMany(fldPath.Index(i).Child("dataVolumes"), length, maxDataDiskCount))
		}
		for j, volume := range worker.DataVolumes {
			dataVolPath := fldPath.Index(i).Child("dataVolumes").Index(j)
			if !dataDiskNameRegexp.MatchString(volume.Name) {
				allErrs = append(allErrs, field.Invalid(dataVolPath.Child("name"), volume.Name, utilvalidation.RegexError(fmt.Sprintf("disk name given: %s does not match the expected pattern", volume.Name), dataDiskNameFmt)))
			} else if len(volume.Name) > 64 {
				allErrs = append(allErrs, field.TooLong(dataVolPath.Child("name"), volume.Name, 64))
			}
			allErrs = append(allErrs, validateDataVolume(&volume, dataVolPath)...)
		}

		if len(worker.Zones) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("zones"), "at least one zone must be configured"))
			continue
		}

		if worker.Maximum != 0 && worker.Minimum == 0 {
			allErrs = append(allErrs, field.Forbidden(fldPath.Index(i).Child("minimum"), "minimum value must be >= 1 if maximum value > 0 (cluster-autoscaler cannot handle min=0)"))
		}

		allErrs = append(allErrs, validateZones(worker.Zones, alicloudZones, fldPath.Index(i).Child("zones"))...)
	}

	return allErrs
}

// ValidateWorkersUpdate validates updates on `workers`
func ValidateWorkersUpdate(oldWorkers, newWorkers []core.Worker, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for i, newWorker := range newWorkers {
		for _, oldWorker := range oldWorkers {
			if newWorker.Name == oldWorker.Name {
				if validation.ShouldEnforceImmutability(newWorker.Zones, oldWorker.Zones) {
					allErrs = append(allErrs, apivalidation.ValidateImmutableField(newWorker.Zones, oldWorker.Zones, fldPath.Index(i).Child("zones"))...)
				}
				break
			}
		}
	}
	return allErrs
}

func validateZones(zones []string, allowedZones sets.String, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for i, workerZone := range zones {
		if !allowedZones.Has(workerZone) {
			allErrs = append(allErrs, field.Invalid(fldPath.Index(i), workerZone, fmt.Sprintf("supported values %v", allowedZones.UnsortedList())))
		}
	}
	return allErrs
}

func validateVolume(vol *core.Volume, fldPath *field.Path) field.ErrorList {
	return validateVolumeFunc(vol.Type, vol.VolumeSize, fldPath)
}

func validateDataVolume(vol *core.DataVolume, fldPath *field.Path) field.ErrorList {
	return validateVolumeFunc(vol.Type, vol.VolumeSize, fldPath)
}

func validateVolumeFunc(volumeType *string, size string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if volumeType == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "must not be empty"))
	}
	if size == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("size"), "must not be empty"))
	}
	return allErrs
}
