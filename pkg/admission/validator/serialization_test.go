// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"reflect"

	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	apisali "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	apisaliv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
)

func equalBackupBucketConfig(a, b *apisali.BackupBucketConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return reflect.DeepEqual(a.Immutability, b.Immutability)
}

var _ = Describe("Decode", func() {
	var (
		decoder runtime.Decoder
		scheme  *runtime.Scheme
	)
	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(core.AddToScheme(scheme)).To(Succeed())
		Expect(apisali.AddToScheme(scheme)).To(Succeed())
		Expect(apisaliv1alpha1.AddToScheme(scheme)).To(Succeed())

		decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()

	})
	DescribeTable("DecodeBackupBucketConfig",
		func(config *runtime.RawExtension, want *apisali.BackupBucketConfig, wantErr bool) {
			got, err := DecodeBackupBucketConfig(decoder, config)
			if wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(equalBackupBucketConfig(got, want)).To(BeTrue())
		},
		Entry("valid config", &runtime.RawExtension{Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig", "immutability": {"retentionType": "bucket", "retentionPeriod": 1, "locked": false }}`)},
			&apisali.BackupBucketConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "alicloud.provider.extensions.gardener.cloud/v1alpha1",
					Kind:       "BackupBucketConfig",
				},
				Immutability: &apisali.ImmutableConfig{
					RetentionType:   "bucket",
					RetentionPeriod: 1,
					Locked:          false,
				},
			}, false),
		Entry("invalid config", &runtime.RawExtension{Raw: []byte(`invalid`)}, nil, true),
		Entry("missing fields", &runtime.RawExtension{Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "BackupBucketConfig"}`)},
			&apisali.BackupBucketConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "alicloud.provider.extensions.gardener.cloud/v1alpha1",
					Kind:       "BackupBucketConfig",
				},
			}, false),
		Entry("different data in provider config", &runtime.RawExtension{Raw: []byte(`{"apiVersion": "alicloud.provider.extensions.gardener.cloud/v1alpha1", "kind": "DifferentConfig", "someField": "someValue"}`)},
			nil, true),
	)
})
