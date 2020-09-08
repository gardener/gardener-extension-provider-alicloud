// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package infrastructure_test

import (
	"context"
	"encoding/json"
	"flag"
	"path/filepath"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/matchers"
	alicloudinstall "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	allCIDR     = "0.0.0.0/0"
	vpcCIDR     = "10.250.0.0/16"
	workersCIDR = "10.250.0.0/21"

	secretName = "cloudprovider"

	sshPublicKey       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDSb1DJfupnWTfKJ0fmRGgnSx8A2/pRd5oC49qE1jFX+/J9L01jUyLc5sBKZXVkfU5q5h0JfbkhJXSIkzqE+rNPnJBI4e+8Lo2TVWLAvVZRA9Fg9Dk3mgkdVB+9qW2mIqtJF5GOWKuk7HkObwpY1pX8kHC/LJVfpNQpVBqWef0WJj6vbyjhlZ3vgRxK9I6wdJzjUYtNsDhvvBTy/IBg/xp82w9T2r3GVfnaTLMeQCW9mPviDKnQsrWMgVb2A0Z4c62EbzzLzQV4ScVJ6JMgOgkMqEPdbnKF8dEQcSu+/DQZoZt56Aeov7T4oamahj9/rIDX+WR1nOcfntIdhCyoB4lISkNFz/MlPC7O8HwJk4P7rojLGNk6xmn6NxY5CJGC2dVxFsb1bmm+fKHAp62mgwEoFZcDyIkcsmnmnID9u0rJNyMz84YUGZ/jEz8LePujDHcXiqgoLsKJ8gNRneISL9+m9s1VK7WxDDIbq8iWzR7XfAVE/GzKpVYkqrWCvjKEeFIDuDUnf3jghQCQMsXnJM7zGWr1tl+Dvl2Avxmj2xyUJXYHbXbl2aM434DgQySnV8JPzYH7EsTmvuhdb8SJIbb/NonFsSM+72HpSzVc083x4B++VL7oP1X8cly62pFVM1fi8sxBio48Hq5SmAUu9T4wUY4J+AKU6osFA/ATlMCIiQ== your_email@example.com"
	sshPublicKeyDigest = "b9b39384513d9300374c98ccc8818a8b"
)

var (
	accessKeyID     = flag.String("access-key-id", "", "Alicloud access key id")
	accessKeySecret = flag.String("access-key-secret", "", "Alicloud access key secret")
	region          = flag.String("region", "", "Alicloud region")
	vpcID           = flag.String("vpc-id", "", "Alicloud existing VPC id")
)

func validateFlags() {
	if len(*accessKeyID) == 0 {
		panic("need an Alicloud access key id")
	}
	if len(*accessKeySecret) == 0 {
		panic("need an Alicloud access key secret")
	}
	if len(*region) == 0 {
		panic("need an Alicloud region")
	}
	if len(*vpcID) == 0 {
		panic("need an Alicloud existing VPC id")
	}
}

type aliClient struct {
	ECS *ecs.Client
	VPC *vpc.Client
}

func newAliClient(region, accessKeyID, accessKeySecret string) *aliClient {
	ecsClient, err := ecs.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		panic(err)
	}

	vpcClient, err := vpc.NewClientWithAccessKey(region, accessKeyID, accessKeySecret)
	if err != nil {
		panic(err)
	}

	return &aliClient{
		ECS: ecsClient,
		VPC: vpcClient,
	}
}

var _ = Describe("Infrastructure tests", func() {
	var (
		ctx    = context.Background()
		logger *logrus.Entry

		testEnv   *envtest.Environment
		mgrCancel context.CancelFunc
		c         client.Client
		decoder   runtime.Decoder

		InfraChartPath string

		alicloudClient *aliClient

		availabilityZone string
	)

	BeforeSuite(func() {
		InfraChartPath = alicloud.InfraChartPath
		repoRoot := filepath.Join("..", "..", "..")
		alicloud.InfraChartPath = filepath.Join(repoRoot, alicloud.InfraChartPath)

		logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
		log := logrus.New()
		log.SetOutput(GinkgoWriter)
		logger = logrus.NewEntry(log)

		By("starting test environment")
		testEnv = &envtest.Environment{
			UseExistingCluster: pointer.BoolPtr(true),
			CRDInstallOptions: envtest.CRDInstallOptions{
				Paths: []string{
					filepath.Join(repoRoot, "example", "20-crd-cluster.yaml"),
					filepath.Join(repoRoot, "example", "20-crd-infrastructure.yaml"),
				},
			},
		}

		cfg, err := testEnv.Start()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())

		By("setup manager")
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).ToNot(HaveOccurred())

		Expect(extensionsv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
		Expect(alicloudinstall.AddToScheme(mgr.GetScheme())).To(Succeed())

		Expect(infrastructure.AddToManager(mgr)).To(Succeed())

		var mgrContext context.Context
		mgrContext, mgrCancel = context.WithCancel(ctx)

		By("start manager")
		go func() {
			err := mgr.Start(mgrContext.Done())
			Expect(err).NotTo(HaveOccurred())
		}()

		c = mgr.GetClient()
		Expect(c).ToNot(BeNil())
		decoder = serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder()

		flag.Parse()
		validateFlags()

		alicloudClient = newAliClient(*region, *accessKeyID, *accessKeySecret)
		availabilityZone = *region + "a"
	})

	AfterSuite(func() {
		defer func() {
			By("stopping manager")
			mgrCancel()
		}()

		By("stopping test environment")
		Expect(testEnv.Stop()).To(Succeed())

		alicloud.InfraChartPath = InfraChartPath
	})

	Context("with infrastructure that requests new vpc (networks.vpc.cidr)", func() {
		AfterEach(func() {
			framework.RunCleanupActions()
		})

		It("should successfully create and delete", func() {
			providerConfig := newProviderConfig(&alicloudv1alpha1.VPC{
				CIDR: pointer.StringPtr(vpcCIDR),
			}, availabilityZone)

			err := runTest(ctx, logger, c, providerConfig, decoder, alicloudClient)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with infrastructure that requests existing vpc", func() {
		AfterEach(func() {
			framework.RunCleanupActions()
		})

		It("should successfully create and delete", func() {
			providerConfig := newProviderConfig(&alicloudv1alpha1.VPC{
				ID: vpcID,
			}, availabilityZone)

			err := runTest(ctx, logger, c, providerConfig, decoder, alicloudClient)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func runTest(ctx context.Context, logger *logrus.Entry, c client.Client, providerConfig *alicloudv1alpha1.InfrastructureConfig, decoder runtime.Decoder, alicloudClient *aliClient) error {
	var (
		infra                     *extensionsv1alpha1.Infrastructure
		namespace                 *corev1.Namespace
		cluster                   *extensionsv1alpha1.Cluster
		infrastructureIdentifiers infrastructureIdentifiers
	)

	var cleanupHandle framework.CleanupActionHandle
	cleanupHandle = framework.AddCleanupAction(func() {
		By("delete infrastructure")
		Expect(client.IgnoreNotFound(c.Delete(ctx, infra))).To(Succeed())

		By("wait until infrastructure is deleted")
		err := common.WaitUntilExtensionCRDeleted(
			ctx, c, logger,
			func() extensionsv1alpha1.Object { return &extensionsv1alpha1.Infrastructure{} },
			"Infrastructure", infra.Namespace, infra.Name,
			10*time.Second, 16*time.Minute,
		)
		Expect(err).NotTo(HaveOccurred())

		By("verify infrastructure deletion")
		verifyDeletion(ctx, alicloudClient, infrastructureIdentifiers)

		Expect(client.IgnoreNotFound(c.Delete(ctx, namespace))).To(Succeed())
		Expect(client.IgnoreNotFound(c.Delete(ctx, cluster))).To(Succeed())

		framework.RemoveCleanupAction(cleanupHandle)
	})

	By("create namespace for test execution")
	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "provider-alicloud-test-",
		},
	}
	if err := c.Create(ctx, namespace); err != nil {
		return err
	}

	By("create cluster")
	cluster = &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace.Name,
		},
		Spec: extensionsv1alpha1.ClusterSpec{},
	}
	if err := c.Create(ctx, cluster); err != nil {
		return err
	}

	By("deploy cloudprovider secret into namespace")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace.Name,
		},
		Data: map[string][]byte{
			alicloud.AccessKeyID:     []byte(*accessKeyID),
			alicloud.AccessKeySecret: []byte(*accessKeySecret),
		},
	}
	if err := c.Create(ctx, secret); err != nil {
		return err
	}

	By("create infrastructure")
	infra, err := newInfrastructure(namespace.Name, providerConfig)
	if err != nil {
		return err
	}

	if err := c.Create(ctx, infra); err != nil {
		return err
	}

	By("wait until infrastructure is created")
	if err := common.WaitUntilExtensionCRReady(
		ctx, c, logger,
		func() runtime.Object { return &extensionsv1alpha1.Infrastructure{} },
		"Infrastucture", infra.Namespace, infra.Name,
		10*time.Second, 30*time.Second, 16*time.Minute, nil,
	); err != nil {
		return err
	}

	By("decode infrastucture status")
	if err := c.Get(ctx, client.ObjectKey{Namespace: infra.Namespace, Name: infra.Name}, infra); err != nil {
		return err
	}

	providerStatus := &alicloudv1alpha1.InfrastructureStatus{}
	if _, _, err := decoder.Decode(infra.Status.ProviderStatus.Raw, nil, providerStatus); err != nil {
		return err
	}

	By("verify infrastructure creation")
	infrastructureIdentifiers = verifyCreation(ctx, alicloudClient, infra, providerStatus, providerConfig)

	return nil
}

func newProviderConfig(vpc *alicloudv1alpha1.VPC, availabilityZone string) *alicloudv1alpha1.InfrastructureConfig {
	return &alicloudv1alpha1.InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: alicloudv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureConfig",
		},
		Networks: alicloudv1alpha1.Networks{
			VPC: *vpc,
			Zones: []alicloudv1alpha1.Zone{
				{
					Name:    availabilityZone,
					Workers: workersCIDR,
				},
			},
		},
	}
}

func newInfrastructure(namespace string, providerConfig *alicloudv1alpha1.InfrastructureConfig) (*extensionsv1alpha1.Infrastructure, error) {
	providerConfigJSON, err := json.Marshal(&providerConfig)
	if err != nil {
		return nil, err
	}

	return &extensionsv1alpha1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "infrastructure",
			Namespace: namespace,
		},
		Spec: extensionsv1alpha1.InfrastructureSpec{
			DefaultSpec: extensionsv1alpha1.DefaultSpec{
				Type: alicloud.Type,
				ProviderConfig: &runtime.RawExtension{
					Raw: providerConfigJSON,
				},
			},
			SecretRef: corev1.SecretReference{
				Name:      secretName,
				Namespace: namespace,
			},
			Region:       *region,
			SSHPublicKey: []byte(sshPublicKey),
		},
	}, nil
}

type infrastructureIdentifiers struct {
	vpcID                 *string
	vswitchID             *string
	natGatewayID          *string
	securityGroupIDs      []string
	keyPairName           *string
	elasticIPAllocationID *string
	snatTableId           *string
	snatEntryId           *string
}

func verifyCreation(
	ctx context.Context,
	alicloudClient *aliClient,
	infra *extensionsv1alpha1.Infrastructure,
	infraStatus *alicloudv1alpha1.InfrastructureStatus,
	providerConfig *alicloudv1alpha1.InfrastructureConfig,
) (
	infrastructureIdentifier infrastructureIdentifiers,
) {
	const (
		sshKeySuffix        = "-ssh-publickey"
		eipSuffix           = "-eip-natgw-z0"
		securityGroupSuffix = "-sg"
	)

	vpcClient := alicloudClient.VPC
	ecsClient := alicloudClient.ECS

	// vpc
	describeVPCsReq := vpc.CreateDescribeVpcsRequest()
	describeVPCsReq.VpcId = infraStatus.VPC.ID
	describeVpcsOutput, err := vpcClient.DescribeVpcs(describeVPCsReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeVpcsOutput.Vpcs.Vpc).To(HaveLen(1))
	Expect(describeVpcsOutput.Vpcs.Vpc[0].VpcId).To(Equal(infraStatus.VPC.ID))
	Expect(describeVpcsOutput.Vpcs.Vpc[0].CidrBlock).To(Equal(vpcCIDR))
	if providerConfig.Networks.VPC.CIDR != nil {
		infrastructureIdentifier.vpcID = pointer.StringPtr(describeVpcsOutput.Vpcs.Vpc[0].VpcId)
	}

	// vswitch
	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VpcId = infraStatus.VPC.ID
	describeVSwitchesOutput, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeVSwitchesOutput.VSwitches.VSwitch[0].CidrBlock).To(Equal(workersCIDR))
	Expect(describeVSwitchesOutput.VSwitches.VSwitch[0].ZoneId).To(Equal(providerConfig.Networks.Zones[0].Name))
	infrastructureIdentifier.vswitchID = pointer.StringPtr(describeVSwitchesOutput.VSwitches.VSwitch[0].VSwitchId)
	if providerConfig.Networks.VPC.CIDR != nil {
		Expect(describeVSwitchesOutput.VSwitches.VSwitch).To(HaveLen(1))
	}

	// nat gateway
	describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
	describeNATGatewaysReq.VpcId = infraStatus.VPC.ID
	describeNatGatewaysOutput, err := vpcClient.DescribeNatGateways(describeNATGatewaysReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeNatGatewaysOutput.NatGateways.NatGateway).To(HaveLen(1))
	Expect(describeNatGatewaysOutput.NatGateways.NatGateway[0].SnatTableIds.SnatTableId).To(HaveLen(1))
	if providerConfig.Networks.VPC.CIDR != nil {
		infrastructureIdentifier.natGatewayID = pointer.StringPtr(describeNatGatewaysOutput.NatGateways.NatGateway[0].NatGatewayId)
	}

	// snat entries
	describeSnatTableEntriesReq := vpc.CreateDescribeSnatTableEntriesRequest()
	describeSnatTableEntriesReq.SnatTableId = describeNatGatewaysOutput.NatGateways.NatGateway[0].SnatTableIds.SnatTableId[0]
	describeSnatTableEntriesReq.SourceVSwitchId = describeVSwitchesOutput.VSwitches.VSwitch[0].VSwitchId
	describeSnatTableEntriesOutput, err := vpcClient.DescribeSnatTableEntries(describeSnatTableEntriesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry).To(HaveLen(1))
	Expect(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SourceCIDR).To(Equal(workersCIDR))
	infrastructureIdentifier.snatTableId = pointer.StringPtr(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SnatTableId)
	infrastructureIdentifier.snatEntryId = pointer.StringPtr(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SnatEntryId)

	// elastic ips
	describeEipAddressesReq := vpc.CreateDescribeEipAddressesRequest()
	describeEipAddressesReq.EipAddress = describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SnatIp
	describeEipAddressesOutput, err := vpcClient.DescribeEipAddresses(describeEipAddressesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeEipAddressesOutput.EipAddresses.EipAddress).To(HaveLen(1))
	Expect(describeEipAddressesOutput.EipAddresses.EipAddress[0].InternetChargeType).To(Equal(alicloudclient.DefaultInternetChargeType))
	Expect(describeEipAddressesOutput.EipAddresses.EipAddress[0].Name).To(Equal(infra.Namespace + eipSuffix))
	infrastructureIdentifier.elasticIPAllocationID = pointer.StringPtr(describeEipAddressesOutput.EipAddresses.EipAddress[0].AllocationId)

	// security groups
	describeSecurityGroupsReq := ecs.CreateDescribeSecurityGroupsRequest()
	describeSecurityGroupsReq.VpcId = infraStatus.VPC.ID
	describeSecurityGroupsReq.SecurityGroupName = infra.Namespace + securityGroupSuffix
	describeSecurityGroupOutput, err := ecsClient.DescribeSecurityGroups(describeSecurityGroupsReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeSecurityGroupOutput.SecurityGroups.SecurityGroup).To(HaveLen(1))
	infrastructureIdentifier.securityGroupIDs = append(infrastructureIdentifier.securityGroupIDs, describeSecurityGroupOutput.SecurityGroups.SecurityGroup[0].SecurityGroupId)

	// security group rules
	describeSecurityGroupAttributeReq := ecs.CreateDescribeSecurityGroupAttributeRequest()
	describeSecurityGroupAttributeReq.SecurityGroupId = describeSecurityGroupOutput.SecurityGroups.SecurityGroup[0].SecurityGroupId
	describeSecurityGroupAttributeOutput, err := ecsClient.DescribeSecurityGroupAttribute(describeSecurityGroupAttributeReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeSecurityGroupAttributeOutput.Permissions.Permission).To(BeSemanticallyEqualTo([]*ecs.Permission{
		{
			IpProtocol:   "TCP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "30000/32767",
			Priority:     "1",
			SourceCidrIp: allCIDR,
		},
		{
			IpProtocol:   "TCP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "1/65535",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "UDP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "1/65535",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
	}))

	// ecs ssh key pair
	describeKeyPairsReq := ecs.CreateDescribeKeyPairsRequest()
	describeKeyPairsReq.KeyPairName = infra.Namespace + sshKeySuffix
	describeKeyPairOutput, err := ecsClient.DescribeKeyPairs(describeKeyPairsReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeKeyPairOutput.KeyPairs.KeyPair[0].KeyPairFingerPrint).To(Equal(sshPublicKeyDigest))
	infrastructureIdentifier.keyPairName = pointer.StringPtr(describeKeyPairOutput.KeyPairs.KeyPair[0].KeyPairName)

	return
}

func verifyDeletion(ctx context.Context, alicloudClient *aliClient, infrastructureIdentifier infrastructureIdentifiers) {
	vpcClient := alicloudClient.VPC
	ecsClient := alicloudClient.ECS

	// vpc
	if infrastructureIdentifier.vpcID != nil {
		describeVPCsReq := vpc.CreateDescribeVpcsRequest()
		describeVPCsReq.VpcId = *infrastructureIdentifier.vpcID
		describeVpcsOutput, err := vpcClient.DescribeVpcs(describeVPCsReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(describeVpcsOutput.Vpcs.Vpc).To(BeEmpty())
	}

	// vswitch
	if infrastructureIdentifier.vswitchID != nil {
		describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
		describeVSwitchesReq.VSwitchId = *infrastructureIdentifier.vswitchID
		describeVSwitchesOutput, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(describeVSwitchesOutput.VSwitches.VSwitch).To(BeEmpty())
	}

	// nat gateway
	if infrastructureIdentifier.natGatewayID != nil {
		describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
		describeNATGatewaysReq.NatGatewayId = *infrastructureIdentifier.natGatewayID
		describeNatGatewaysOutput, err := vpcClient.DescribeNatGateways(describeNATGatewaysReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(describeNatGatewaysOutput.NatGateways.NatGateway).To(BeEmpty())
	}

	// snat entries
	if infrastructureIdentifier.snatEntryId != nil && infrastructureIdentifier.snatTableId != nil {
		describeSnatTableEntriesReq := vpc.CreateDescribeSnatTableEntriesRequest()
		describeSnatTableEntriesReq.SnatTableId = *infrastructureIdentifier.snatTableId
		describeSnatTableEntriesReq.SnatEntryId = *infrastructureIdentifier.snatEntryId
		describeSnatTableEntriesOutput, err := vpcClient.DescribeSnatTableEntries(describeSnatTableEntriesReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry).To(BeEmpty())
	}

	// elastic ip
	if infrastructureIdentifier.elasticIPAllocationID != nil {
		describeEipAddressesReq := vpc.CreateDescribeEipAddressesRequest()
		describeEipAddressesReq.AllocationId = *infrastructureIdentifier.elasticIPAllocationID
		describeEipAddressesOutput, err := vpcClient.DescribeEipAddresses(describeEipAddressesReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(describeEipAddressesOutput.EipAddresses.EipAddress).To(BeEmpty())
	}

	// security groups
	if len(infrastructureIdentifier.securityGroupIDs) > 0 {
		describeSecurityGroupsReq := ecs.CreateDescribeSecurityGroupsRequest()
		for _, securityGroupID := range infrastructureIdentifier.securityGroupIDs {
			describeSecurityGroupsReq.SecurityGroupId = securityGroupID
			describeSecurityGroupOutput, err := ecsClient.DescribeSecurityGroups(describeSecurityGroupsReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(describeSecurityGroupOutput.SecurityGroups.SecurityGroup).To(BeEmpty())
		}
	}

	// ecs ssh key pair
	if infrastructureIdentifier.keyPairName != nil {
		describeKeyPairsReq := ecs.CreateDescribeKeyPairsRequest()
		describeKeyPairsReq.KeyPairName = *infrastructureIdentifier.keyPairName
		describeKeyPairOutput, err := ecsClient.DescribeKeyPairs(describeKeyPairsReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(describeKeyPairOutput.KeyPairs.KeyPair).To(BeEmpty())
	}
}
