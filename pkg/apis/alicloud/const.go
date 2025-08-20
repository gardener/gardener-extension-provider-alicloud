// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

const (
	// AnnotationKeyUseFlow is the annotation key used to enable reconciliation with flow instead of terraformer.
	AnnotationKeyUseFlow = "alicloud.provider.extensions.gardener.cloud/use-flow"
	// GlobalAnnotationKeyUseFlow is the annotation key used to enable reconciliation with flow instead of terraformer.
	GlobalAnnotationKeyUseFlow = "provider.extensions.gardener.cloud/use-flow"
	// SeedAnnotationKeyUseFlow is the label for seeds to enable flow reconciliation for all of its shoots if value is `true`
	// or for new shoots only with value `new`
	SeedAnnotationKeyUseFlow = AnnotationKeyUseFlow
	// SeedAnnotationUseFlowValueNew is the value to restrict flow reconciliation to new shoot clusters
	SeedAnnotationUseFlowValueNew = "new"
)

var (
	// ValidFlowAnnotations is a list of all the available annotations to indicate the use of the flow reconciler.
	ValidFlowAnnotations = []string{AnnotationKeyUseFlow, GlobalAnnotationKeyUseFlow}
)
