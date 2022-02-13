// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
