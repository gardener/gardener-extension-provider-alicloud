// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"
)

func (w *workerDelegate) GetMachineControllerManagerChartValues(_ context.Context) (map[string]interface{}, error) {
	return nil, nil
}

func (w *workerDelegate) GetMachineControllerManagerShootChartValues(_ context.Context) (map[string]interface{}, error) {
	return nil, nil
}
