// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"context"
	"strconv"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/flow"
	utilclient "github.com/gardener/gardener/pkg/utils/kubernetes/client"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/gardener/pkg/utils/version"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/kube-aggregator/pkg/controllers/autoregister"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Provider is the kubernetes provider label.
	Provider = "provider"
	// KubernetesProvider is the 'kubernetes' value of the Provider label.
	KubernetesProvider = "kubernetes"

	// KubeAggregatorAutoManaged is the label whether an APIService is automanaged by kube-aggregator.
	KubeAggregatorAutoManaged = autoregister.AutoRegisterManagedLabel

	// MetadataNameField ist the `metadata.name` field for a field selector.
	MetadataNameField = "metadata.name"
)

var (
	// FinalizeAfterFiveMinutes is an option to finalize resources after five minutes.
	FinalizeAfterFiveMinutes = utilclient.FinalizeGracePeriodSeconds(5 * 60)

	// FinalizeAfterOneHour is an option to finalize resources after one hour.
	FinalizeAfterOneHour = utilclient.FinalizeGracePeriodSeconds(60 * 60)

	// ZeroGracePeriod can be used for deleting resources with no grace period.
	ZeroGracePeriod = client.GracePeriodSeconds(0)
	// GracePeriodFiveMinutes can be used for deleting resources with a grace period of five minutes.
	GracePeriodFiveMinutes = client.GracePeriodSeconds(5 * 60)

	// NotSystemComponent is a requirement that something doesn't have the GardenRole GardenRoleSystemComponent.
	NotSystemComponent = utils.MustNewRequirement(v1beta1constants.GardenRole, selection.NotEquals, v1beta1constants.GardenRoleSystemComponent)
	// NoCleanupPrevention is a requirement that the ShootNoCleanup label of something is not true.
	NoCleanupPrevention = utils.MustNewRequirement(v1beta1constants.ShootNoCleanup, selection.NotEquals, "true")
	// NotKubernetesProvider is a requirement that the Provider label of something is not KubernetesProvider.
	NotKubernetesProvider = utils.MustNewRequirement(Provider, selection.NotEquals, KubernetesProvider)
	// NotKubeAggregatorAutoManaged is a requirement that something is not auto-managed by Kube-Aggregator.
	NotKubeAggregatorAutoManaged = utils.MustNewRequirement(KubeAggregatorAutoManaged, selection.DoesNotExist)

	// CleanupSelector is a selector that excludes system components and all resources not considered for auto cleanup.
	CleanupSelector = labels.NewSelector().Add(NotSystemComponent).Add(NoCleanupPrevention)

	// NoCleanupPreventionListOption are CollectionMatching that exclude system components or non-auto cleaned up resource.
	NoCleanupPreventionListOption = client.MatchingLabelsSelector{Selector: CleanupSelector}

	// MutatingWebhookConfigurationCleanOption is the delete selector for MutatingWebhookConfigurations.
	MutatingWebhookConfigurationCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// ValidatingWebhookConfigurationCleanOption is the delete selector for ValidatingWebhookConfigurations.
	ValidatingWebhookConfigurationCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// CustomResourceDefinitionCleanOption is the delete selector for CustomResources.
	CustomResourceDefinitionCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// DaemonSetCleanOption is the delete selector for DaemonSets.
	DaemonSetCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// DeploymentCleanOption is the delete selector for Deployments.
	DeploymentCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// StatefulSetCleanOption is the delete selector for StatefulSets.
	StatefulSetCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// ServiceCleanOption is the delete selector for Services.
	ServiceCleanOption = utilclient.ListWith{
		client.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(NotKubernetesProvider, NotSystemComponent, NoCleanupPrevention),
		},
	}

	// NamespaceMatchingLabelsSelector is the delete label selector for Namespaces.
	NamespaceMatchingLabelsSelector = utilclient.ListWith{&NoCleanupPreventionListOption}

	// NamespaceMatchingFieldsSelector is the delete field selector for Namespaces.
	NamespaceMatchingFieldsSelector = utilclient.ListWith{
		client.MatchingFieldsSelector{
			Selector: fields.AndSelectors(
				fields.OneTermNotEqualSelector(MetadataNameField, metav1.NamespacePublic),
				fields.OneTermNotEqualSelector(MetadataNameField, metav1.NamespaceSystem),
				fields.OneTermNotEqualSelector(MetadataNameField, metav1.NamespaceDefault),
				fields.OneTermNotEqualSelector(MetadataNameField, corev1.NamespaceNodeLease),
			),
		},
	}

	// APIServiceCleanOption is the delete selector for APIServices.
	APIServiceCleanOption = utilclient.ListWith{
		client.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(NotSystemComponent, NotKubeAggregatorAutoManaged),
		},
	}

	// CronJobCleanOption is the delete selector for CronJobs.
	CronJobCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// IngressCleanOption is the delete selector for Ingresses.
	IngressCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// JobCleanOption is the delete selector for Jobs.
	JobCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// PodCleanOption is the delete selector for Pods.
	PodCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// ReplicaSetCleanOption is the delete selector for ReplicaSets.
	ReplicaSetCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// ReplicationControllerCleanOption is the delete selector for ReplicationControllers.
	ReplicationControllerCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// PersistentVolumeClaimCleanOption is the delete selector for PersistentVolumeClaims.
	PersistentVolumeClaimCleanOption = utilclient.ListWith{&NoCleanupPreventionListOption}

	// NamespaceErrorToleration are the errors to be tolerated during deletion.
	NamespaceErrorToleration = utilclient.TolerateErrors{apierrors.IsConflict}
)

func cleanResourceFn(cleanOps utilclient.CleanOps, c client.Client, list client.ObjectList, opts ...utilclient.CleanOption) flow.TaskFn {
	return func(ctx context.Context) error {
		return retry.Until(ctx, DefaultInterval, func(ctx context.Context) (done bool, err error) {
			if err := cleanOps.CleanAndEnsureGone(ctx, c, list, opts...); err != nil {
				if utilclient.AreObjectsRemaining(err) {
					return retry.MinorError(helper.NewErrorWithCodes(err.Error(), gardencorev1beta1.ErrorCleanupClusterResources))
				}
				return retry.SevereError(err)
			}
			return retry.Ok()
		})
	}
}

// CleanWebhooks deletes all Webhooks in the Shoot cluster that are not being managed by the addon manager.
func (b *Botanist) CleanWebhooks(ctx context.Context) error {
	var (
		c       = b.K8sShootClient.Client()
		ensurer = utilclient.GoneBeforeEnsurer(b.Shoot.GetInfo().GetDeletionTimestamp().Time)
		ops     = utilclient.NewCleanOps(utilclient.DefaultCleaner(), ensurer)
	)

	cleanOptions, err := b.getCleanOptions(GracePeriodFiveMinutes, FinalizeAfterFiveMinutes, v1beta1constants.AnnotationShootCleanupWebhooksFinalizeGracePeriodSeconds, 1)
	if err != nil {
		return err
	}

	if version.ConstraintK8sLessEqual115.Check(b.Shoot.KubernetesVersion) {
		return flow.Parallel(
			cleanResourceFn(ops, c, &admissionregistrationv1beta1.MutatingWebhookConfigurationList{}, MutatingWebhookConfigurationCleanOption, cleanOptions),
			cleanResourceFn(ops, c, &admissionregistrationv1beta1.ValidatingWebhookConfigurationList{}, ValidatingWebhookConfigurationCleanOption, cleanOptions),
		)(ctx)
	}

	return flow.Parallel(
		cleanResourceFn(ops, c, &admissionregistrationv1.MutatingWebhookConfigurationList{}, MutatingWebhookConfigurationCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &admissionregistrationv1.ValidatingWebhookConfigurationList{}, ValidatingWebhookConfigurationCleanOption, cleanOptions),
	)(ctx)
}

// CleanExtendedAPIs removes API extensions like CRDs and API services from the Shoot cluster.
func (b *Botanist) CleanExtendedAPIs(ctx context.Context) error {
	var (
		c           = b.K8sShootClient.Client()
		ensurer     = utilclient.GoneBeforeEnsurer(b.Shoot.GetInfo().GetDeletionTimestamp().Time)
		defaultOps  = utilclient.NewCleanOps(utilclient.DefaultCleaner(), ensurer)
		crdCleanOps = utilclient.NewCleanOps(utilclient.DefaultCleaner(), ensurer)
	)

	cleanOptions, err := b.getCleanOptions(GracePeriodFiveMinutes, FinalizeAfterOneHour, v1beta1constants.AnnotationShootCleanupExtendedAPIsFinalizeGracePeriodSeconds, 0.1)
	if err != nil {
		return err
	}

	var crdList client.ObjectList = &apiextensionsv1.CustomResourceDefinitionList{}

	if version.ConstraintK8sLessEqual115.Check(b.Shoot.KubernetesVersion) {
		crdList = &apiextensionsv1beta1.CustomResourceDefinitionList{}
	}

	return flow.Parallel(
		cleanResourceFn(defaultOps, c, &apiregistrationv1.APIServiceList{}, APIServiceCleanOption, cleanOptions),
		cleanResourceFn(crdCleanOps, c, crdList, CustomResourceDefinitionCleanOption, cleanOptions),
	)(ctx)
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func (b *Botanist) CleanKubernetesResources(ctx context.Context) error {
	var (
		c       = b.K8sShootClient.Client()
		ensurer = utilclient.GoneBeforeEnsurer(b.Shoot.GetInfo().GetDeletionTimestamp().Time)
		ops     = utilclient.NewCleanOps(utilclient.DefaultCleaner(), ensurer)
	)

	cleanOptions, err := b.getCleanOptions(GracePeriodFiveMinutes, FinalizeAfterFiveMinutes, v1beta1constants.AnnotationShootCleanupKubernetesResourcesFinalizeGracePeriodSeconds, 1)
	if err != nil {
		return err
	}

	if metav1.HasAnnotation(b.Shoot.GetInfo().ObjectMeta, v1beta1constants.AnnotationShootSkipCleanup) {
		return flow.Parallel(
			cleanResourceFn(ops, c, &corev1.ServiceList{}, ServiceCleanOption, cleanOptions),
			cleanResourceFn(ops, c, &corev1.PersistentVolumeClaimList{}, PersistentVolumeClaimCleanOption, cleanOptions),
		)(ctx)
	}

	var ingressList client.ObjectList = &networkingv1.IngressList{}
	if version.ConstraintK8sLess119.Check(b.Shoot.KubernetesVersion) {
		ingressList = &extensionsv1beta1.IngressList{}
	}

	return flow.Parallel(
		cleanResourceFn(ops, c, &batchv1beta1.CronJobList{}, CronJobCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &appsv1.DaemonSetList{}, DaemonSetCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &appsv1.DeploymentList{}, DeploymentCleanOption, cleanOptions),
		cleanResourceFn(ops, c, ingressList, IngressCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &batchv1.JobList{}, JobCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &corev1.PodList{}, PodCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &appsv1.ReplicaSetList{}, ReplicaSetCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &corev1.ReplicationControllerList{}, ReplicationControllerCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &corev1.ServiceList{}, ServiceCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &appsv1.StatefulSetList{}, StatefulSetCleanOption, cleanOptions),
		cleanResourceFn(ops, c, &corev1.PersistentVolumeClaimList{}, PersistentVolumeClaimCleanOption, cleanOptions),
	)(ctx)
}

// CleanShootNamespaces deletes all non-system namespaces in the Shoot cluster.
// It assumes that all workload resources are cleaned up in previous step(s).
func (b *Botanist) CleanShootNamespaces(ctx context.Context) error {
	var (
		c                 = b.K8sShootClient.Client()
		namespaceCleaner  = utilclient.NewNamespaceCleaner(b.K8sShootClient.Kubernetes().CoreV1().Namespaces())
		namespaceCleanOps = utilclient.NewCleanOps(namespaceCleaner, utilclient.DefaultGoneEnsurer())
	)

	cleanOptions, err := b.getCleanOptions(ZeroGracePeriod, FinalizeAfterFiveMinutes, v1beta1constants.AnnotationShootCleanupNamespaceResourcesFinalizeGracePeriodSeconds, 0)
	if err != nil {
		return err
	}

	return cleanResourceFn(namespaceCleanOps, c, &corev1.NamespaceList{}, cleanOptions, NamespaceMatchingLabelsSelector, NamespaceMatchingFieldsSelector, NamespaceErrorToleration)(ctx)
}

// CleanVolumeAttachments cleans up all VolumeAttachments in the cluster, waits for them to be gone and finalizes any
// remaining ones after five minutes.
func CleanVolumeAttachments(ctx context.Context, c client.Client) error {
	return cleanResourceFn(utilclient.DefaultCleanOps(), c, &storagev1.VolumeAttachmentList{}, utilclient.DeleteWith{ZeroGracePeriod}, FinalizeAfterFiveMinutes)(ctx)
}

func (b *Botanist) getCleanOptions(
	defaultGracePeriodSeconds client.GracePeriodSeconds,
	defaultFinalizeAfter utilclient.FinalizeGracePeriodSeconds,
	annotationKey string,
	gracePeriodSecondsFactor float64,
) (
	*utilclient.CleanOptions,
	error,
) {
	var (
		gracePeriodSeconds = defaultGracePeriodSeconds
		finalizeAfter      = defaultFinalizeAfter
	)

	if v, ok := b.Shoot.GetInfo().Annotations[annotationKey]; ok {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}

		if int64(seconds) < int64(defaultFinalizeAfter) {
			gracePeriodSeconds = client.GracePeriodSeconds(int(float64(seconds) * gracePeriodSecondsFactor))
			finalizeAfter = utilclient.FinalizeGracePeriodSeconds(seconds)
		}
	}

	cleanOpts := &utilclient.CleanOptions{}
	utilclient.DeleteWith{gracePeriodSeconds}.ApplyToClean(cleanOpts)
	finalizeAfter.ApplyToClean(cleanOpts)

	return cleanOpts, nil
}
