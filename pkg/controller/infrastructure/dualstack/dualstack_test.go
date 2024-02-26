package dualstack_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/dualstack"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
)

func TestDualStack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DualStack Suite")
}

var _ = Describe("DualStackValues", func() {
	var (
		ctrl *gomock.Controller

		clientFactory   *mockalicloudclient.MockClientFactory
		nlbClient       *mockalicloudclient.MockNLB
		region          string
		accessKeyID     string
		accessKeySecret string
		cidr            string
		credentials     *alicloud.Credentials
		zone_list       []string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		cidr = "192.168.0.0/16"
		accessKeyID = "accessKeyID"
		accessKeySecret = "accessKeySecret"
		credentials = &alicloud.Credentials{
			AccessKeyID:     accessKeyID,
			AccessKeySecret: accessKeySecret,
		}
		region = "region"
		zone_list = []string{
			"zone_a",
			"zone_b",
			"zone_c",
		}
		clientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		nlbClient = mockalicloudclient.NewMockNLB(ctrl)

	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should compute the dualstack values when enable dualStack", func() {
		gomock.InOrder(
			clientFactory.EXPECT().NewNLBClient(region, accessKeyID, accessKeySecret).Return(nlbClient, nil),
			nlbClient.EXPECT().GetNLBAvailableZones(region).Return(zone_list, nil),
		)
		cidr = "192.168.0.0/16"
		dualStackvalue, err := dualstack.CreateDualStackValues(true, region, &cidr, credentials, clientFactory)
		Expect(err).NotTo(HaveOccurred())
		Expect(dualStackvalue).To(Equal(&dualstack.DualStack{
			Enabled:            true,
			Zone_A:             zone_list[0],
			Zone_A_CIDR:        "192.168.255.240/28",
			Zone_A_IPV6_SUBNET: 255,
			Zone_B:             zone_list[1],
			Zone_B_IPV6_SUBNET: 254,
			Zone_B_CIDR:        "192.168.255.224/28",
		}))

	})

	It("should compute the dualstack values when not enable dualStack", func() {
		dualStackvalue, err := dualstack.CreateDualStackValues(false, region, &cidr, credentials, clientFactory)
		Expect(err).NotTo(HaveOccurred())
		Expect(dualStackvalue.Enabled).To(Equal(false))
	})

	It("should fail to compute the dualstack when not enough avalable zones", func() {
		zone_list = []string{
			"zone_a",
		}
		gomock.InOrder(
			clientFactory.EXPECT().NewNLBClient(region, accessKeyID, accessKeySecret).Return(nlbClient, nil),
			nlbClient.EXPECT().GetNLBAvailableZones(region).Return(zone_list, nil),
		)
		_, err := dualstack.CreateDualStackValues(true, region, &cidr, credentials, clientFactory)
		Expect(err).To(HaveOccurred())

	})

	It("should fail to compute the dualstack values when vpc cidr not be set)", func() {
		_, err := dualstack.CreateDualStackValues(true, region, nil, credentials, clientFactory)
		Expect(err).To(HaveOccurred())
	})

	It("should fail to compute the dualstack when vpc cidr not big enough", func() {
		cidr = "192.168.0.0/28"
		gomock.InOrder(
			clientFactory.EXPECT().NewNLBClient(region, accessKeyID, accessKeySecret).Return(nlbClient, nil),
			nlbClient.EXPECT().GetNLBAvailableZones(region).Return(zone_list, nil),
		)
		_, err := dualstack.CreateDualStackValues(true, region, &cidr, credentials, clientFactory)
		Expect(err).To(HaveOccurred())

	})

	It("should fail to compute the dualstack when getNLB client fail", func() {
		gomock.InOrder(
			clientFactory.EXPECT().NewNLBClient(region, accessKeyID, accessKeySecret).Return(nil, fmt.Errorf("some err")),
		)
		_, err := dualstack.CreateDualStackValues(true, region, &cidr, credentials, clientFactory)
		Expect(err).To(HaveOccurred())

	})

	It("should fail to compute the dualstack when list available zones fail", func() {
		gomock.InOrder(
			clientFactory.EXPECT().NewNLBClient(region, accessKeyID, accessKeySecret).Return(nlbClient, nil),
			nlbClient.EXPECT().GetNLBAvailableZones(region).Return(nil, fmt.Errorf("some err")),
		)
		_, err := dualstack.CreateDualStackValues(true, region, &cidr, credentials, clientFactory)
		Expect(err).To(HaveOccurred())

	})
})
