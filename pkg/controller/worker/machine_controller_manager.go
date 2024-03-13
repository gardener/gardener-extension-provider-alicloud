// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
)

func (w *workerDelegate) GetMachineControllerManagerChartValues(ctx context.Context) (map[string]interface{}, error) {
	namespace := &corev1.Namespace{}
	if err := w.client.Get(ctx, kutil.Key(w.worker.Namespace), namespace); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"providerName": alicloud.Name,
		"namespace": map[string]interface{}{
			"uid": namespace.UID,
		},
		"podLabels": map[string]interface{}{
			v1beta1constants.LabelPodMaintenanceRestart: "true",
		},
	}, nil
}

func (w *workerDelegate) GetMachineControllerManagerShootChartValues(_ context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"providerName": alicloud.Name,
	}, nil
}
