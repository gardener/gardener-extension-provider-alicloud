// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/security"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	alicloudvalidation "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

type credentialsBinding struct {
	apiReader client.Reader
}

// NewCredentialsBindingValidator returns a new instance of a credentials binding validator.
func NewCredentialsBindingValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &credentialsBinding{
		apiReader: mgr.GetAPIReader(),
	}
}

// Validate checks whether the given CredentialsBinding refers to valid Alicloud credentials.
func (cb *credentialsBinding) Validate(ctx context.Context, newObj, oldObj client.Object) error {
	credentialsBinding, ok := newObj.(*security.CredentialsBinding)
	if !ok {
		return fmt.Errorf("wrong object type %T", newObj)
	}

	if oldObj != nil {
		_, ok := oldObj.(*security.CredentialsBinding)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", oldObj)
		}

		// The relevant fields of the credentials binding are immutable so we can exit early on update
		return nil
	}

	// Explicitly use the client.Reader to prevent controller-runtime to start Informer for Secrets/WorkloadIdentities
	// under the hood. The latter increases the memory usage of the component.
	var credentialsKey = client.ObjectKey{Namespace: credentialsBinding.CredentialsRef.Namespace, Name: credentialsBinding.CredentialsRef.Name}
	switch {
	case credentialsBinding.CredentialsRef.APIVersion == corev1.SchemeGroupVersion.String() && credentialsBinding.CredentialsRef.Kind == "Secret":
		secret := &corev1.Secret{}
		if err := cb.apiReader.Get(ctx, credentialsKey, secret); err != nil {
			return err
		}

		return alicloudvalidation.ValidateCloudProviderSecret(secret)
	case credentialsBinding.CredentialsRef.APIVersion == securityv1alpha1.SchemeGroupVersion.String() && credentialsBinding.CredentialsRef.Kind == "WorkloadIdentity":
		wi := &securityv1alpha1.WorkloadIdentity{}
		if err := cb.apiReader.Get(ctx, credentialsKey, wi); err != nil {
			return err
		}

		fldPath := field.NewPath("spec", "targetSystem", "providerConfig")
		if errList := ValidateWorkloadIdentity(wi, nil, fldPath); len(errList) > 0 {
			return fmt.Errorf("referenced workload identity %s is not valid: %w", credentialsKey, errList.ToAggregate())
		}
		return nil
	default:
		return fmt.Errorf("unsupported credentials reference: version %q, kind %q", credentialsBinding.CredentialsRef.APIVersion, credentialsBinding.CredentialsRef.Kind)
	}
}
