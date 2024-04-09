// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package dnsrecord_test

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/dnsrecord"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client/mock"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/dnsrecord"
)

const (
	name                = "alicloud-external"
	namespace           = "shoot--foobar--alicloud"
	domainName          = "shoot.example.com"
	domainId            = "1"
	compositeDomainName = domainName + ":" + domainId
	dnsName             = "api.alicloud.foobar." + domainName
	address             = "1.2.3.4"

	accessKeyID     = "accessKeyID"
	accessKeySecret = "accessKeySecret"
)

var _ = Describe("Actuator", func() {
	var (
		ctrl                  *gomock.Controller
		c                     *mockclient.MockClient
		mgr                   *mockmanager.MockManager
		sw                    *mockclient.MockStatusWriter
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		dnsClient             *mockalicloudclient.MockDNS
		ctx                   context.Context
		logger                logr.Logger
		a                     dnsrecord.Actuator
		dns                   *extensionsv1alpha1.DNSRecord
		secret                *corev1.Secret
		domainNames           map[string]string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		c = mockclient.NewMockClient(ctrl)
		mgr = mockmanager.NewMockManager(ctrl)

		mgr.EXPECT().GetClient().Return(c)

		sw = mockclient.NewMockStatusWriter(ctrl)
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		dnsClient = mockalicloudclient.NewMockDNS(ctrl)

		c.EXPECT().Status().Return(sw).AnyTimes()

		ctx = context.TODO()
		logger = log.Log.WithName("test")

		a = NewActuator(mgr, alicloudClientFactory)

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

		domainNames = map[string]string{
			domainName:    compositeDomainName,
			"example.com": "example.com:2",
			"other.com":   "other.com:3",
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	var (
		expectGetDNSRecordSecret = func() {
			c.EXPECT().Get(ctx, kutil.Key(namespace, name), gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
				func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					*obj = *secret
					return nil
				},
			)
		}
		expectUpdateDNSRecordStatus = func(zone string) {
			sw.EXPECT().Patch(ctx, gomock.AssignableToTypeOf(&extensionsv1alpha1.DNSRecord{}), gomock.Any()).DoAndReturn(
				func(_ context.Context, obj *extensionsv1alpha1.DNSRecord, _ client.Patch, _ ...client.PatchOption) error {
					Expect(obj.Status).To(Equal(extensionsv1alpha1.DNSRecordStatus{
						Zone: ptr.To(zone),
					}))
					return nil
				},
			)
		}
	)

	Describe("#Reconcile", func() {
		It("should reconcile the DNSRecord if a zone is not specified", func() {
			expectGetDNSRecordSecret()
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainNames(ctx).Return(domainNames, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, "comment-"+dnsName, "TXT").Return(nil)
			expectUpdateDNSRecordStatus(compositeDomainName)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reconcile the DNSRecord if a zone is specified and it's a domain name", func() {
			dns.Spec.Zone = ptr.To(domainName)

			expectGetDNSRecordSecret()
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, domainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, domainName, "comment-"+dnsName, "TXT").Return(nil)
			expectUpdateDNSRecordStatus(domainName)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reconcile the DNSRecord if a zone is specified and it's a domain id", func() {
			dns.Spec.Zone = ptr.To(domainId)

			expectGetDNSRecordSecret()
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainName(ctx, domainId).Return(compositeDomainName, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, "comment-"+dnsName, "TXT").Return(nil)
			expectUpdateDNSRecordStatus(compositeDomainName)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reconcile the DNSRecord if a zone is specified and it's different from the status zone", func() {
			dns.Spec.Zone = ptr.To(domainId)
			dns.Status.Zone = ptr.To("example.com:2")

			expectGetDNSRecordSecret()
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainName(ctx, domainId).Return(compositeDomainName, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, "comment-"+dnsName, "TXT").Return(nil)
			expectUpdateDNSRecordStatus(compositeDomainName)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("#Delete", func() {
		It("should delete the DNSRecord with a composite domain name in status", func() {
			dns.Status.Zone = ptr.To(compositeDomainName)

			expectGetDNSRecordSecret()
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA)).Return(nil)

			err := a.Delete(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete the DNSRecord with a domain name in status", func() {
			dns.Status.Zone = ptr.To(domainName)

			expectGetDNSRecordSecret()
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, domainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA)).Return(nil)

			err := a.Delete(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
