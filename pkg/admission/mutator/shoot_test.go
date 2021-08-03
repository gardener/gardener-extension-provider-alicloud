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

	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

var _ = Describe("Mutating Shoot", func() {
	var (
		oldShoot *corev1beta1.Shoot
		newShoot *corev1beta1.Shoot
		shoot    = &shootMutator{}
		ctx      = context.TODO()
	)

	BeforeEach(func() {
		oldShoot = &corev1beta1.Shoot{
			Spec: corev1beta1.ShootSpec{
				Provider: corev1beta1.Provider{
					Workers: []corev1beta1.Worker{
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    "GardenLinux",
									Version: pointer.StringPtr("1.0"),
								},
							},
							Volume: &corev1beta1.Volume{
								Encrypted: pointer.BoolPtr(true),
							},
						},
					},
				},
			},
		}

		newShoot = &corev1beta1.Shoot{
			Spec: corev1beta1.ShootSpec{
				Provider: corev1beta1.Provider{
					Workers: []corev1beta1.Worker{
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    "GardenLinux",
									Version: pointer.StringPtr("1.0"),
								},
							},
							Volume: &corev1beta1.Volume{},
						},
						{
							Machine: corev1beta1.Machine{
								Image: &corev1beta1.ShootMachineImage{
									Name:    "GardenLinux",
									Version: pointer.StringPtr("1.0"),
								},
							},
						},
					},
				},
			},
		}
	})
	Context("#Encrypted System Disk", func() {
		It("should not reconcile infra if no system disk is encrypted", func() {
			err := shoot.Mutate(ctx, oldShoot, newShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(!checkReconcileInfraAnnotation(newShoot.Annotations))
		})

		It("should not reconcile infra if system disk is already encrypted", func() {
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, oldShoot, newShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(!checkReconcileInfraAnnotation(newShoot.Annotations))
		})

		It("should not reconcile infra if new version of machine is not encrypted", func() {
			newShoot.Spec.Provider.Workers[1].Machine.Image.Version = pointer.StringPtr("2.0")
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, oldShoot, newShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(!checkReconcileInfraAnnotation(newShoot.Annotations))
		})

		It("should reconcile infra if new version of machine is added and it is encrypted", func() {
			newShoot.Spec.Provider.Workers[0].Machine.Image.Version = pointer.StringPtr("2.0")
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, oldShoot, newShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(checkReconcileInfraAnnotation(newShoot.Annotations))
		})

		It("should reconcile infra if machine is changed to be encrypted", func() {
			oldShoot.Spec.Provider.Workers[0].Volume.Encrypted = nil
			newShoot.Spec.Provider.Workers[0].Volume.Encrypted = pointer.BoolPtr(true)
			err := shoot.Mutate(ctx, oldShoot, newShoot)
			Expect(err).NotTo(HaveOccurred())
			Expect(checkReconcileInfraAnnotation(newShoot.Annotations))
		})
	})
})

func checkReconcileInfraAnnotation(annotations map[string]string) bool {
	tasks := controllerutils.GetTasks(annotations)
	for _, t := range tasks {
		if t == v1beta1constants.ShootTaskDeployInfrastructure {
			return true
		}
	}

	return false
}
