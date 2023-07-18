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

package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"path/filepath"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/matchers"
	alicloudinstall "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
)

var (
	ctx = context.Background()
	log logr.Logger

	testEnv   *envtest.Environment
	mgrCancel context.CancelFunc
	c         client.Client
	decoder   runtime.Decoder

	clientFactory alicloudclient.ClientFactory

	availabilityZone string
)

var _ = BeforeSuite(func() {
	repoRoot := filepath.Join("..", "..", "..")

	// enable manager logs
	logf.SetLogger(logger.MustNewZapLogger(logger.DebugLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)))

	log = logf.Log.WithName("infrastructure-test")

	By("starting test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster: pointer.Bool(true),
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_clusters.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_infrastructures.yaml"),
			},
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	By("setup manager")
	mgr, err := manager.New(cfg, manager.Options{
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	Expect(extensionsv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(alicloudinstall.AddToScheme(mgr.GetScheme())).To(Succeed())

	Expect(infrastructure.AddToManagerWithOptions(ctx, mgr, infrastructure.AddOptions{
		// During testing in testmachinery cluster, there is no gardener-resource-manager to inject the volume mount.
		// Hence, we need to run without projected token mount.
		DisableProjectedTokenMount: true,
	})).To(Succeed())

	var mgrContext context.Context
	mgrContext, mgrCancel = context.WithCancel(ctx)

	By("start manager")
	go func() {
		err := mgr.Start(mgrContext)
		Expect(err).NotTo(HaveOccurred())
	}()

	c = mgr.GetClient()
	Expect(c).ToNot(BeNil())
	decoder = serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder()

	flag.Parse()
	validateFlags()

	clientFactory = alicloudclient.NewClientFactory()

	availabilityZone = getSingleZone(*region)

	By("ensure encrypted image is cleaned in the current account")
	Expect(deleteEncryptedImageStackIfExists(mgrContext, clientFactory)).To(Succeed())

	priorityClass := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: v1beta1constants.PriorityClassNameShootControlPlane300,
		},
		Description:   "PriorityClass for Shoot control plane components",
		GlobalDefault: false,
		Value:         999998300,
	}
	Expect(client.IgnoreAlreadyExists(c.Create(ctx, priorityClass))).To(BeNil())
})

var _ = AfterSuite(func() {
	defer func() {
		By("stopping manager")
		mgrCancel()
	}()

	By("running cleanup actions")
	framework.RunCleanupActions()

	By("stopping test environment")
	Expect(testEnv.Stop()).To(Succeed())
})

var _ = Describe("Infrastructure tests", func() {
	Context("with infrastructure that requests new vpc (networks.vpc.cidr)", func() {
		It("should successfully create and delete", func() {
			providerConfig := newProviderConfig(&alicloudv1alpha1.VPC{
				CIDR: pointer.String(vpcCIDR),
			}, availabilityZone)

			err := runTest(ctx, log, c, providerConfig, decoder, clientFactory)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with infrastructure that requests existing vpc", func() {
		It("should successfully create and delete", func() {
			identifiers := prepareVPC(ctx, clientFactory, *region, vpcCIDR, natGatewayCIDR)
			framework.AddCleanupAction(func() {
				cleanupVPC(ctx, clientFactory, identifiers)
			})

			providerConfig := newProviderConfig(&alicloudv1alpha1.VPC{
				ID: identifiers.vpcID,
			}, availabilityZone)

			err := runTest(ctx, log, c, providerConfig, decoder, clientFactory)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with invalid credentials", func() {
		It("should fail creation but succeed deletion", func() {
			providerConfig := newProviderConfig(&alicloudv1alpha1.VPC{
				CIDR: pointer.String(vpcCIDR),
			}, availabilityZone)

			var (
				namespace *corev1.Namespace
				cluster   *extensionsv1alpha1.Cluster
				infra     *extensionsv1alpha1.Infrastructure
				err       error
			)

			framework.AddCleanupAction(func() {
				By("cleaning up namespace and cluster")
				Expect(client.IgnoreNotFound(c.Delete(ctx, namespace))).To(Succeed())
				Expect(client.IgnoreNotFound(c.Delete(ctx, cluster))).To(Succeed())
			})

			defer func() {
				By("delete infrastructure")
				Expect(client.IgnoreNotFound(c.Delete(ctx, infra))).To(Succeed())

				By("wait until infrastructure is deleted")
				// deletion should succeed even though creation failed with invalid credentials (no-op)
				err := extensions.WaitUntilExtensionObjectDeleted(
					ctx,
					c,
					log,
					infra,
					extensionsv1alpha1.InfrastructureResource,
					10*time.Second,
					30*time.Minute,
				)
				Expect(err).NotTo(HaveOccurred())
			}()

			By("create namespace for test execution")
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "provider-alicloud-test-",
				},
			}
			Expect(c.Create(ctx, namespace)).To(Succeed())

			By("deploy invalid cloudprovider secret into namespace")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace.Name,
				},
				Data: map[string][]byte{
					alicloud.AccessKeyID:     []byte("invalid"),
					alicloud.AccessKeySecret: []byte("fake"),
				},
			}
			Expect(c.Create(ctx, secret)).To(Succeed())

			By("create cluster which contains information of shoot info. It is used for encrypted image testing")
			cluster, err = newCluster(namespace.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Create(ctx, cluster)).To(Succeed())

			By("create infrastructure")
			infra, err = newInfrastructure(namespace.Name, providerConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Create(ctx, infra)).To(Succeed())

			By("wait until infrastructure creation has failed")
			err = extensions.WaitUntilExtensionObjectReady(
				ctx,
				c,
				log,
				infra,
				extensionsv1alpha1.InfrastructureResource,
				10*time.Second,
				30*time.Second,
				5*time.Minute,
				nil,
			)
			Expect(err).To(MatchError(ContainSubstring("Specified access key is not found")))
			var errorWithCode *gardencorev1beta1helper.ErrorWithCodes
			Expect(errors.As(err, &errorWithCode)).To(BeTrue())
			Expect(errorWithCode.Codes()).To(ConsistOf(gardencorev1beta1.ErrorInfraUnauthenticated, gardencorev1beta1.ErrorConfigurationProblem))
		})
	})
})

func runTest(ctx context.Context, logger logr.Logger, c client.Client, providerConfig *alicloudv1alpha1.InfrastructureConfig, decoder runtime.Decoder, clientFactory alicloudclient.ClientFactory) error {
	var (
		namespace                 *corev1.Namespace
		cluster                   *extensionsv1alpha1.Cluster
		infra                     *extensionsv1alpha1.Infrastructure
		infrastructureIdentifiers infrastructureIdentifiers
		err                       error
	)

	framework.AddCleanupAction(func() {
		By("cleaning up namespace and cluster")
		Expect(client.IgnoreNotFound(c.Delete(ctx, namespace))).To(Succeed())
		Expect(client.IgnoreNotFound(c.Delete(ctx, cluster))).To(Succeed())
	})

	defer func() {
		By("delete infrastructure")
		Expect(client.IgnoreNotFound(c.Delete(ctx, infra))).To(Succeed())

		By("wait until infrastructure is deleted")
		err := extensions.WaitUntilExtensionObjectDeleted(
			ctx,
			c,
			logger,
			infra,
			extensionsv1alpha1.InfrastructureResource,
			10*time.Second,
			30*time.Minute,
		)
		Expect(err).NotTo(HaveOccurred())

		By("verify infrastructure deletion")
		verifyDeletion(clientFactory, infrastructureIdentifiers)
	}()

	By("create namespace for test execution")
	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "provider-alicloud-test-",
		},
	}
	if err := c.Create(ctx, namespace); err != nil {
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

	By("create cluster which contains information of shoot info. It is used for encrypted image testing")
	cluster, err = newCluster(namespace.Name)
	if err != nil {
		return err
	}

	if err := c.Create(ctx, cluster); err != nil {
		return err
	}

	By("create infrastructure")
	infra, err = newInfrastructure(namespace.Name, providerConfig)
	if err != nil {
		return err
	}

	if err := c.Create(ctx, infra); err != nil {
		return err
	}

	if *enableEncryptedImage {
		By("wait until encrypted image is ready")
		if err := verifyStackExists(ctx, clientFactory); err != nil {
			return err
		}
	}

	By("wait until infrastructure is created")
	if err := extensions.WaitUntilExtensionObjectReady(
		ctx,
		c,
		logger,
		infra,
		extensionsv1alpha1.InfrastructureResource,
		10*time.Second,
		30*time.Second,
		16*time.Minute,
		nil,
	); err != nil {
		return err
	}

	By("decode infrastructure status")
	if err := c.Get(ctx, client.ObjectKey{Namespace: infra.Namespace, Name: infra.Name}, infra); err != nil {
		return err
	}

	providerStatus := &alicloudv1alpha1.InfrastructureStatus{}
	if _, _, err := decoder.Decode(infra.Status.ProviderStatus.Raw, nil, providerStatus); err != nil {
		return err
	}

	By("verify infrastructure creation")
	infrastructureIdentifiers = verifyCreation(clientFactory, infra, providerStatus, providerConfig)

	if *enableEncryptedImage {
		By("verify image prepared in infrastructure status")
		if err := verifyImageInfraStatus(providerStatus); err != nil {
			return err
		}
	}

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
			Region: *region,
		},
	}, nil
}

type infrastructureIdentifiers struct {
	vpcID                 *string
	vswitchID             *string
	natGatewayID          *string
	securityGroupIDs      []string
	elasticIPAllocationID *string
	snatTableId           *string
	snatEntryId           *string
}

func verifyCreation(
	clientFactory alicloudclient.ClientFactory,
	infra *extensionsv1alpha1.Infrastructure,
	infraStatus *alicloudv1alpha1.InfrastructureStatus,
	providerConfig *alicloudv1alpha1.InfrastructureConfig,
) (
	infrastructureIdentifier infrastructureIdentifiers,
) {
	const (
		eipSuffix           = "-eip-natgw-z0"
		securityGroupSuffix = "-sg"
	)

	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	// vpc
	describeVPCsReq := vpc.CreateDescribeVpcsRequest()
	describeVPCsReq.VpcId = infraStatus.VPC.ID
	describeVpcsOutput, err := vpcClient.DescribeVpcs(describeVPCsReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeVpcsOutput.Vpcs.Vpc).To(HaveLen(1))
	Expect(describeVpcsOutput.Vpcs.Vpc[0].VpcId).To(Equal(infraStatus.VPC.ID))
	Expect(describeVpcsOutput.Vpcs.Vpc[0].CidrBlock).To(Equal(vpcCIDR))
	if providerConfig.Networks.VPC.CIDR != nil {
		infrastructureIdentifier.vpcID = pointer.String(describeVpcsOutput.Vpcs.Vpc[0].VpcId)
	}

	// vswitch
	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VpcId = infraStatus.VPC.ID
	describeVSwitchesOutput, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeVSwitchesOutput.VSwitches.VSwitch[0].CidrBlock).To(Equal(workersCIDR))
	Expect(describeVSwitchesOutput.VSwitches.VSwitch[0].ZoneId).To(Equal(providerConfig.Networks.Zones[0].Name))
	infrastructureIdentifier.vswitchID = pointer.String(describeVSwitchesOutput.VSwitches.VSwitch[0].VSwitchId)
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
		infrastructureIdentifier.natGatewayID = pointer.String(describeNatGatewaysOutput.NatGateways.NatGateway[0].NatGatewayId)
	}

	// snat entries
	describeSnatTableEntriesReq := vpc.CreateDescribeSnatTableEntriesRequest()
	describeSnatTableEntriesReq.SnatTableId = describeNatGatewaysOutput.NatGateways.NatGateway[0].SnatTableIds.SnatTableId[0]
	describeSnatTableEntriesReq.SourceVSwitchId = describeVSwitchesOutput.VSwitches.VSwitch[0].VSwitchId
	describeSnatTableEntriesOutput, err := vpcClient.DescribeSnatTableEntries(describeSnatTableEntriesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry).To(HaveLen(1))
	Expect(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SourceCIDR).To(Equal(workersCIDR))
	infrastructureIdentifier.snatTableId = pointer.String(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SnatTableId)
	infrastructureIdentifier.snatEntryId = pointer.String(describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SnatEntryId)

	// elastic ips
	describeEipAddressesReq := vpc.CreateDescribeEipAddressesRequest()
	describeEipAddressesReq.EipAddress = describeSnatTableEntriesOutput.SnatTableEntries.SnatTableEntry[0].SnatIp
	describeEipAddressesOutput, err := vpcClient.DescribeEipAddresses(describeEipAddressesReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(describeEipAddressesOutput.EipAddresses.EipAddress).To(HaveLen(1))
	Expect(describeEipAddressesOutput.EipAddresses.EipAddress[0].InternetChargeType).To(Equal(alicloudclient.DefaultInternetChargeType))
	Expect(describeEipAddressesOutput.EipAddresses.EipAddress[0].Name).To(Equal(infra.Namespace + eipSuffix))
	infrastructureIdentifier.elasticIPAllocationID = pointer.String(describeEipAddressesOutput.EipAddresses.EipAddress[0].AllocationId)

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
			PortRange:    "1/22",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "TCP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "24/513",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "TCP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "515/65535",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "UDP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "1/22",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "UDP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "24/513",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "UDP",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "515/65535",
			Priority:     "1",
			SourceCidrIp: vpcCIDR,
		},
		{
			IpProtocol:   "ALL",
			Direction:    "ingress",
			Policy:       "Accept",
			PortRange:    "-1/-1",
			Priority:     "1",
			SourceCidrIp: podCIDR,
		},
	}))

	return
}

func verifyDeletion(clientFactory alicloudclient.ClientFactory, infrastructureIdentifier infrastructureIdentifiers) {
	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

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
		describeSnatTableEntriesOutput, _ := vpcClient.DescribeSnatTableEntries(describeSnatTableEntriesReq)
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

}

func prepareVPC(ctx context.Context, clientFactory alicloudclient.ClientFactory, region, vpcCIDR, natGatewayCIDR string) infrastructureIdentifiers {
	vpcClient, err := clientFactory.NewVPCClient(region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())
	createVpcReq := vpc.CreateCreateVpcRequest()
	createVpcReq.VpcName = "provider-alicloud-infra-test"
	createVpcReq.CidrBlock = vpcCIDR
	createVpcReq.RegionId = region
	createVPCsResp, err := vpcClient.CreateVpc(createVpcReq)
	Expect(err).NotTo(HaveOccurred())

	describeVpcsReq := vpc.CreateDescribeVpcsRequest()
	describeVpcsReq.VpcId = createVPCsResp.VpcId
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeVpcsResp, err := vpcClient.DescribeVpcs(describeVpcsReq)
		if err != nil {
			return false, err
		}

		if describeVpcsResp.Vpcs.Vpc[0].Status != availableStatus {
			return false, nil
		}

		return true, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())

	createVSwitchsReq := vpc.CreateCreateVSwitchRequest()
	createVSwitchsReq.VpcId = createVPCsResp.VpcId
	createVSwitchsReq.RegionId = region
	createVSwitchsReq.CidrBlock = natGatewayCIDR
	createVSwitchsReq.ZoneId = getSingleZone(region)
	createVSwitchsResp, err := vpcClient.CreateVSwitch(createVSwitchsReq)
	Expect(err).NotTo(HaveOccurred())

	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VSwitchId = createVSwitchsResp.VSwitchId
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeVSwitchesResp, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)
		if err != nil {
			return false, err
		}

		if describeVSwitchesResp.VSwitches.VSwitch[0].Status != availableStatus {
			return false, nil
		}

		return true, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())

	createNatGatewayReq := vpc.CreateCreateNatGatewayRequest()
	createNatGatewayReq.VpcId = createVPCsResp.VpcId
	createNatGatewayReq.RegionId = region
	createNatGatewayReq.VSwitchId = createVSwitchsResp.VSwitchId
	createNatGatewayReq.NatType = natGatewayType
	createNatGatewayResp, err := vpcClient.CreateNatGateway(createNatGatewayReq)
	Expect(err).NotTo(HaveOccurred())

	describeNatGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
	describeNatGatewaysReq.NatGatewayId = createNatGatewayResp.NatGatewayId
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeNatGatewaysResp, err := vpcClient.DescribeNatGateways(describeNatGatewaysReq)
		if err != nil {
			return false, err
		}

		if describeNatGatewaysResp.NatGateways.NatGateway[0].Status != availableStatus {
			return false, nil
		}

		return true, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())

	return infrastructureIdentifiers{
		vpcID:        pointer.String(createVPCsResp.VpcId),
		vswitchID:    pointer.String(createVSwitchsResp.VSwitchId),
		natGatewayID: pointer.String(createNatGatewayResp.NatGatewayId),
	}
}

func cleanupVPC(ctx context.Context, clientFactory alicloudclient.ClientFactory, identifiers infrastructureIdentifiers) {
	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())
	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	deleteNatGatewayReq := vpc.CreateDeleteNatGatewayRequest()
	deleteNatGatewayReq.NatGatewayId = *identifiers.natGatewayID
	_, err = vpcClient.DeleteNatGateway(deleteNatGatewayReq)
	Expect(err).NotTo(HaveOccurred())

	describeNatGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
	describeNatGatewaysReq.NatGatewayId = *identifiers.natGatewayID
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeNatGatewaysResp, err := vpcClient.DescribeNatGateways(describeNatGatewaysReq)
		if err != nil {
			return false, err
		}

		if len(describeNatGatewaysResp.NatGateways.NatGateway) == 0 {
			return true, nil
		}

		return false, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())

	describeSecurityGroupsReq := ecs.CreateDescribeSecurityGroupsRequest()
	describeSecurityGroupsReq.VpcId = *identifiers.vpcID
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeSecurityGroupsResp, err := ecsClient.DescribeSecurityGroups(describeSecurityGroupsReq)

		if err != nil {
			return false, err
		}

		if len(describeSecurityGroupsResp.SecurityGroups.SecurityGroup) == 0 {
			return true, nil
		}

		return false, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())

	deleteVSwitchReq := vpc.CreateDeleteVSwitchRequest()
	deleteVSwitchReq.VSwitchId = *identifiers.vswitchID
	_, err = vpcClient.DeleteVSwitch(deleteVSwitchReq)
	Expect(err).NotTo(HaveOccurred())

	describeVSwitchesReq := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesReq.VSwitchId = *identifiers.vswitchID
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeVSwitchesResp, err := vpcClient.DescribeVSwitches(describeVSwitchesReq)

		if err != nil {
			return false, err
		}

		if len(describeVSwitchesResp.VSwitches.VSwitch) == 0 {
			return true, nil
		}

		return false, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())

	deleteVpcReq := vpc.CreateDeleteVpcRequest()
	deleteVpcReq.VpcId = *identifiers.vpcID
	_, err = vpcClient.DeleteVpc(deleteVpcReq)
	Expect(err).NotTo(HaveOccurred())

	describeVpcsReq := vpc.CreateDescribeVpcsRequest()
	describeVpcsReq.VpcId = *identifiers.vpcID
	err = wait.PollUntil(5*time.Second, func() (bool, error) {
		describeVpcsResp, err := vpcClient.DescribeVpcs(describeVpcsReq)

		if err != nil {
			return false, err
		}

		if len(describeVpcsResp.Vpcs.Vpc) == 0 {
			return true, nil
		}

		return false, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())
}
