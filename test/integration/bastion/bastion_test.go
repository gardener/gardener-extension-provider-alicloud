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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	aliapi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	alicloudinstall "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"

	bastionctrl "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/bastion"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/gardener/gardener/pkg/logger"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	availableStatus     = "Available"
	userDataConst       = "IyEvYmluL2Jhc2ggLWV1CmlkIGdhcmRlbmVyIHx8IHVzZXJhZGQgZ2FyZGVuZXIgLW1VCm1rZGlyIC1wIC9ob21lL2dhcmRlbmVyLy5zc2gKZWNobyAic3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDazYyeDZrN2orc0lkWG9TN25ITzRrRmM3R0wzU0E2UmtMNEt4VmE5MUQ5RmxhcmtoRzFpeU85WGNNQzZqYnh4SzN3aWt0M3kwVTBkR2h0cFl6Vjh3YmV3Z3RLMWJBWnl1QXJMaUhqbnJnTFVTRDBQazNvWGh6RkpKN0MvRkxNY0tJZFN5bG4vMENKVkVscENIZlU5Y3dqQlVUeHdVQ2pnVXRSYjdZWHN6N1Y5dllIVkdJKzRLaURCd3JzOWtVaTc3QWMyRHQ1UzBJcit5dGN4b0p0bU5tMWgxTjNnNzdlbU8rWXhtWEo4MzFXOThoVFVTeFljTjNXRkhZejR5MWhrRDB2WHE1R1ZXUUtUQ3NzRE1wcnJtN0FjQTBCcVRsQ0xWdWl3dXVmTEJLWGhuRHZRUEQrQ2Jhbk03bUZXRXdLV0xXelZHME45Z1VVMXE1T3hhMzhvODUgbWVAbWFjIiA+IC9ob21lL2dhcmRlbmVyLy5zc2gvYXV0aG9yaXplZF9rZXlzCmNob3duIGdhcmRlbmVyOmdhcmRlbmVyIC9ob21lL2dhcmRlbmVyLy5zc2gvYXV0aG9yaXplZF9rZXlzCmVjaG8gImdhcmRlbmVyIEFMTD0oQUxMKSBOT1BBU1NXRDpBTEwiID4vZXRjL3N1ZG9lcnMuZC85OS1nYXJkZW5lci11c2VyCg=="
	natGatewayType      = "Enhanced"
	vpcCIDR             = "10.250.0.0/16"
	natGatewayCIDR      = "10.250.128.0/21" // Enhanced NatGateway need bind with VSwitch, natGatewayCIDR is used for this VSwitch
	podCIDR             = "100.96.0.0/11"
	securityGroupSuffix = "-sg"
	imageID             = "m-gw8iwwd4iiln01dj646s"
)

var myPublicIP = ""

var (
	accessKeyID     = flag.String("access-key-id", "", "Alicloud access key id")
	accessKeySecret = flag.String("access-key-secret", "", "Alicloud access key secret")
	region          = flag.String("region", "", "Alicloud region")
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
}

type infrastructureIdentifiers struct {
	vpcID            *string
	vswitchID        *string
	natGatewayID     *string
	securityGroupIDs *string
	zone             *string
}

var (
	ctx = context.Background()
	log logr.Logger

	extensionscluster *extensionsv1alpha1.Cluster
	controllercluster *controller.Cluster
	options           *bastionctrl.Options
	bastion           *extensionsv1alpha1.Bastion
	secret            *corev1.Secret

	clientFactory alicloudclient.ClientFactory

	testEnv   *envtest.Environment
	mgrCancel context.CancelFunc
	c         client.Client

	internalChartsPath string
	name               string
	vpcName            string
)

var _ = BeforeSuite(func() {
	flag.Parse()
	validateFlags()

	internalChartsPath = alicloud.InternalChartsPath
	repoRoot := filepath.Join("..", "..", "..")
	alicloud.InternalChartsPath = filepath.Join(repoRoot, alicloud.InternalChartsPath)

	// enable manager logs
	logf.SetLogger(logger.MustNewZapLogger(logger.DebugLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)))
	log = logf.Log.WithName("bastion-test")

	randString, err := randomString()
	Expect(err).NotTo(HaveOccurred())
	// bastion name prefix
	name = fmt.Sprintf("alicloud-it-bastion-%s", randString)
	vpcName = fmt.Sprintf("%s-vpc", name)
	myPublicIP, err = getMyPublicIPWithMask()
	Expect(err).NotTo(HaveOccurred())

	By("starting test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster: pointer.BoolPtr(true),
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_clusters.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_bastions.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_workers.yaml"),
			},
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	By("setup manager")
	mgr, err := manager.New(cfg, manager.Options{
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(extensionsv1alpha1.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(alicloudinstall.AddToScheme(mgr.GetScheme())).To(Succeed())
	Expect(bastionctrl.AddToManager(mgr)).To(Succeed())

	var mgrContext context.Context
	mgrContext, mgrCancel = context.WithCancel(ctx)

	By("start manager")
	go func() {
		err := mgr.Start(mgrContext)
		Expect(err).NotTo(HaveOccurred())
	}()

	c = mgr.GetClient()
	Expect(c).NotTo(BeNil())

	extensionscluster, controllercluster = createClusters(name)

	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1constants.SecretNameCloudProvider,
			Namespace: name,
		},
		Data: map[string][]byte{
			alicloud.AccessKeyID:     []byte(*accessKeyID),
			alicloud.AccessKeySecret: []byte(*accessKeySecret),
		},
	}

	clientFactory = alicloudclient.NewClientFactory()
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

	alicloud.InternalChartsPath = internalChartsPath
})

var _ = Describe("Bastion tests", func() {
	It("should successfully create and delete", func() {
		By("setup Infrastructure ")
		identifiers := prepareVPCandShootSecurityGroup(ctx, clientFactory, name, vpcName, *region, vpcCIDR, natGatewayCIDR)
		framework.AddCleanupAction(func() {
			cleanupVPC(ctx, clientFactory, identifiers)
		})

		By("create namespace for test execution")
		worker := createWorker(name, *identifiers.vpcID, *identifiers.vswitchID, *identifiers.zone, imageID, *identifiers.securityGroupIDs)

		setupEnvironmentObjects(ctx, c, namespace(name), secret, extensionscluster, worker)
		framework.AddCleanupAction(func() {
			teardownShootEnvironment(ctx, c, namespace(name), secret, extensionscluster, worker)
		})

		bastion, options = createBastion(controllercluster, name)

		By("setup bastion")
		err := c.Create(ctx, bastion)
		Expect(err).NotTo(HaveOccurred())

		framework.AddCleanupAction(func() {
			teardownBastion(ctx, log, c, bastion)
			By("verify bastion deletion")
			verifyDeletion(clientFactory, options)
		})

		By("wait until bastion is reconciled")
		Expect(extensions.WaitUntilExtensionObjectReady(
			ctx,
			c,
			log,
			bastion,
			extensionsv1alpha1.BastionResource,
			60*time.Second,
			60*time.Second,
			10*time.Minute,
			nil,
		)).To(Succeed())

		time.Sleep(60 * time.Second)
		verifyPort22IsOpen(ctx, c, bastion)
		verifyPort42IsClosed(ctx, c, bastion)

		By("verify cloud resources")
		verifyCreation(clientFactory, options)
	})
})

func randomString() (string, error) {
	suffix, err := gardenerutils.GenerateRandomStringFromCharset(5, "0123456789abcdefghijklmnopqrstuvwxyz")
	if err != nil {
		return "", err
	}

	return suffix, nil
}

func getMyPublicIPWithMask() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := net.ParseIP(string(body))
	var mask net.IPMask
	if ip.To4() != nil {
		mask = net.CIDRMask(24, 32) // use a /24 net for IPv4
	} else {
		return "", fmt.Errorf("not valid IPv4 address")
	}

	cidr := net.IPNet{
		IP:   ip,
		Mask: mask,
	}

	full := cidr.String()

	_, ipnet, _ := net.ParseCIDR(full)

	return ipnet.String(), nil
}

func verifyPort22IsOpen(ctx context.Context, c client.Client, bastion *extensionsv1alpha1.Bastion) {
	By("check connection to port 22 open should not error")
	bastionUpdated := &extensionsv1alpha1.Bastion{}
	Expect(c.Get(ctx, client.ObjectKey{Namespace: bastion.Namespace, Name: bastion.Name}, bastionUpdated)).To(Succeed())

	ipAddress := bastionUpdated.Status.Ingress.IP
	address := net.JoinHostPort(ipAddress, "22")
	conn, err := net.DialTimeout("tcp", address, 60*time.Second)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(conn).NotTo(BeNil())
}

func verifyPort42IsClosed(ctx context.Context, c client.Client, bastion *extensionsv1alpha1.Bastion) {
	By("check connection to port 42 which should fail")

	bastionUpdated := &extensionsv1alpha1.Bastion{}
	Expect(c.Get(ctx, client.ObjectKey{Namespace: bastion.Namespace, Name: bastion.Name}, bastionUpdated)).To(Succeed())

	ipAddress := bastionUpdated.Status.Ingress.IP
	address := net.JoinHostPort(ipAddress, "42")
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	Expect(err).Should(HaveOccurred())
	Expect(conn).To(BeNil())
}

func createClusters(name string) (*extensionsv1alpha1.Cluster, *controller.Cluster) {
	infrastructureConfig := createInfrastructureConfig()
	infrastructureConfigJSON, _ := json.Marshal(&infrastructureConfig)

	shoot := createShoot(infrastructureConfigJSON)
	shootJSON, _ := json.Marshal(shoot)

	cloudProfile := createCloudProfile()
	cloudProfileJSON, _ := json.Marshal(cloudProfile)

	extensionscluster := &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			CloudProfile: runtime.RawExtension{
				Object: cloudProfile,
				Raw:    cloudProfileJSON,
			},
			Seed: runtime.RawExtension{
				Raw: []byte("{}"),
			},
			Shoot: runtime.RawExtension{
				Object: shoot,
				Raw:    shootJSON,
			},
		},
	}

	cluster := &controller.Cluster{
		ObjectMeta:   metav1.ObjectMeta{Name: name},
		Shoot:        shoot,
		CloudProfile: cloudProfile,
	}
	return extensionscluster, cluster
}

func createInfrastructureConfig() *alicloudv1alpha1.InfrastructureConfig {
	return &alicloudv1alpha1.InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: alicloudv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureConfig",
		},
	}
}

func createWorker(name, vpcID, vSwitchID, zone, machineImageID, shootSecurityGroupID string) *extensionsv1alpha1.Worker {
	return &extensionsv1alpha1.Worker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
		Spec: extensionsv1alpha1.WorkerSpec{
			DefaultSpec: extensionsv1alpha1.DefaultSpec{
				Type: alicloud.Type,
			},
			InfrastructureProviderStatus: &runtime.RawExtension{
				Object: &aliapi.InfrastructureStatus{
					VPC: aliapi.VPCStatus{
						ID: vpcID,
						VSwitches: []aliapi.VSwitch{
							{
								ID:   vSwitchID,
								Zone: zone,
							},
						},
						SecurityGroups: []aliapi.SecurityGroup{
							{
								ID: shootSecurityGroupID,
							},
						},
					},
					MachineImages: []aliapi.MachineImage{
						{
							ID: machineImageID,
						},
					},
				},
			},
			Pools: []extensionsv1alpha1.WorkerPool{},
		},
	}
}

func createShoot(infrastructureConfig []byte) *gardencorev1beta1.Shoot {
	return &gardencorev1beta1.Shoot{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.gardener.cloud/v1beta1",
			Kind:       "Shoot",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: gardencorev1beta1.ShootSpec{
			Region:            *region,
			SecretBindingName: v1beta1constants.SecretNameCloudProvider,
			Provider: gardencorev1beta1.Provider{
				InfrastructureConfig: &runtime.RawExtension{
					Raw: infrastructureConfig,
				},
			},
			Networking: gardencorev1beta1.Networking{
				Pods: pointer.String(podCIDR),
			},
		},
	}
}

func createCloudProfile() *gardencorev1beta1.CloudProfile {
	cloudProfileConfig := &alicloudv1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: alicloudv1alpha1.SchemeGroupVersion.String(),
			Kind:       "CloudProfileConfig",
		},
	}

	cloudProfileConfigJSON, _ := json.Marshal(cloudProfileConfig)

	cloudProfile := &gardencorev1beta1.CloudProfile{
		Spec: gardencorev1beta1.CloudProfileSpec{
			ProviderConfig: &runtime.RawExtension{
				Raw: cloudProfileConfigJSON,
			},
		},
	}
	return cloudProfile
}

func createBastion(cluster *controller.Cluster, name string) (*extensionsv1alpha1.Bastion, *bastionctrl.Options) {
	bastion := &extensionsv1alpha1.Bastion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-bastion",
			Namespace: name,
		},
		Spec: extensionsv1alpha1.BastionSpec{
			DefaultSpec: extensionsv1alpha1.DefaultSpec{
				Type: alicloud.Type,
			},
			UserData: []byte(userDataConst),
			Ingress: []extensionsv1alpha1.BastionIngressPolicy{
				{IPBlock: networkingv1.IPBlock{
					CIDR: myPublicIP,
				}},
			},
		},
	}

	options, err := bastionctrl.DetermineOptions(bastion, cluster)
	Expect(err).NotTo(HaveOccurred())

	return bastion, options
}

func prepareVPCandShootSecurityGroup(ctx context.Context, clientFactory alicloudclient.ClientFactory, name, vpcName, region, vpcCIDR, natGatewayCIDR string) infrastructureIdentifiers {
	vpcClient, err := clientFactory.NewVPCClient(region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	// vpc
	createVpcReq := vpc.CreateCreateVpcRequest()
	createVpcReq.CidrBlock = vpcCIDR
	createVpcReq.RegionId = region
	createVpcReq.VpcName = vpcName
	createVpcReq.Description = name
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

	// vswitch
	createVSwitchsReq := vpc.CreateCreateVSwitchRequest()
	createVSwitchsReq.VpcId = createVPCsResp.VpcId
	createVSwitchsReq.RegionId = region
	createVSwitchsReq.CidrBlock = natGatewayCIDR
	createVSwitchsReq.ZoneId = region + "a"
	createVSwitchsReq.Description = name
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

	// natgateway
	createNatGatewayReq := vpc.CreateCreateNatGatewayRequest()
	createNatGatewayReq.VpcId = createVPCsResp.VpcId
	createNatGatewayReq.RegionId = region
	createNatGatewayReq.VSwitchId = createVSwitchsResp.VSwitchId
	createNatGatewayReq.NatType = natGatewayType
	createNatGatewayReq.Description = name
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

	// shoot security group
	ecsClient, err := clientFactory.NewECSClient(region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	createSecurityGroupsResp, err := ecsClient.CreateSecurityGroups(createVPCsResp.VpcId, name+securityGroupSuffix)
	Expect(err).NotTo(HaveOccurred())

	return infrastructureIdentifiers{
		vpcID:            pointer.StringPtr(createVPCsResp.VpcId),
		vswitchID:        pointer.StringPtr(createVSwitchsResp.VSwitchId),
		natGatewayID:     pointer.StringPtr(createNatGatewayResp.NatGatewayId),
		securityGroupIDs: pointer.StringPtr(createSecurityGroupsResp.SecurityGroupId),
		zone:             pointer.StringPtr(createVSwitchsReq.ZoneId),
	}
}

func cleanupVPC(ctx context.Context, clientFactory alicloudclient.ClientFactory, identifiers infrastructureIdentifiers) {
	vpcClient, err := clientFactory.NewVPCClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())
	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	// cleanup - natGateWay
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

	err = ecsClient.DeleteSecurityGroups(*identifiers.securityGroupIDs)
	Expect(err).NotTo(HaveOccurred())

	// cleanup - vswitch
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

	// cleanup - vpc
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

func namespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func setupEnvironmentObjects(ctx context.Context, c client.Client, namespace *corev1.Namespace, secret *corev1.Secret, cluster *extensionsv1alpha1.Cluster, worker *extensionsv1alpha1.Worker) {
	Expect(c.Create(ctx, namespace)).To(Succeed())
	Expect(c.Create(ctx, cluster)).To(Succeed())
	Expect(c.Create(ctx, secret)).To(Succeed())
	Expect(c.Create(ctx, worker)).To(Succeed())
}

func teardownShootEnvironment(ctx context.Context, c client.Client, namespace *corev1.Namespace, secret *corev1.Secret, cluster *extensionsv1alpha1.Cluster, worker *extensionsv1alpha1.Worker) {
	Expect(client.IgnoreNotFound(c.Delete(ctx, worker))).To(Succeed())
	Expect(client.IgnoreNotFound(c.Delete(ctx, secret))).To(Succeed())
	Expect(client.IgnoreNotFound(c.Delete(ctx, cluster))).To(Succeed())
	Expect(client.IgnoreNotFound(c.Delete(ctx, namespace))).To(Succeed())
}

func teardownBastion(ctx context.Context, logger logr.Logger, c client.Client, bastion *extensionsv1alpha1.Bastion) {
	By("delete bastion")
	Expect(client.IgnoreNotFound(c.Delete(ctx, bastion))).To(Succeed())

	By("wait until bastion is deleted")
	err := extensions.WaitUntilExtensionObjectDeleted(ctx, c, logger, bastion, extensionsv1alpha1.BastionResource, 20*time.Second, 15*time.Minute)
	Expect(err).NotTo(HaveOccurred())
}

func verifyDeletion(clientFactory alicloudclient.ClientFactory, options *bastionctrl.Options) {
	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	By("bastion instance should be gone")
	response, err := ecsClient.GetInstances(options.BastionInstanceName)
	Expect(err).NotTo(HaveOccurred())
	Expect(response.Instances.Instance).To(HaveLen(0))

	By("bastion security group should be gone")
	sgResponse, err := ecsClient.GetSecurityGroup(options.SecurityGroupName)
	Expect(err).NotTo(HaveOccurred())
	Expect(sgResponse.SecurityGroups.SecurityGroup).To(HaveLen(0))
}

func verifyCreation(clientFactory alicloudclient.ClientFactory, options *bastionctrl.Options) {
	ecsClient, err := clientFactory.NewECSClient(*region, *accessKeyID, *accessKeySecret)
	Expect(err).NotTo(HaveOccurred())

	By("checking bastion instance")
	response, err := ecsClient.GetInstances(options.BastionInstanceName)
	Expect(err).NotTo(HaveOccurred())
	Expect(response.Instances.Instance[0].InstanceName).To(Equal(options.BastionInstanceName))

	By("checking bastion security group")
	sgResponse, err := ecsClient.GetSecurityGroup(options.SecurityGroupName)
	Expect(err).NotTo(HaveOccurred())
	Expect(sgResponse.SecurityGroups.SecurityGroup[0].SecurityGroupName).To(Equal(options.SecurityGroupName))

	By("checking bastion security group rules")
	describeSecurityGroupAttributeReq := ecs.CreateDescribeSecurityGroupAttributeRequest()
	describeSecurityGroupAttributeReq.SecurityGroupId = sgResponse.SecurityGroups.SecurityGroup[0].SecurityGroupId
	describeSecurityGroupAttributeReq.RegionId = options.Region
	rulesResponse, err := ecsClient.DescribeSecurityGroupAttribute(describeSecurityGroupAttributeReq)
	Expect(err).NotTo(HaveOccurred())
	Expect(rulesResponse.Permissions.Permission).To(HaveLen(3))
}
