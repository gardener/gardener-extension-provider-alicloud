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

package infrastructure

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the infrastructure controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// MachineImageOwnerSecretRef is the secret reference which contains credential of AliCloud subaccount for customized images.
	MachineImageOwnerSecretRef *corev1.SecretReference
	// ToBeSharedImageIDs specifies custom image IDs which need to be shared by shoots
	ToBeSharedImageIDs []string
	// DisableProjectedTokenMount specifies whether the projected token mount shall be disabled for the terraformer.
	// Used for testing only.
	DisableProjectedTokenMount bool
}

// AddToManagerWithOptions adds a controller with the given AddOptions to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, options AddOptions) error {
	actuator, err := NewActuator(mgr, options.MachineImageOwnerSecretRef, options.ToBeSharedImageIDs, options.DisableProjectedTokenMount)
	if err != nil {
		return err
	}

	return infrastructure.Add(ctx, mgr, infrastructure.AddArgs{
		Actuator:          actuator,
		ConfigValidator:   NewConfigValidator(mgr, log.Log, aliclient.FactoryFunc(aliclient.NewActor)),
		ControllerOptions: options.Controller,
		Predicates:        infrastructure.DefaultPredicates(ctx, mgr, options.IgnoreOperationAnnotation),
		Type:              alicloud.Type,
	})
}

// AddToManager adds a controller with the default AddOptions.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}
