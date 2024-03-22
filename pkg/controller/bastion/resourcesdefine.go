// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bastion

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func ingressAllowSSH(securityGroupId string, perm IngressPermission) *ecs.AuthorizeSecurityGroupRequest {
	request := ecs.CreateAuthorizeSecurityGroupRequest()
	request.SecurityGroupId = securityGroupId
	request.Description = "SSH access for Bastion"
	request.IpProtocol = "TCP"
	request.PortRange = sshPort + "/" + sshPort

	if perm.EtherType == ipv4Type {
		request.SourceCidrIp = perm.CIDR
	} else {
		request.Ipv6SourceCidrIp = perm.CIDR
	}

	return request
}

func egressAllowSSHToWorker(sourceCidrIp, securityGroupId, destSecurityGroupId string) *ecs.AuthorizeSecurityGroupEgressRequest {
	request := ecs.CreateAuthorizeSecurityGroupEgressRequest()
	request.SecurityGroupId = securityGroupId
	request.Description = "Allow Bastion egress to Shoot workers"
	request.IpProtocol = "TCP"
	request.SourceCidrIp = sourceCidrIp
	request.PortRange = sshPort + "/" + sshPort
	request.DestGroupId = destSecurityGroupId
	return request
}

func egressDenyAll(securityGroupId string) *ecs.AuthorizeSecurityGroupEgressRequest {
	request := ecs.CreateAuthorizeSecurityGroupEgressRequest()
	request.SecurityGroupId = securityGroupId
	request.Description = "Bastion egress deny"
	request.IpProtocol = "TCP"
	request.PortRange = "1/65535"
	request.Priority = "100"
	request.DestCidrIp = "0.0.0.0/0"
	return request
}

func revokeSecurityGroupRequest(securityGroupId, ipProtocol, portRange, sourceCidrIp, ipv6SourceCidrIp string) *ecs.RevokeSecurityGroupRequest {
	request := ecs.CreateRevokeSecurityGroupRequest()
	request.SecurityGroupId = securityGroupId
	request.IpProtocol = ipProtocol
	request.PortRange = portRange
	request.SourceCidrIp = sourceCidrIp
	request.Ipv6SourceCidrIp = ipv6SourceCidrIp
	return request
}

func revokeSecurityGroupEgressRequest(securityGroupId, ipProtocol, portRange string) *ecs.RevokeSecurityGroupEgressRequest {
	request := ecs.CreateRevokeSecurityGroupEgressRequest()
	request.SecurityGroupId = securityGroupId
	request.IpProtocol = ipProtocol
	request.PortRange = portRange
	return request
}

func describeSecurityGroupAttributeRequest(securityGroupId, direction string) *ecs.DescribeSecurityGroupAttributeRequest {
	request := ecs.CreateDescribeSecurityGroupAttributeRequest()
	request.SecurityGroupId = securityGroupId
	request.Direction = direction
	return request
}
