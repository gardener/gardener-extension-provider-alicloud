// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/common"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/imagevector"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
	mockinfrastructure "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/controller/infrastructure"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	realterraformer "github.com/gardener/gardener/extensions/pkg/terraformer"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/chartrenderer"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	mockchartrenderer "github.com/gardener/gardener/pkg/mock/gardener/chartrenderer"
	mockgardenerchartrenderer "github.com/gardener/gardener/pkg/mock/gardener/chartrenderer"
	mockterraformer "github.com/gardener/gardener/pkg/mock/gardener/extensions/terraformer"
	"github.com/gardener/gardener/pkg/mock/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/manifest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

func ExpectInject(ok bool, err error) {
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeTrue(), "no injection happened")
}

func ExpectEncode(data []byte, err error) []byte {
	Expect(err).NotTo(HaveOccurred())
	Expect(data).NotTo(BeNil())
	return data
}

func mkManifest(name string, content string) manifest.Manifest {
	return manifest.Manifest{
		Name:    fmt.Sprintf("/templates/%s", name),
		Content: content,
	}
}

var _ = Describe("Actuator", func() {
	var (
		ctrl       *gomock.Controller
		scheme     *runtime.Scheme
		serializer runtime.Serializer
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		scheme = runtime.NewScheme()
		install.Install(scheme)
		Expect(controller.AddToScheme(scheme)).To(Succeed())
		serializer = json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Actuator", func() {
		var (
			ctx                   context.Context
			logger                *logr.MockLogger
			alicloudClientFactory *mockalicloudclient.MockClientFactory
			vpcClient             *mockalicloudclient.MockVPC
			terraformerFactory    *mockterraformer.MockFactory
			terraformer           *mockterraformer.MockTerraformer
			shootECSClient        *mockalicloudclient.MockECS
			shootSTSClient        *mockalicloudclient.MockSTS
			chartRendererFactory  *mockchartrenderer.MockFactory
			terraformChartOps     *mockinfrastructure.MockTerraformChartOps
			actuator              infrastructure.Actuator
			c                     *mockclient.MockClient
			initializer           *mockterraformer.MockInitializer
			restConfig            rest.Config

			chartRenderer *mockgardenerchartrenderer.MockInterface

			cidr   string
			config alicloudv1alpha1.InfrastructureConfig

			configYAML      []byte
			secretNamespace string
			secretName      string
			region          string
			infra           extensionsv1alpha1.Infrastructure
			accessKeyID     string
			accessKeySecret string
			cluster         controller.Cluster

			initializerValues InitializerValues
			chartValues       map[string]interface{}

			mainContent      string
			variablesContent string
			tfVarsContent    string

			vpcID           string
			vpcCIDRString   string
			natGatewayID    string
			securityGroupID string
			keyPairName     string
			rawState        *realterraformer.RawState
		)

		Describe("#Reconcile", func() {
			BeforeEach(func() {
				ctx = context.TODO()
				logger = logr.NewMockLogger(ctrl)
				alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
				vpcClient = mockalicloudclient.NewMockVPC(ctrl)
				terraformerFactory = mockterraformer.NewMockFactory(ctrl)
				terraformer = mockterraformer.NewMockTerraformer(ctrl)
				shootECSClient = mockalicloudclient.NewMockECS(ctrl)
				shootSTSClient = mockalicloudclient.NewMockSTS(ctrl)
				chartRendererFactory = mockchartrenderer.NewMockFactory(ctrl)
				terraformChartOps = mockinfrastructure.NewMockTerraformChartOps(ctrl)
				actuator = NewActuatorWithDeps(
					logger,
					alicloudClientFactory,
					terraformerFactory,
					chartRendererFactory,
					terraformChartOps,
					nil,
					nil,
				)
				c = mockclient.NewMockClient(ctrl)
				initializer = mockterraformer.NewMockInitializer(ctrl)

				chartRenderer = mockgardenerchartrenderer.NewMockInterface(ctrl)

				cidr = "192.168.0.0/16"
				config = alicloudv1alpha1.InfrastructureConfig{
					Networks: alicloudv1alpha1.Networks{
						VPC: alicloudv1alpha1.VPC{
							CIDR: &cidr,
						},
					},
				}
				configYAML = ExpectEncode(runtime.Encode(serializer, &config))
				secretNamespace = "secretns"
				secretName = "secret"
				region = "region"
				infra = extensionsv1alpha1.Infrastructure{
					Spec: extensionsv1alpha1.InfrastructureSpec{
						DefaultSpec: extensionsv1alpha1.DefaultSpec{
							ProviderConfig: &runtime.RawExtension{
								Raw: configYAML,
							},
						},
						Region: region,
						SecretRef: corev1.SecretReference{
							Namespace: secretNamespace,
							Name:      secretName,
						},
					},
				}
				accessKeyID = "accessKeyID"
				accessKeySecret = "accessKeySecret"
				cluster = controller.Cluster{
					Shoot: &gardencorev1beta1.Shoot{
						Spec: gardencorev1beta1.ShootSpec{
							Region: region,
						},
					},
				}

				initializerValues = InitializerValues{}
				chartValues = map[string]interface{}{}

				mainContent = "main"
				variablesContent = "variables"
				tfVarsContent = "tfVars"

				vpcID = "vpcID"
				vpcCIDRString = "vpcCIDR"
				natGatewayID = "natGatewayID"
				securityGroupID = "sgID"
				keyPairName = "keyPairName"
			})

			It("should correctly reconcile the infrastructure", func() {
				rawState = &realterraformer.RawState{
					Data:     "",
					Encoding: "none",
				}
				describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
				describeNATGatewaysReq.VpcId = vpcID

				gomock.InOrder(
					chartRendererFactory.EXPECT().NewForConfig(&restConfig).Return(chartRenderer, nil),

					c.EXPECT().Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, gomock.AssignableToTypeOf(&corev1.Secret{})).
						SetArg(2, corev1.Secret{
							Data: map[string][]byte{
								alicloud.AccessKeyID:     []byte(accessKeyID),
								alicloud.AccessKeySecret: []byte(accessKeySecret),
							},
						}),

					terraformerFactory.EXPECT().NewForConfig(gomock.Any(), &restConfig, TerraformerPurpose, infra.Namespace, infra.Name, imagevector.TerraformerImage()).
						Return(terraformer, nil),

					terraformer.EXPECT().SetTerminationGracePeriodSeconds(int64(630)).Return(terraformer),
					terraformer.EXPECT().SetDeadlineCleaning(5*time.Minute).Return(terraformer),
					terraformer.EXPECT().SetDeadlinePod(15*time.Minute).Return(terraformer),

					terraformer.EXPECT().SetVariablesEnvironment(map[string]string{
						common.TerraformVarAccessKeyID:     accessKeyID,
						common.TerraformVarAccessKeySecret: accessKeySecret,
					}).Return(terraformer),

					alicloudClientFactory.EXPECT().NewVPCClient(region, accessKeyID, accessKeySecret).Return(vpcClient, nil),

					terraformer.EXPECT().GetStateOutputVariables(TerraformerOutputKeyVPCID).
						Return(map[string]string{
							TerraformerOutputKeyVPCID: vpcID,
						}, nil),

					vpcClient.EXPECT().DescribeNatGateways(describeNATGatewaysReq).Return(&vpc.DescribeNatGatewaysResponse{
						NatGateways: vpc.NatGateways{
							NatGateway: []vpc.NatGateway{
								{
									NatGatewayId: natGatewayID,
								},
							},
						},
					}, nil),

					terraformChartOps.EXPECT().ComputeCreateVPCInitializerValues(&config, alicloudclient.DefaultInternetChargeType).Return(&initializerValues),
					terraformChartOps.EXPECT().ComputeChartValues(&infra, &config, &initializerValues).Return(chartValues),

					chartRenderer.EXPECT().Render(
						alicloud.InfraChartPath,
						alicloud.InfraRelease,
						infra.Namespace,
						chartValues,
					).Return(&chartrenderer.RenderedChart{
						Manifests: []manifest.Manifest{
							mkManifest(realterraformer.MainKey, mainContent),
							mkManifest(realterraformer.VariablesKey, variablesContent),
							mkManifest(realterraformer.TFVarsKey, tfVarsContent),
						},
					}, nil),

					terraformerFactory.EXPECT().DefaultInitializer(c, mainContent, variablesContent, []byte(tfVarsContent), gomock.AssignableToTypeOf(realterraformer.CreateState)).Return(initializer),

					terraformer.EXPECT().InitializeWith(initializer).Return(terraformer),

					terraformer.EXPECT().Apply(),

					c.EXPECT().Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, gomock.AssignableToTypeOf(&corev1.Secret{})).
						SetArg(2, corev1.Secret{
							Data: map[string][]byte{
								alicloud.AccessKeyID:     []byte(accessKeyID),
								alicloud.AccessKeySecret: []byte(accessKeySecret),
							},
						}),
					logger.EXPECT().Info("Creating Alicloud ECS client for Shoot", "infrastructure", infra.Name),
					alicloudClientFactory.EXPECT().NewECSClient(region, accessKeyID, accessKeySecret).Return(shootECSClient, nil),
					logger.EXPECT().Info("Creating Alicloud STS client for Shoot", "infrastructure", infra.Name),
					alicloudClientFactory.EXPECT().NewSTSClient(region, accessKeyID, accessKeySecret).Return(shootSTSClient, nil),
					shootSTSClient.EXPECT().GetAccountIDFromCallerIdentity(ctx).Return("", nil),
					logger.EXPECT().Info("Sharing customized image with Shoot's Alicloud account from Seed", "infrastructure", infra.Name),

					terraformer.EXPECT().GetStateOutputVariables(TerraformerOutputKeyVPCID, TerraformerOutputKeyVPCCIDR, TerraformerOutputKeySecurityGroupID, TerraformerOutputKeyKeyPairName).
						Return(map[string]string{
							TerraformerOutputKeyVPCID:           vpcID,
							TerraformerOutputKeyVPCCIDR:         vpcCIDRString,
							TerraformerOutputKeySecurityGroupID: securityGroupID,
							TerraformerOutputKeyKeyPairName:     keyPairName,
						}, nil),
					terraformer.EXPECT().GetRawState(ctx).Return(rawState, nil),
					c.EXPECT().Status().Return(c),
					c.EXPECT().Get(ctx, client.ObjectKey{Namespace: infra.Namespace, Name: infra.Name}, &infra),

					c.EXPECT().Update(ctx, &infra),
				)

				ExpectInject(inject.ClientInto(c, actuator))
				ExpectInject(inject.SchemeInto(scheme, actuator))
				ExpectInject(inject.ConfigInto(&restConfig, actuator))

				Expect(actuator.Reconcile(ctx, &infra, &cluster)).To(Succeed())
				Expect(infra.Status.ProviderStatus.Object).To(Equal(&alicloudv1alpha1.InfrastructureStatus{
					TypeMeta: StatusTypeMeta,
					VPC: alicloudv1alpha1.VPCStatus{
						ID: vpcID,
						SecurityGroups: []alicloudv1alpha1.SecurityGroup{
							{
								Purpose: alicloudv1alpha1.PurposeNodes,
								ID:      securityGroupID,
							},
						},
					},
					KeyPairName: keyPairName,
				}))
			})

			It("should correctly restore the infrastructure", func() {
				state := "some data"
				rawState = &realterraformer.RawState{
					Data:     "c29tZSBkYXRh",
					Encoding: "base64",
				}
				rawStateInBytes, _ := rawState.Marshal()
				infra.Status.State = &runtime.RawExtension{
					Raw: rawStateInBytes,
				}
				describeNATGatewaysReq := vpc.CreateDescribeNatGatewaysRequest()
				describeNATGatewaysReq.VpcId = vpcID

				gomock.InOrder(
					chartRendererFactory.EXPECT().NewForConfig(&restConfig).Return(chartRenderer, nil),

					c.EXPECT().Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, gomock.AssignableToTypeOf(&corev1.Secret{})).
						SetArg(2, corev1.Secret{
							Data: map[string][]byte{
								alicloud.AccessKeyID:     []byte(accessKeyID),
								alicloud.AccessKeySecret: []byte(accessKeySecret),
							},
						}),

					terraformerFactory.EXPECT().NewForConfig(gomock.Any(), &restConfig, TerraformerPurpose, infra.Namespace, infra.Name, imagevector.TerraformerImage()).
						Return(terraformer, nil),

					terraformer.EXPECT().SetTerminationGracePeriodSeconds(int64(630)).Return(terraformer),
					terraformer.EXPECT().SetDeadlineCleaning(5*time.Minute).Return(terraformer),
					terraformer.EXPECT().SetDeadlinePod(15*time.Minute).Return(terraformer),

					terraformer.EXPECT().SetVariablesEnvironment(map[string]string{
						common.TerraformVarAccessKeyID:     accessKeyID,
						common.TerraformVarAccessKeySecret: accessKeySecret,
					}).Return(terraformer),

					alicloudClientFactory.EXPECT().NewVPCClient(region, accessKeyID, accessKeySecret).Return(vpcClient, nil),

					terraformer.EXPECT().GetStateOutputVariables(TerraformerOutputKeyVPCID).
						Return(map[string]string{
							TerraformerOutputKeyVPCID: vpcID,
						}, nil),

					vpcClient.EXPECT().DescribeNatGateways(describeNATGatewaysReq).Return(&vpc.DescribeNatGatewaysResponse{
						NatGateways: vpc.NatGateways{
							NatGateway: []vpc.NatGateway{
								{
									NatGatewayId: natGatewayID,
								},
							},
						},
					}, nil),

					terraformChartOps.EXPECT().ComputeCreateVPCInitializerValues(&config, alicloudclient.DefaultInternetChargeType).Return(&initializerValues),
					terraformChartOps.EXPECT().ComputeChartValues(&infra, &config, &initializerValues).Return(chartValues),

					chartRenderer.EXPECT().Render(
						alicloud.InfraChartPath,
						alicloud.InfraRelease,
						infra.Namespace,
						chartValues,
					).Return(&chartrenderer.RenderedChart{
						Manifests: []manifest.Manifest{
							mkManifest(realterraformer.MainKey, mainContent),
							mkManifest(realterraformer.VariablesKey, variablesContent),
							mkManifest(realterraformer.TFVarsKey, tfVarsContent),
						},
					}, nil),

					terraformerFactory.EXPECT().DefaultInitializer(c, mainContent, variablesContent, []byte(tfVarsContent), realterraformer.CreateOrUpdateState{State: &state}).Return(initializer),

					terraformer.EXPECT().InitializeWith(initializer).Return(terraformer),

					terraformer.EXPECT().Apply(),

					c.EXPECT().Get(ctx, client.ObjectKey{Namespace: secretNamespace, Name: secretName}, gomock.AssignableToTypeOf(&corev1.Secret{})).
						SetArg(2, corev1.Secret{
							Data: map[string][]byte{
								alicloud.AccessKeyID:     []byte(accessKeyID),
								alicloud.AccessKeySecret: []byte(accessKeySecret),
							},
						}),
					logger.EXPECT().Info("Creating Alicloud ECS client for Shoot", "infrastructure", infra.Name),
					alicloudClientFactory.EXPECT().NewECSClient(region, accessKeyID, accessKeySecret).Return(shootECSClient, nil),
					logger.EXPECT().Info("Creating Alicloud STS client for Shoot", "infrastructure", infra.Name),
					alicloudClientFactory.EXPECT().NewSTSClient(region, accessKeyID, accessKeySecret).Return(shootSTSClient, nil),
					shootSTSClient.EXPECT().GetAccountIDFromCallerIdentity(ctx).Return("", nil),
					logger.EXPECT().Info("Sharing customized image with Shoot's Alicloud account from Seed", "infrastructure", infra.Name),

					terraformer.EXPECT().GetStateOutputVariables(TerraformerOutputKeyVPCID, TerraformerOutputKeyVPCCIDR, TerraformerOutputKeySecurityGroupID, TerraformerOutputKeyKeyPairName).
						Return(map[string]string{
							TerraformerOutputKeyVPCID:           vpcID,
							TerraformerOutputKeyVPCCIDR:         vpcCIDRString,
							TerraformerOutputKeySecurityGroupID: securityGroupID,
							TerraformerOutputKeyKeyPairName:     keyPairName,
						}, nil),
					terraformer.EXPECT().GetRawState(ctx).Return(rawState, nil),
					c.EXPECT().Status().Return(c),
					c.EXPECT().Get(ctx, client.ObjectKey{Namespace: infra.Namespace, Name: infra.Name}, &infra),

					c.EXPECT().Update(ctx, &infra),
				)

				ExpectInject(inject.ClientInto(c, actuator))
				ExpectInject(inject.SchemeInto(scheme, actuator))
				ExpectInject(inject.ConfigInto(&restConfig, actuator))

				Expect(actuator.Restore(ctx, &infra, &cluster)).To(Succeed())
				Expect(infra.Status.ProviderStatus.Object).To(Equal(&alicloudv1alpha1.InfrastructureStatus{
					TypeMeta: StatusTypeMeta,
					VPC: alicloudv1alpha1.VPCStatus{
						ID: vpcID,
						SecurityGroups: []alicloudv1alpha1.SecurityGroup{
							{
								Purpose: alicloudv1alpha1.PurposeNodes,
								ID:      securityGroupID,
							},
						},
					},
					KeyPairName: keyPairName,
				}))
			})
		})
	})
})
