// SPDX-FileCopyrightText: Contributors to the Gardener project
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
	"strings"
	"testing"

	"github.com/gardener/gardener/extensions/pkg/webhook/cloudprovider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			ObjectMeta: metav1.ObjectMeta{Name: "cloudprovider", Namespace: "shoot--test--alicloud"},
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte("abcdefghijklmnop"),
				alicloud.AccessKeySecret: []byte("abcdefghijklmnopqrstuvwxyz0123"),
			},
		}

		ensurer = NewEnsurer(logger)
	})

	Describe("#EnsureCloudProviderSecret", func() {
		It("should fail as no accessKeyID is present", func() {
			delete(secret.Data, alicloud.AccessKeyID)
			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)
			Expect(err).To(MatchError(ContainSubstring("missing %q field", alicloud.AccessKeyID)))
		})
		It("should fail as no secretAccessKey is present", func() {
			delete(secret.Data, alicloud.AccessKeySecret)
			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)
			Expect(err).To(MatchError(ContainSubstring("missing %q field", alicloud.AccessKeySecret)))
		})
		It("should replace existing credentials file", func() {
			secret.Data[alicloud.CredentialsFile] = []byte("shared-credentials-file")

			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data).To(Equal(map[string][]byte{
				alicloud.AccessKeyID:     []byte("abcdefghijklmnop"),
				alicloud.AccessKeySecret: []byte("abcdefghijklmnopqrstuvwxyz0123"),
				alicloud.CredentialsFile: []byte(`[default]
type = access_key
access_key_id = abcdefghijklmnop
access_key_secret = abcdefghijklmnopqrstuvwxyz0123`),
			}))
		})
		It("should add credentials file", func() {
			err := ensurer.EnsureCloudProviderSecret(ctx, nil, secret, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data).To(Equal(map[string][]byte{
				alicloud.AccessKeyID:     []byte("abcdefghijklmnop"),
				alicloud.AccessKeySecret: []byte("abcdefghijklmnopqrstuvwxyz0123"),
				alicloud.CredentialsFile: []byte(`[default]
type = access_key
access_key_id = abcdefghijklmnop
access_key_secret = abcdefghijklmnopqrstuvwxyz0123`),
			}))
		})

		DescribeTable("should reject invalid credentials before modifying the secret on create or update",
			func(key, value string) {
				for _, update := range []bool{false, true} {
					newSecret := secret.DeepCopy()
					var oldSecret *corev1.Secret
					if update {
						newSecret.Data[alicloud.CredentialsFile] = []byte("existing-credentials-file")
						oldSecret = newSecret.DeepCopy()
					}
					newSecret.Data[key] = []byte(value)
					before := newSecret.DeepCopy()

					err := ensurer.EnsureCloudProviderSecret(ctx, nil, newSecret, oldSecret)
					Expect(err).To(MatchError(ContainSubstring("field %q in secret shoot--test--alicloud/cloudprovider", key)))
					Expect(newSecret).To(Equal(before))
					if value != "" {
						Expect(err.Error()).NotTo(ContainSubstring(value))
					}
				}
			},
			Entry("empty access key ID", alicloud.AccessKeyID, ""),
			Entry("short access key ID", alicloud.AccessKeyID, strings.Repeat("a", 15)),
			Entry("long access key ID", alicloud.AccessKeyID, strings.Repeat("a", 129)),
			Entry("invalid access key ID alphabet", alicloud.AccessKeyID, strings.Repeat("a", 16)+"#comment"),
			Entry("LF in access key ID", alicloud.AccessKeyID, strings.Repeat("a", 16)+"\ntype = ecs_ram_role\nrole_name ="),
			Entry("CR in access key ID", alicloud.AccessKeyID, strings.Repeat("a", 16)+"\rtype = ecs_ram_role\rrole_name ="),
			Entry("CRLF in access key ID", alicloud.AccessKeyID, strings.Repeat("a", 16)+"\r\n[default]\r\ntype = ecs_ram_role"),
			Entry("empty secret access key", alicloud.AccessKeySecret, ""),
			Entry("short secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 29)),
			Entry("long secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 129)),
			Entry("LF in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 30)+"\ntype = ecs_ram_role\nrole_name ="),
			Entry("CR in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 30)+"\rtype = ecs_ram_role\rrole_name ="),
			Entry("CRLF in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 30)+"\r\ntype = ecs_ram_role\r\nrole_name ="),
			Entry("duplicate section in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 30)+"\n[default]\ntype = ecs_ram_role"),
			Entry("NUL in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 15)+"\x00"+strings.Repeat("b", 15)),
			Entry("tab in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 15)+"\t"+strings.Repeat("b", 15)),
			Entry("hash comment in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 30)+"#comment"),
			Entry("semicolon comment in secret access key", alicloud.AccessKeySecret, strings.Repeat("b", 30)+";comment"),
		)
	})
})
