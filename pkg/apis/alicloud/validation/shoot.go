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
	"net"
	"regexp"

	"github.com/gardener/gardener/pkg/apis/core"
	validationutils "github.com/gardener/gardener/pkg/utils/validation"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

const (
	maxDataDiskCount = 64

	dataDiskNameFmt string = `^[a-zA-Z][a-zA-Z0-9\.\-_:]+$`

	// ReservedCIDR is a IPV4 CIDR reserved for AliCloud internal usage.
	// For example: the meta service endpoint is 100.100.100.200.
	ReservedCIDR = "100.64.0.0/10"
)

// ValidateNetworkingUpdate validates the network setting of a Shoot during update.
// The network CIDR settings should be immutable.
func ValidateNetworkingUpdate(oldNetworking, newNetworking core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateNetworkImmutable(oldNetworking.Nodes, newNetworking.Nodes, fldPath.Child("nodes"))...)
	allErrs = append(allErrs, validateNetworkImmutable(oldNetworking.Pods, newNetworking.Pods, fldPath.Child("pods"))...)
	allErrs = append(allErrs, validateNetworkImmutable(oldNetworking.Services, newNetworking.Services, fldPath.Child("services"))...)

	return allErrs
}

// ValidateNetworking validates the network settings of a Shoot during creation.
func ValidateNetworking(networking core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateNetworkCIDRNotNil(networking.Nodes, fldPath.Child("nodes"))...)

	allErrs = append(allErrs, validateNetworkCIDR(networking.Nodes, fldPath.Child("nodes"))...)
	allErrs = append(allErrs, validateNetworkCIDR(networking.Pods, fldPath.Child("pods"))...)
	allErrs = append(allErrs, validateNetworkCIDR(networking.Services, fldPath.Child("services"))...)

	return allErrs
}

// ValidateWorkers validates the workers of a Shoot.
func ValidateWorkers(workers []core.Worker, zones []apisalicloud.Zone, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	alicloudZones := sets.New[string]()
	dataDiskNameRegexp := regexp.MustCompile(dataDiskNameFmt)

	for _, alicloudZone := range zones {
		alicloudZones.Insert(alicloudZone.Name)
	}

	for i, worker := range workers {
		if worker.Volume == nil {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("volume"), "must not be nil"))
		} else {
			allErrs = append(allErrs, validateVolume(worker.Volume, fldPath.Index(i).Child("volume"))...)
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
				if validationutils.ShouldEnforceImmutability(newWorker.Zones, oldWorker.Zones) {
					allErrs = append(allErrs, apivalidation.ValidateImmutableField(newWorker.Zones, oldWorker.Zones, fldPath.Index(i).Child("zones"))...)
				}
				break
			}
		}
	}
	return allErrs
}

func validateZones(zones []string, allowedZones sets.Set[string], fldPath *field.Path) field.ErrorList {
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

func validateNetworkCIDR(cidr *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if cidr != nil && cidrvalidation.NetworksIntersect(*cidr, ReservedCIDR) {
		allErrs = append(allErrs, field.Invalid(fldPath, fldPath, fmt.Sprintf("must not overlap with %s, it is reserved by Alicloud", ReservedCIDR)))
	}

	return allErrs
}

func validateNetworkImmutable(cidrOld, cidrNew *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if cidrOld != nil {
		if _, _, err := net.ParseCIDR(*cidrOld); err == nil {
			// Validate the immutable field only when the old CIDR is validated.
			// 1. Allow the update from invalidated CIDR to validated, like "null" => "10.250.0.0/16".
			// 2. Deny the update from validated CIDR to validated, like "10.252.0.0/16" => 10.250.0.0/16.
			allErrs = append(allErrs, apivalidation.ValidateImmutableField(cidrNew, cidrOld, fldPath)...)
		}
	}

	return allErrs
}

func validateNetworkCIDRNotNil(cidr *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if cidr == nil {
		allErrs = append(allErrs, field.Required(fldPath, "a CIDR must be provided"))
	}

	return allErrs
}
