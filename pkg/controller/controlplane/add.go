// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"sync/atomic"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/imagevector"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the Alicloud controlplane controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// ShootWebhookConfig specifies the desired Shoot MutatingWebhooksConfiguration.
	ShootWebhookConfig *atomic.Value
	// WebhookServerNamespace is the namespace in which the webhook server runs.
	WebhookServerNamespace string
	// CSI specifies the configuration for CSI components
	CSI config.CSI
	// ExtensionClass defines the extension class this extension is responsible for.
	ExtensionClass extensionsv1alpha1.ExtensionClass
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	genericActuator, err := genericactuator.NewActuator(
		mgr,
		alicloud.Name,
		secretConfigsFunc,
		shootAccessSecretsFunc,
		nil,
		nil,
		nil,
		controlPlaneChart,
		controlPlaneShootChart,
		controlPlaneShootCRDsChart,
		storageClassChart,
		nil,
		NewValuesProvider(mgr, opts.CSI),
		extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
		imagevector.ImageVector(),
		"",
		opts.ShootWebhookConfig,
		opts.WebhookServerNamespace,
	)
	if err != nil {
		return err
	}

	return controlplane.Add(ctx, mgr, controlplane.AddArgs{
		Actuator:          genericActuator,
		ControllerOptions: opts.Controller,
		Predicates:        controlplane.DefaultPredicates(ctx, mgr, opts.IgnoreOperationAnnotation),
		Type:              alicloud.Type,
		ExtensionClass:    opts.ExtensionClass,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}
