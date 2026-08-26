// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package dnsrecord_test

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/dnsrecord"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	testutils "github.com/gardener/gardener/pkg/utils/test"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	credentialsFile = "credentialsFile"
)

var _ = Describe("Actuator", func() {
	var (
		ctrl                  *gomock.Controller
		c                     client.Client
		mgr                   *testutils.FakeManager
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

		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		dnsClient = mockalicloudclient.NewMockDNS(ctrl)

		ctx = context.TODO()
		logger = log.Log.WithName("test")

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
				alicloud.CredentialsFile: []byte(credentialsFile),
			},
		}

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		c = fakeclient.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret, dns).
			WithStatusSubresource(&extensionsv1alpha1.DNSRecord{}).
			Build()
		mgr = &testutils.FakeManager{Client: c}
		a = NewActuator(mgr, alicloudClientFactory)

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
		expectUpdateDNSRecordStatus = func(zone string) {
			updated := &extensionsv1alpha1.DNSRecord{}
			Expect(c.Get(ctx, client.ObjectKeyFromObject(dns), updated)).To(Succeed())
			Expect(updated.Status.Zone).To(Equal(ptr.To(zone)))
		}
	)

	Describe("#Reconcile", func() {
		It("should reconcile the DNSRecord if a zone is not specified", func() {
			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainNames(ctx).Return(domainNames, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, "comment-"+dnsName, "TXT").Return(nil)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
			expectUpdateDNSRecordStatus(compositeDomainName)
		})

		It("should reconcile the DNSRecord if a zone is specified and it's a domain name", func() {
			dns.Spec.Zone = ptr.To(domainName)

			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, domainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, domainName, "comment-"+dnsName, "TXT").Return(nil)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
			expectUpdateDNSRecordStatus(domainName)
		})

		It("should reconcile the DNSRecord if a zone is specified and it's a domain id", func() {
			dns.Spec.Zone = ptr.To(domainId)

			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainName(ctx, domainId).Return(compositeDomainName, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, "comment-"+dnsName, "TXT").Return(nil)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
			expectUpdateDNSRecordStatus(compositeDomainName)
		})

		It("should reconcile the DNSRecord if a zone is specified and it's different from the status zone", func() {
			dns.Spec.Zone = ptr.To(domainId)
			dns.Status.Zone = ptr.To("example.com:2")

			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().GetDomainName(ctx, domainId).Return(compositeDomainName, nil)
			dnsClient.EXPECT().CreateOrUpdateDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA), []string{address}, int64(120)).Return(nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, "comment-"+dnsName, "TXT").Return(nil)

			err := a.Reconcile(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
			expectUpdateDNSRecordStatus(compositeDomainName)
		})
	})

	Describe("#Delete", func() {
		It("should delete the DNSRecord with a composite domain name in status", func() {
			dns.Status.Zone = ptr.To(compositeDomainName)

			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, compositeDomainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA)).Return(nil)

			err := a.Delete(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete the DNSRecord with a domain name in status", func() {
			dns.Status.Zone = ptr.To(domainName)

			alicloudClientFactory.EXPECT().NewDNSClient(alicloud.DefaultDNSRegion, accessKeyID, accessKeySecret).Return(dnsClient, nil)
			dnsClient.EXPECT().DeleteDomainRecords(ctx, domainName, dnsName, string(extensionsv1alpha1.DNSRecordTypeA)).Return(nil)

			err := a.Delete(ctx, logger, dns, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
