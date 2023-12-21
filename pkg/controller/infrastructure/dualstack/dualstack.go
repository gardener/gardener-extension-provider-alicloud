package dualstack

import (
	"fmt"
	"net"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
)

// DualStack contains values of EIP used to render terraform charts
type DualStack struct {
	Enabled          bool
	Zone_A           string
	Zone_A_CIDR      string
	Zone_A_IPV6_MASK int
	Zone_B           string
	Zone_B_CIDR      string
	Zone_B_IPV6_MASK int
}

// CreateDualStackValues create DualStack values
func CreateDualStackValues(
	enableDualStack bool,
	region string,
	vpcCidr *string,
	credentials *alicloud.Credentials,
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
	dualStack.Zone_A_IPV6_MASK = 255
	dualStack.Zone_B_IPV6_MASK = 254
	nlbClient, err := alicloudclient.NewClientFactory().NewNLBClient(region, credentials.AccessKeyID, credentials.AccessKeySecret)
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

	subCidrs, err := getLastSubCidr(*vpcCidr, 28, 2)
	if err != nil || subCidrs == nil || len(subCidrs) < 2 {
		return nil, fmt.Errorf("get sub cidr failed")
	}

	dualStack.Zone_A_CIDR = subCidrs[0]
	dualStack.Zone_B_CIDR = subCidrs[1]

	return &dualStack, nil
}

func getLastSubCidr(originCidr string, subNetMaskLen, count int) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(originCidr)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 1
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
		subCidrs = append(subCidrs, fmt.Sprintf("%s/%d", ipInc(ipnet.IP, (subnets-index)*ip_size), subNetMaskLen))
	}
	return subCidrs, nil

}

func ipInc(ip net.IP, step int) net.IP {
	res := make(net.IP, len(ip))
	copy(res, ip)
	for j := len(res) - 1; j >= 0; j-- {
		if step > 255 {
			res[j] += byte(step % 256)
			step /= 256
		} else {
			res[j] += byte(step)
			step = 0
		}
		if step == 0 {
			break
		}
	}
	return res
}
