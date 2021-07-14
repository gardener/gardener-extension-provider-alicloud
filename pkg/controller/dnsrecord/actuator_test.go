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

package dnsrecord_test

import (
	"context"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/mock"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/dnsrecord"

	"github.com/gardener/gardener/extensions/pkg/controller/dnsrecord"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	name       = "alicloud-external"
	namespace  = "shoot--foobar--alicloud"
	domainName = "shoot.example.com"
	dnsName    = "api.alicloud.foobar." + domainName
	address    = "1.2.3.4"

	accessKeyID     = "accessKeyID"
	accessKeySecret = "accessKeySecret"
)

var _ = Describe("Actuator", func() {
	var (
		ctrl                  *gomock.Controller
		c                     *mockclient.MockClient
		sw                    *mockclient.MockStatusWriter
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		dnsClient             *mockalicloudclient.MockDNS
		ctx                   context.Context
		logger                logr.Logger
		a                     dnsrecord.Actuator
		dns                   *extensionsv1alpha1.DNSRecord
		secret                *corev1.Secret
		domainNames           []string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		c = mockclient.NewMockClient(ctrl)
		sw = mockclient.NewMockStatusWriter(ctrl)
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		dnsClient = mockalicloudclient.NewMockDNS(ctrl)

		c.EXPECT().Status().Return(sw).AnyTimes()

		ctx = context.TODO()
		logger = log.Log.WithName("test")

		a = NewActuator(alicloudClientFactory, logger)

		err := a.(inject.Client).InjectClient(c)
		Expect(err).NotTo(HaveOccurred())

		dns = &extensionsv1alpha1.DNSRecord{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: extensionsv1alpha1.DNSRecordSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type: alicloud.DNSType,
				},
				SecretRef: corev1.SecretReference{
					Name:      name,
					Namespace: namespace,
				},
				Name:       dnsName,
				RecordType: extensionsv1alpha1.DNSRecordTypeA,
				Values:     []string{address},
			},
		}
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(accessKeyID),
				alicloud.AccessKeySecret: []byte(accessKeySecret),
			},
		}

		domainNames = []string{
			domainName,
			"example.com",
			"other.com",
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Reconcile", func() {
		It("should reconcile the DNSRecord", func() {
			c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret) error {
					*obj = *secret
					return nil
				},
			)
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainNames(ctx).Return(domainNames, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, domainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, domainName, "comment-"+dnsName, "TXT").Return(nil)
			c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&extensionsv1alpha1.DNSRecord{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, obj *extensionsv1alpha1.DNSRecord) error {
					*obj = *dns
					return nil
				},
			)
			sw.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&extensionsv1alpha1.DNSRecord{})).DoAndReturn(
				func(_ context.Context, obj *extensionsv1alpha1.DNSRecord, opts ...client.UpdateOption) error {
					Expect(obj.Status).To(Equal(extensionsv1alpha1.DNSRecordStatus{
						Zone: pointer.String(domainName),
					}))
					return nil
				},
			)

			err := a.Reconcile(ctx, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("#Delete", func() {
		It("should delete the DNSRecord", func() {
			dns.Status.Zone = pointer.String(domainName)

			c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret) error {
					*obj = *secret
					return nil
				},
			)
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, domainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA)).Return(nil)

			err := a.Delete(ctx, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
