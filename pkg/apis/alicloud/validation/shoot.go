// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"math"
	"net"
	"regexp"

	"github.com/gardener/gardener/pkg/apis/core"
	corehelper "github.com/gardener/gardener/pkg/apis/core/helper"
	validationutils "github.com/gardener/gardener/pkg/utils/validation"
	cidrvalidation "github.com/gardener/gardener/pkg/utils/validation/cidr"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
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
func ValidateNetworkingUpdate(oldNetworking, newNetworking *core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if newNetworking == nil {
		allErrs = append(allErrs, field.Required(fldPath, "networking field can't be empty"))
		return allErrs
	}

	allErrs = append(allErrs, validateNetworkImmutable(oldNetworking.Nodes, newNetworking.Nodes, fldPath.Child("nodes"))...)
	allErrs = append(allErrs, validateNetworkImmutable(oldNetworking.Pods, newNetworking.Pods, fldPath.Child("pods"))...)
	allErrs = append(allErrs, validateNetworkImmutable(oldNetworking.Services, newNetworking.Services, fldPath.Child("services"))...)

	return allErrs
}

// ValidateNetworking validates the network settings of a Shoot during creation.
func ValidateNetworking(networking *core.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if networking == nil {
		allErrs = append(allErrs, field.Required(fldPath, "networking field can't be empty"))
		return allErrs
	}

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

				if corehelper.IsUpdateStrategyInPlace(newWorker.UpdateStrategy) {
					if !apiequality.Semantic.DeepEqual(newWorker.ProviderConfig, oldWorker.ProviderConfig) {
						allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("providerConfig"), newWorker.ProviderConfig, "providerConfig is immutable when update strategy is in-place"))
					}

					if !apiequality.Semantic.DeepEqual(newWorker.DataVolumes, oldWorker.DataVolumes) {
						allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("dataVolumes"), newWorker.DataVolumes, "dataVolumes is immutable when update strategy is in-place"))
					}
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
	if len(zones) > math.MaxInt32 {
		allErrs = append(allErrs, field.Invalid(fldPath, len(zones), "too many zones"))
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
