// Copyright (c) 2025 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package cloudprovider_test

import (
	"context"
	"testing"

	"github.com/gardener/gardener/extensions/pkg/webhook/cloudprovider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/webhook/cloudprovider"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudProvider Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	var (
		logger = log.Log.WithName("alicloud-cloudprovider-webhook-test")
		ctx    = context.TODO()

		ensurer cloudprovider.Ensurer

		secret *corev1.Secret
	)

	BeforeEach(func() {
		secret = &corev1.Secret{
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte("access-key-id"),
				alicloud.AccessKeySecret: []byte("access-key-secret"),
			},
		}

		ensurer = NewEnsurer(logger)
	})

	Describe("#EnsureCloudProviderSecret", func() {
		It("should fail as no accessKeyID is present", func() {
			delete(secret.Data, alicloud.AccessKeyID)
			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)
			Expect(err).To(MatchError(ContainSubstring("could not mutate cloudprovider secret as %q field is missing", alicloud.AccessKeyID)))
		})
		It("should fail as no secretAccessKey is present", func() {
			delete(secret.Data, alicloud.AccessKeySecret)
			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)
			Expect(err).To(MatchError(ContainSubstring("could not mutate cloudprovider secret as %q field is missing", alicloud.AccessKeySecret)))
		})
		It("should replace esixting credentials file", func() {
			secret.Data[alicloud.CredentialsFile] = []byte("shared-credentials-file")

			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data).To(Equal(map[string][]byte{
				alicloud.AccessKeyID:     []byte("access-key-id"),
				alicloud.AccessKeySecret: []byte("access-key-secret"),
				alicloud.CredentialsFile: []byte(`[default]
enable = true
type = access_key
access_key_id = access-key-id
access_key_secret = access-key-secret`),
			}))
		})
		It("should add credentials file", func() {
			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data).To(Equal(map[string][]byte{
				alicloud.AccessKeyID:     []byte("access-key-id"),
				alicloud.AccessKeySecret: []byte("access-key-secret"),
				alicloud.CredentialsFile: []byte(`[default]
enable = true
type = access_key
access_key_id = access-key-id
access_key_secret = access-key-secret`),
			}))
		})
	})
})
