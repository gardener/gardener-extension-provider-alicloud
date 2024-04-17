package dualstack

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

// DualStack contains values of EIP used to render terraform charts
// Alibaba requires users to provide two vswitch for the deployment of NLB when creating a service that supports IPV6.
// Here we parpare two new vswitch for user to deploy of Alibaba's NLB to enable dual stack. We named them A and B.
// Therefore, these two vswitch need to support IPV6, and obviously, an IPV4 CIDR is also needed.
// Zone_A Zone_B is zhe zone name.
// Zone_A_CIDR Zone_B_CIDR is IPV4 CIDR.
// Zone_A_IPV6_SUBNET Zone_B_IPV6_SUBNET is IPV6 subnet identifier.
type DualStack struct {
	Enabled            bool
	Zone_A             string
	Zone_A_CIDR        string
	Zone_A_IPV6_SUBNET int
	Zone_B             string
	Zone_B_CIDR        string
	Zone_B_IPV6_SUBNET int
}

// CreateDualStackValues create DualStack values
func CreateDualStackValues(
	enableDualStack bool,
	region string,
	vpcCidr *string,
	credentials *alicloud.Credentials,
	clientFactory alicloudclient.ClientFactory,
) (*DualStack, error) {
	dualStack := DualStack{
		Enabled: enableDualStack,
	}
	if !enableDualStack {
		return &dualStack, nil
	}
	if vpcCidr == nil {
		return nil, fmt.Errorf("vpcCidr must be set")
	}
	// we use last two subnet to distinguish from user's subnet
	dualStack.Zone_A_IPV6_SUBNET = 255
	dualStack.Zone_B_IPV6_SUBNET = 254
	nlbClient, err := clientFactory.NewNLBClient(region, credentials.AccessKeyID, credentials.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	zones, err := nlbClient.GetNLBAvailableZones(region)
	if err != nil {
		return nil, err
	}
	if len(zones) < 2 {
		return nil, fmt.Errorf("not enough available zones for DualStack")
	}
	dualStack.Zone_A = zones[0]
	dualStack.Zone_B = zones[1]

	// DualStack only for managed vpc

	subCidrs, err := getLastIpv4SubCidr(*vpcCidr, 28, 2)
	if err != nil || subCidrs == nil || len(subCidrs) < 2 {
		return nil, fmt.Errorf("get sub cidr failed")
	}

	dualStack.Zone_A_CIDR = subCidrs[0]
	dualStack.Zone_B_CIDR = subCidrs[1]

	return &dualStack, nil
}

func getLastIpv4SubCidr(originCidr string, subNetMaskLen, count int) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(originCidr)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		return nil, fmt.Errorf("count must greater than 0")
	}

	orgin_net_mask_len, _ := ipnet.Mask.Size()
	if orgin_net_mask_len > subNetMaskLen {
		return nil, fmt.Errorf("not enough capacity to divide sub cidr")
	}
	sub_ip_mask_len := 32 - subNetMaskLen

	subnets := 1 << uint(32-orgin_net_mask_len-sub_ip_mask_len)
	if count > subnets {
		return nil, fmt.Errorf("not enough subnets")
	}
	subCidrs := make([]string, 0, count)
	ip_size := 1 << uint(sub_ip_mask_len)
	for index := 1; index <= count; index++ {
		subCidrs = append(subCidrs, fmt.Sprintf("%s/%d", ipv4Inc(ipnet.IP, uint32((subnets-index)*ip_size)), subNetMaskLen))
	}
	return subCidrs, nil

}

func ipv4Inc(ip net.IP, step uint32) net.IP {
	ipInt := binary.BigEndian.Uint32(ip.To4())
	ipInt += step
	newIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(newIP, ipInt)
	return newIP
}
