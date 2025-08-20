// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
)

// ValidateCloudProfileConfig validates a CloudProfileConfig object.
func ValidateCloudProfileConfig(cloudProfile *apisalicloud.CloudProfileConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	machineImagesPath := fldPath.Child("machineImages")
	if len(cloudProfile.MachineImages) == 0 {
		allErrs = append(allErrs, field.Required(machineImagesPath, "must provide at least one machine image"))
	}
	for i, machineImage := range cloudProfile.MachineImages {
		idxPath := machineImagesPath.Index(i)
		allErrs = append(allErrs, ValidateMachineImage(idxPath, machineImage)...)
	}

	return allErrs
}

// ValidateMachineImage validates a CloudProfileConfig MachineImages entry.
func ValidateMachineImage(validationPath *field.Path, machineImage apisalicloud.MachineImages) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(machineImage.Name) == 0 {
		allErrs = append(allErrs, field.Required(validationPath.Child("name"), "must provide a name"))
	}

	if len(machineImage.Versions) == 0 {
		allErrs = append(allErrs, field.Required(validationPath.Child("versions"), fmt.Sprintf("must provide at least one version for machine image %q", machineImage.Name)))
	}
	for j, version := range machineImage.Versions {
		jdxPath := validationPath.Child("versions").Index(j)

		if len(version.Version) == 0 {
			allErrs = append(allErrs, field.Required(jdxPath.Child("version"), "must provide a version"))
		}

		if len(version.Regions) == 0 {
			allErrs = append(allErrs, field.Required(jdxPath.Child("regions"), fmt.Sprintf("must provide at least one region for machine image %q and version %q", machineImage.Name, version.Version)))
		}
		for k, region := range version.Regions {
			kdxPath := jdxPath.Child("regions").Index(k)
			if len(region.Name) == 0 {
				allErrs = append(allErrs, field.Required(kdxPath.Child("name"), "must provide a name"))
			}
			if len(region.ID) == 0 {
				allErrs = append(allErrs, field.Required(kdxPath.Child("id"), "must provide an id"))
			}
		}
	}

	return allErrs
}
