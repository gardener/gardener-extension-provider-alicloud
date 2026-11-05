// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	alicloudvalidation "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

type workloadIdentity struct{}

// NewWorkloadIdentityValidator returns a validator for WorkloadIdentity resources.
func NewWorkloadIdentityValidator() extensionswebhook.Validator {
	return &workloadIdentity{}
}

func (w *workloadIdentity) Validate(_ context.Context, newObj, oldObj client.Object) error {
	fldPath := field.NewPath("spec", "targetSystem", "providerConfig")
	return ValidateWorkloadIdentity(newObj, oldObj, fldPath).ToAggregate()
}

// ValidateWorkloadIdentity validates a WorkloadIdentity resource.
// Exported for reuse by the credentialsbinding and shoot validators.
func ValidateWorkloadIdentity(newObj, oldObj client.Object, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	wid, ok := newObj.(*securityv1alpha1.WorkloadIdentity)
	if !ok {
		return append(allErrs, field.InternalError(fldPath, fmt.Errorf("expected *securityv1alpha1.WorkloadIdentity, got %T", newObj)))
	}

	if wid.Spec.TargetSystem.ProviderConfig == nil {
		return append(allErrs, field.Required(fldPath, "the target system is missing configuration"))
	}

	newConfig, err := helper.WorkloadIdentityConfigFromRaw(wid.Spec.TargetSystem.ProviderConfig)
	if err != nil {
		return append(allErrs, field.InternalError(fldPath, fmt.Errorf("cannot decode target system configuration: %w", err)))
	}

	if oldObj != nil {
		oldWID := oldObj.(*securityv1alpha1.WorkloadIdentity)
		if oldWID.Spec.TargetSystem.ProviderConfig != nil {
			oldConfig, err := helper.WorkloadIdentityConfigFromRaw(oldWID.Spec.TargetSystem.ProviderConfig)
			if err != nil {
				return append(allErrs, field.InternalError(fldPath, fmt.Errorf("cannot decode old target system configuration: %w", err)))
			}
			return alicloudvalidation.ValidateWorkloadIdentityConfigUpdate(oldConfig, newConfig, fldPath)
		}
	}

	return alicloudvalidation.ValidateWorkloadIdentityConfig(newConfig, fldPath)
}
