// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gardener/gardener/extensions/pkg/terraformer"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/gardener/gardener-extension-provider-alicloud/imagevector"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

const (
	terraformVarAccessKeyID     = "TF_VAR_ACCESS_KEY_ID"
	terraformVarAccessKeySecret = "TF_VAR_ACCESS_KEY_SECRET"
	terraformProvider           = "provider[\"registry.terraform.io/hashicorp/alicloud\"]"
	terraformProviderOld        = "provider.alicloud"
)

type tfState struct {
	Resources []tfStateResource `json:"resources"`
}

type tfStateResource struct {
	Mode      string        `json:"mode"`
	Type      string        `json:"type"`
	Name      string        `json:"name"`
	Provider  string        `json:"provider"`
	Instances []interface{} `json:"instances"`
}

// NewTerraformer creates a new Terraformer.
func NewTerraformer(
	logger logr.Logger,
	factory terraformer.Factory,
	config *rest.Config,
	purpose string,
	infra *extensionsv1alpha1.Infrastructure,
	disableProjectedTokenMount bool,
) (
	terraformer.Terraformer,
	error,
) {
	tf, err := factory.NewForConfig(logger, config, purpose, infra.Namespace, infra.Name, imagevector.TerraformerImage())
	if err != nil {
		return nil, err
	}

	owner := metav1.NewControllerRef(infra, extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.InfrastructureResource))
	return tf.
		UseProjectedTokenMount(!disableProjectedTokenMount).
		SetLogLevel("info").
		SetTerminationGracePeriodSeconds(630).
		SetDeadlineCleaning(5 * time.Minute).
		SetDeadlinePod(15 * time.Minute).
		SetOwnerRef(owner), nil
}

// NewTerraformerWithAuth creates a new Terraformer and initializes it with the credentials.
func NewTerraformerWithAuth(
	logger logr.Logger,
	factory terraformer.Factory,
	config *rest.Config,
	purpose string,
	infra *extensionsv1alpha1.Infrastructure,
	disableProjectedTokenMount bool,
) (
	terraformer.Terraformer,
	error,
) {
	tf, err := NewTerraformer(logger, factory, config, purpose, infra, disableProjectedTokenMount)
	if err != nil {
		return nil, err
	}

	return tf.SetEnvVars(TerraformerEnvVars(infra.Spec.SecretRef)...), nil
}

// TerraformerEnvVars computes the Terraformer environment variables from the given secret ref.
func TerraformerEnvVars(secretRef corev1.SecretReference) []corev1.EnvVar {
	return []corev1.EnvVar{{
		Name: terraformVarAccessKeyID,
		ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretRef.Name,
			},
			Key: alicloud.AccessKeyID,
		}},
	}, {
		Name: terraformVarAccessKeySecret,
		ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretRef.Name,
			},
			Key: alicloud.AccessKeySecret,
		}},
	}}
}

// IsStateEmpty checks the Terraformer state: 1. is empty or not; 2. contains resources or not
func IsStateEmpty(ctx context.Context, tf terraformer.Terraformer) (bool, error) {
	stateConfigMap, err := tf.GetState(ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if len(stateConfigMap) == 0 {
		return true, nil
	}

	var state tfState
	if err := json.Unmarshal(stateConfigMap, &state); err != nil {
		return false, err
	}

	for _, res := range state.Resources {
		if (res.Provider == terraformProvider || res.Provider == terraformProviderOld) && len(res.Instances) > 0 {
			return false, nil
		}
	}

	return true, nil
}
