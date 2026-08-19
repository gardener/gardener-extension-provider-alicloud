// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
)

// byoIdentifiers tracks the pre-created "user-owned" resources used in BYO integration tests.
type byoIdentifiers struct {
	vpcID           string
	vswitchID       string
	securityGroupID string
}

// prepareBYOResources creates a VPC, VSwitch, and Security Group that simulate
// the pre-existing infrastructure a user provides in BYO mode.
// No NAT Gateway is created — the extension must not create one either.
func prepareBYOResources(ctx context.Context, clientFactory alicloudclient.ClientFactory, region, availabilityZone string) byoIdentifiers {
	vpcClient, err := clientFactory.NewVPCClient(region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())
	ecsClient, err := clientFactory.NewECSClient(region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	// 1. Create VPC
	createVpcReq := vpc.CreateCreateVpcRequest()
	createVpcReq.VpcName = "provider-alicloud-byo-test"
	createVpcReq.CidrBlock = vpcCIDR
	createVpcReq.RegionId = region
	createVpcResp, err := vpcClient.CreateVpc(createVpcReq)
	Expect(err).NotTo(HaveOccurred())
	vpcID := createVpcResp.VpcId

	describeVpcsReq := vpc.CreateDescribeVpcsRequest()
	describeVpcsReq.VpcId = vpcID
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		resp, err := vpcClient.DescribeVpcs(describeVpcsReq)
		if err != nil {
			return false, err
		}
		return resp.Vpcs.Vpc[0].Status == availableStatus, nil
	})
	Expect(err).NotTo(HaveOccurred())

	// 2. Create VSwitch
	createVSwitchReq := vpc.CreateCreateVSwitchRequest()
	createVSwitchReq.VpcId = vpcID
	createVSwitchReq.RegionId = region
	createVSwitchReq.CidrBlock = workersCIDR
	createVSwitchReq.ZoneId = availabilityZone
	createVSwitchResp, err := vpcClient.CreateVSwitch(createVSwitchReq)
	Expect(err).NotTo(HaveOccurred())
	vswitchID := createVSwitchResp.VSwitchId

	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VSwitchId = vswitchID
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		resp, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
		if err != nil {
			return false, err
		}
		return resp.VSwitches.VSwitch[0].Status == availableStatus, nil
	})
	Expect(err).NotTo(HaveOccurred())

	// 3. Create Security Group (no rules — BYO SG is managed by the user)
	createSGResp, err := ecsClient.CreateSecurityGroups(vpcID, "provider-alicloud-byo-sg")
	Expect(err).NotTo(HaveOccurred())

	return byoIdentifiers{
		vpcID:           vpcID,
		vswitchID:       vswitchID,
		securityGroupID: createSGResp.SecurityGroupId,
	}
}

// cleanupBYOResources deletes the pre-created BYO resources after the test completes.
// Must be called after verifyBYODeletion so the resources are still present at verification time.
func cleanupBYOResources(ctx context.Context, clientFactory alicloudclient.ClientFactory, ids byoIdentifiers) {
	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())
	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	// 1. Delete Security Group
	Expect(ecsClient.DeleteSecurityGroups(ids.securityGroupID)).To(Succeed())

	// 2. Delete VSwitch and wait
	deleteVSwitchReq := vpc.CreateDeleteVSwitchRequest()
	deleteVSwitchReq.VSwitchId = ids.vswitchID
	_, err = vpcClient.DeleteVSwitch(deleteVSwitchReq)
	Expect(err).NotTo(HaveOccurred())

	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VSwitchId = ids.vswitchID
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		resp, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
		if err != nil {
			return false, err
		}
		return len(resp.VSwitches.VSwitch) == 0, nil
	})
	Expect(err).NotTo(HaveOccurred())

	// 3. Delete VPC and wait
	deleteVpcReq := vpc.CreateDeleteVpcRequest()
	deleteVpcReq.VpcId = ids.vpcID
	_, err = vpcClient.DeleteVpc(deleteVpcReq)
	Expect(err).NotTo(HaveOccurred())

	describeVpcsReq := vpc.CreateDescribeVpcsRequest()
	describeVpcsReq.VpcId = ids.vpcID
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, false, func(_ context.Context) (bool, error) {
		resp, err := vpcClient.DescribeVpcs(describeVpcsReq)
		if err != nil {
			return false, err
		}
		return len(resp.Vpcs.Vpc) == 0, nil
	})
	Expect(err).NotTo(HaveOccurred())
}

// newBYOProviderConfig builds an InfrastructureConfig for BYO mode: user-provided VPC,
// VSwitch, and Security Group; no Workers CIDR and no NAT Gateway config.
func newBYOProviderConfig(ids byoIdentifiers, availabilityZone string) *alicloudv1alpha1.InfrastructureConfig {
	return &alicloudv1alpha1.InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: alicloudv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureConfig",
		},
		Networks: alicloudv1alpha1.Networks{
			VPC: alicloudv1alpha1.VPC{
				ID: ptr.To(ids.vpcID),
			},
			NodesSecurityGroupID: ptr.To(ids.securityGroupID),
			Zones: []alicloudv1alpha1.Zone{
				{
					Name:             availabilityZone,
					WorkersVSwitchID: ptr.To(ids.vswitchID),
				},
			},
		},
	}
}

// verifyBYOCreation checks that:
//   - InfrastructureStatus reflects the user-provided VPC, VSwitch, and SG IDs
//   - The extension did NOT create a NAT Gateway in the user's VPC
func verifyBYOCreation(
	clientFactory alicloudclient.ClientFactory,
	_ *extensionsv1alpha1.Infrastructure,
	infraStatus *alicloudv1alpha1.InfrastructureStatus,
	ids byoIdentifiers,
) {
	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	Expect(infraStatus.VPC.ID).To(Equal(ids.vpcID))
	Expect(infraStatus.VPC.VSwitches).To(ContainElement(HaveField("ID", ids.vswitchID)))
	Expect(infraStatus.VPC.SecurityGroups).To(ContainElement(HaveField("ID", ids.securityGroupID)))

	// Extension must not have created a NAT Gateway in the BYO VPC.
	describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
	describeNATGatewaysReq.VpcId = ids.vpcID
	describeNatGatewaysOutput, err := vpcClient.DescribeNatGateways(describeNATGatewaysReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeNatGatewaysOutput.NatGateways.NatGateway).To(BeEmpty())
}

// verifyBYODeletion checks that the user-provided VSwitch, Security Group, and VPC
// still exist after the shoot Infrastructure has been deleted.
func verifyBYODeletion(clientFactory alicloudclient.ClientFactory, ids byoIdentifiers) {
	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())
	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	// VSwitch must still exist
	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VSwitchId = ids.vswitchID
	describeVSwitchesOutput, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeVSwitchesOutput.VSwitches.VSwitch).To(HaveLen(1))
	Expect(describeVSwitchesOutput.VSwitches.VSwitch[0].VSwitchId).To(Equal(ids.vswitchID))

	// Security Group must still exist
	describeSecurityGroupsReq := ecs.CreateDescribeSecurityGroupsRequest()
	describeSecurityGroupsReq.SecurityGroupId = ids.securityGroupID
	describeSecurityGroupOutput, err := ecsClient.DescribeSecurityGroups(describeSecurityGroupsReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeSecurityGroupOutput.SecurityGroups.SecurityGroup).To(HaveLen(1))

	// VPC must still exist
	describeVPCsReq := vpc.CreateDescribeVpcsRequest()
	describeVPCsReq.VpcId = ids.vpcID
	describeVpcsOutput, err := vpcClient.DescribeVpcs(describeVPCsReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeVpcsOutput.Vpcs.Vpc).To(HaveLen(1))
}

// runBYOTest runs the full BYO infrastructure integration test via runTestBase,
// substituting BYO-specific creation and deletion verification.
func runBYOTest(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	providerConfig *alicloudv1alpha1.InfrastructureConfig,
	decoder runtime.Decoder,
	clientFactory alicloudclient.ClientFactory,
	ids byoIdentifiers,
) error {
	return runTestBase(ctx, logger, c, providerConfig, decoder, clientFactory,
		func(cf alicloudclient.ClientFactory, infra *extensionsv1alpha1.Infrastructure, status *alicloudv1alpha1.InfrastructureStatus) error {
			verifyBYOCreation(cf, infra, status, ids)
			return nil
		},
		func(cf alicloudclient.ClientFactory) {
			verifyBYODeletion(cf, ids)
		},
	)
}
