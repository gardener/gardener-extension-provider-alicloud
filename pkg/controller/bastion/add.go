// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	aliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are Options to apply when adding the Azure bastion controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	return bastion.Add(mgr, bastion.AddArgs{
		Actuator:          newActuator(mgr),
		ConfigValidator:   NewConfigValidator(mgr, aliclient.NewClientFactory()),
		ControllerOptions: opts.Controller,
		Predicates:        bastion.DefaultPredicates(opts.IgnoreOperationAnnotation),
		Type:              alicloud.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(_ context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
