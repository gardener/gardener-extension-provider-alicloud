// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package mutator

import (
	"context"
	"fmt"
	"reflect"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ShootMutatorName = "shoots.mutator"
const MutatorPath = "/webhooks/mutate"

// NewShootMutator returns a new instance of a shoot validator.
func NewShootMutator() extensionswebhook.Mutator {
	return &shootMutator{}
}

type shootMutator struct {
}

func (s *shootMutator) Mutate(_ context.Context, new, old client.Object) error {
	shoot, ok := new.(*corev1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if old != nil {
		oldShoot, ok := old.(*corev1beta1.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", old)
		}
		return s.mutateShootUpdate(oldShoot, shoot)
	}

	// Only consider update for now.
	return nil
}

func (s *shootMutator) mutateShootUpdate(oldShoot, shoot *corev1beta1.Shoot) error {
	if !equality.Semantic.DeepEqual(oldShoot.Spec, shoot.Spec) {
		s.mutateForEncryptedSystemDiskChange(oldShoot, shoot)
	}

	return nil
}

func (s *shootMutator) mutateForEncryptedSystemDiskChange(oldShoot, shoot *corev1beta1.Shoot) {
	if requireNewEncryptedImage(oldShoot.Spec.Provider.Workers, shoot.Spec.Provider.Workers) {
		logger.Info("Need to reconcile infra as new encrypted system disk found in workers", "name", shoot.Name, "namespace", shoot.Namespace)
		if shoot.Annotations == nil {
			shoot.Annotations = make(map[string]string)
		}

		controllerutils.AddTasks(shoot.Annotations, v1beta1constants.ShootTaskDeployInfrastructure)
	}
}

// Check encrypted flag in new workers' volumes. If it is changed to be true, check for old workers
// if there is already a volume is set to be encrypted and also the OS version is the same.
func requireNewEncryptedImage(oldWorkers, newWorkers []corev1beta1.Worker) bool {
	var imagesEncrypted []*corev1beta1.ShootMachineImage
	for _, w := range oldWorkers {
		if w.Volume != nil && w.Volume.Encrypted != nil && *w.Volume.Encrypted {
			if w.Machine.Image != nil {
				imagesEncrypted = append(imagesEncrypted, w.Machine.Image)
			}
		}
	}

	for _, w := range newWorkers {
		if w.Volume != nil && w.Volume.Encrypted != nil && *w.Volume.Encrypted {
			if w.Machine.Image != nil {
				found := false
				for _, image := range imagesEncrypted {
					if w.Machine.Image.Name == image.Name && reflect.DeepEqual(w.Machine.Image.Version, image.Version) {
						found = true
						break
					}
				}

				if !found {
					return true
				}
			}
		}
	}

	return false
}
