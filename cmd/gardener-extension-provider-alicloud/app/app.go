// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"
	"github.com/gardener/gardener/extensions/pkg/util"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	gardenerhealthz "github.com/gardener/gardener/pkg/healthz"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudinstall "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	alicloudcmd "github.com/gardener/gardener-extension-provider-alicloud/pkg/cmd"
	alicloudbackupbucket "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/backupbucket"
	alicloudbackupentry "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/backupentry"
	alicloudbastion "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/bastion"
	alicloudcontrolplane "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/controlplane"
	aliclouddnsrecord "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/dnsrecord"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/healthcheck"
	alicloudinfrastructure "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
	alicloudworker "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/worker"
	alicloudcontrolplaneexposure "github.com/gardener/gardener-extension-provider-alicloud/pkg/webhook/controlplaneexposure"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/webhook/shoot"
)

// NewControllerManagerCommand creates a new command for running a Alicloud provider controller.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	var (
		generalOpts = &controllercmd.GeneralOptions{}
		restOpts    = &controllercmd.RESTOptions{}
		mgrOpts     = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(alicloud.Name),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
			WebhookServerPort:       443,
			WebhookCertDir:          "/tmp/gardener-extensions-cert",
			MetricsBindAddress:      ":8080",
			HealthBindAddress:       ":8081",
		}
		configFileOpts = &alicloudcmd.ConfigOptions{}

		// options for the backupbucket controller
		backupBucketCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the backupentry controller
		backupEntryCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the bastion controller
		bastionCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the health care controller
		healthCheckCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the heartbeat controller
		heartbeatCtrlOpts = &heartbeatcmd.Options{
			ExtensionName:        alicloud.Name,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}

		// options for the controlplane controller
		controlPlaneCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the dnsrecord controller
		dnsRecordCtrlOpts = &alicloudcmd.DNSRecordControllerOptions{
			ControllerOptions: controllercmd.ControllerOptions{
				MaxConcurrentReconciles: 5,
			},
			ProviderClientQPS:         25,
			ProviderClientBurst:       1,
			ProviderClientWaitTimeout: 2 * time.Second,
		}

		// options for the infrastructure controller
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		reconcileOpts = &controllercmd.ReconcilerOptions{}

		// options for the worker controller
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the webhook server
		webhookServerOptions = &webhookcmd.ServerOptions{
			Namespace: os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
		}

		controllerSwitches = alicloudcmd.ControllerSwitchOptions()
		webhookSwitches    = alicloudcmd.WebhookSwitchOptions()
		webhookOptions     = webhookcmd.NewAddToManagerOptions(
			alicloud.Name,
			genericactuator.ShootWebhooksResourceName,
			genericactuator.ShootWebhookNamespaceSelector(alicloud.Type),
			webhookServerOptions,
			webhookSwitches,
		)

		aggOption = controllercmd.NewOptionAggregator(
			generalOpts,
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("backupbucket-", backupBucketCtrlOpts),
			controllercmd.PrefixOption("backupentry-", backupEntryCtrlOpts),
			controllercmd.PrefixOption("bastion-", bastionCtrlOpts),
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("dnsrecord-", dnsRecordCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", infraCtrlOpts),
			controllercmd.PrefixOption("worker-", workerCtrlOpts),
			controllercmd.PrefixOption("healthcheck-", healthCheckCtrlOpts),
			controllercmd.PrefixOption("heartbeat-", heartbeatCtrlOpts),
			configFileOpts,
			controllerSwitches,
			reconcileOpts,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s-controller-manager", alicloud.Name),

		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := aggOption.Complete(); err != nil {
				return fmt.Errorf("error completing options: %w", err)
			}

			if err := heartbeatCtrlOpts.Validate(); err != nil {
				return err
			}

			util.ApplyClientConnectionConfigurationToRESTConfig(configFileOpts.Completed().Config.ClientConnection, restOpts.Completed().Config)

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				return fmt.Errorf("could not instantiate manager: %w", err)
			}

			scheme := mgr.GetScheme()
			if err := controller.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := alicloudinstall.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := druidv1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := autoscalingv1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}
			if err := machinev1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			// add common meta types to schema for controller-runtime to use v1.ListOptions
			metav1.AddToGroupVersion(scheme, machinev1alpha1.SchemeGroupVersion)

			log := mgr.GetLogger()
			log.Info("Getting rest config for garden")
			gardenRESTConfig, err := kubernetes.RESTConfigFromKubeconfigFile(os.Getenv("GARDEN_KUBECONFIG"), kubernetes.AuthTokenFile)
			if err != nil {
				return err
			}

			log.Info("Setting up cluster object for garden")
			gardenCluster, err := cluster.New(gardenRESTConfig, func(opts *cluster.Options) {
				opts.Scheme = kubernetes.GardenScheme
				opts.Logger = log
			})
			if err != nil {
				return fmt.Errorf("failed creating garden cluster object: %w", err)
			}

			log.Info("Adding garden cluster to manager")
			if err := mgr.Add(gardenCluster); err != nil {
				return fmt.Errorf("failed adding garden cluster to manager: %w", err)
			}

			log.Info("Adding controllers to manager")
			configFileOpts.Completed().ApplyMachineImageOwnerSecretRef(&alicloudinfrastructure.DefaultAddOptions.MachineImageOwnerSecretRef)
			configFileOpts.Completed().ApplyToBeSharedImageIDs(&alicloudinfrastructure.DefaultAddOptions.ToBeSharedImageIDs)
			configFileOpts.Completed().ApplyETCDStorage(&alicloudcontrolplaneexposure.DefaultAddOptions.ETCDStorage)
			configFileOpts.Completed().ApplyService(&shoot.DefaultAddOptions.Service)
			configFileOpts.Completed().ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckConfig)
			configFileOpts.Completed().ApplyCSI(&alicloudcontrolplane.DefaultAddOptions.CSI)
			healthCheckCtrlOpts.Completed().Apply(&healthcheck.DefaultAddOptions.Controller)
			heartbeatCtrlOpts.Completed().Apply(&heartbeat.DefaultAddOptions)
			backupBucketCtrlOpts.Completed().Apply(&alicloudbackupbucket.DefaultAddOptions.Controller)
			backupEntryCtrlOpts.Completed().Apply(&alicloudbackupentry.DefaultAddOptions.Controller)
			bastionCtrlOpts.Completed().Apply(&alicloudbastion.DefaultAddOptions.Controller)
			controlPlaneCtrlOpts.Completed().Apply(&alicloudcontrolplane.DefaultAddOptions.Controller)
			dnsRecordCtrlOpts.Completed().Apply(&aliclouddnsrecord.DefaultAddOptions.Controller)
			dnsRecordCtrlOpts.Completed().ApplyRateLimiter(&aliclouddnsrecord.DefaultAddOptions.RateLimiter)
			infraCtrlOpts.Completed().Apply(&alicloudinfrastructure.DefaultAddOptions.Controller)
			reconcileOpts.Completed().Apply(&alicloudinfrastructure.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&alicloudcontrolplane.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&alicloudworker.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&alicloudbastion.DefaultAddOptions.IgnoreOperationAnnotation)
			workerCtrlOpts.Completed().Apply(&alicloudworker.DefaultAddOptions.Controller)

			alicloudworker.DefaultAddOptions.GardenCluster = gardenCluster

			atomicShootWebhookConfig, err := webhookOptions.Completed().AddToManager(ctx, mgr, nil)
			if err != nil {
				return fmt.Errorf("could not add webhooks to manager: %w", err)
			}
			alicloudcontrolplane.DefaultAddOptions.ShootWebhookConfig = atomicShootWebhookConfig
			alicloudcontrolplane.DefaultAddOptions.WebhookServerNamespace = webhookOptions.Server.Namespace

			if err := controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
				return fmt.Errorf("could not add controllers to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("informer-sync", gardenerhealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
				return fmt.Errorf("could not add readycheck for informers: %w", err)
			}

			if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
				return fmt.Errorf("could not add health check to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("webhook-server", mgr.GetWebhookServer().StartedChecker()); err != nil {
				return fmt.Errorf("could not add ready check for webhook server to manager: %w", err)
			}

			if err := mgr.Start(ctx); err != nil {
				return fmt.Errorf("error running manager: %w", err)
			}

			return nil
		},
	}

	verflag.AddFlags(cmd.Flags())
	aggOption.AddFlags(cmd.Flags())

	return cmd
}
