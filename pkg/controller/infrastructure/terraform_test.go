// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure_test

import (
	"strconv"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/dualstack"
)

var _ = Describe("TerraformChartOps", func() {
	var (
		ctrl *gomock.Controller
		ops  = DefaultTerraformOps()
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#ComputeCreateVPCInitializerValues", func() {
		It("should compute the values from the config", func() {
			var (
				cidr               = "192.168.0.0/16"
				internetChargeType = "foo"
				config             = v1alpha1.InfrastructureConfig{
					Networks: v1alpha1.Networks{
						VPC: v1alpha1.VPC{
							CIDR: &cidr,
						},
					},
				}
			)

			Expect(ops.ComputeCreateVPCInitializerValues(&config, internetChargeType)).To(Equal(&InitializerValues{
				VPC: VPC{
					CreateVPC: true,
					VPCID:     TerraformDefaultVPCID,
					VPCCIDR:   cidr,
				},
				NATGateway: NATGateway{
					CreateNATGateway: true,
					NATGatewayID:     TerraformDefaultNATGatewayID,
					SNATTableIDs:     TerraformDefaultSNATTableIDs,
				},
				EIP: EIP{
					InternetChargeType: internetChargeType,
				},
			}))
		})
	})

	Describe("#ComputeUseVPCInitializerValues", func() {
		It("should compute the values from the infra and config", func() {
			var (
				id           = "id"
				cidr         = "192.168.0.0/16"
				natGatewayID = "natGatewayID"
				sNATTableIDs = "sNATTableIDs"
				info         = alicloudclient.VPCInfo{
					CIDR:         cidr,
					NATGatewayID: natGatewayID,
					SNATTableIDs: sNATTableIDs,
				}
				config = v1alpha1.InfrastructureConfig{
					Networks: v1alpha1.Networks{
						VPC: v1alpha1.VPC{
							ID: &id,
						},
					},
				}
			)

			Expect(ops.ComputeUseVPCInitializerValues(&config, &info)).To(Equal(&InitializerValues{
				VPC: VPC{
					CreateVPC: false,
					VPCID:     strconv.Quote(id),
					VPCCIDR:   cidr,
				},
				NATGateway: NATGateway{
					NATGatewayID: strconv.Quote(natGatewayID),
					SNATTableIDs: strconv.Quote(sNATTableIDs),
				},
			}))
		})
	})

	Describe("#ComputeTerraformerChartValues", func() {
		It("should compute the terraformer chart values", func() {
			var (
				namespace = "cluster-foo"
				region    = "region"

				infra = extensionsv1alpha1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
					},
					Spec: extensionsv1alpha1.InfrastructureSpec{
						Region: region,
					},
				}

				zone1Name       = "zone1"
				zone1Worker     = "192.168.0.0/16"
				zone1EipAllocID = "eip-ufxsdckfgitzcz"
				zone1NatGateway = v1alpha1.NatGatewayConfig{
					EIPAllocationID: &zone1EipAllocID,
				}

				zone2Name   = "zone2"
				zone2Worker = "192.169.0.0/16"

				config = v1alpha1.InfrastructureConfig{
					Networks: v1alpha1.Networks{
						Zones: []v1alpha1.Zone{
							{
								Name:       zone1Name,
								Workers:    zone1Worker,
								NatGateway: &zone1NatGateway,
							},
							{
								Name:    zone2Name,
								Workers: zone2Worker,
							},
						},
					},
				}

				vpcCIDR            = "192.170.0.0/16"
				vpcID              = "vpcID"
				natGatewayID       = "natGatewayID"
				sNATTableIDs       = "sNATTableIDs"
				internetChargeType = "internetChargeType"
				podCIDR            = "100.96.0.0/11"
				Zone_A             = "Zone_A"
				Zone_A_CIDR        = "192.170.254.0/24"
				Zone_A_IPV6_SUBNET = 254
				Zone_B             = "Zone_B"
				Zone_B_CIDR        = "192.170.255.0/24"
				Zone_B_IPV6_SUBNET = 255

				values = InitializerValues{
					VPC: VPC{
						CreateVPC: true,
						VPCCIDR:   vpcCIDR,
						VPCID:     vpcID,
					},
					NATGateway: NATGateway{
						CreateNATGateway: true,
						NATGatewayID:     natGatewayID,
						SNATTableIDs:     sNATTableIDs,
					},
					EIP: EIP{
						InternetChargeType: internetChargeType,
					},
					DualStack: dualstack.DualStack{
						Enabled:            true,
						Zone_A:             Zone_A,
						Zone_A_CIDR:        Zone_A_CIDR,
						Zone_A_IPV6_SUBNET: Zone_A_IPV6_SUBNET,
						Zone_B:             Zone_B,
						Zone_B_CIDR:        Zone_B_CIDR,
						Zone_B_IPV6_SUBNET: Zone_B_IPV6_SUBNET,
					},
				}
			)

			Expect(ops.ComputeChartValues(&infra, &config, &podCIDR, &values)).To(Equal(map[string]interface{}{
				"alicloud": map[string]interface{}{
					"region": region,
				},
				"vpc": map[string]interface{}{
					"create": true,
					"cidr":   vpcCIDR,
					"id":     vpcID,
				},
				"natGateway": map[string]interface{}{
					"create":       true,
					"id":           natGatewayID,
					"sNatTableIDs": sNATTableIDs,
				},
				"eip": map[string]interface{}{
					"internetChargeType": internetChargeType,
				},
				"clusterName": namespace,
				"zones": []map[string]interface{}{
					{
						"name": zone1Name,
						"cidr": map[string]interface{}{
							"workers": zone1Worker,
						},
						"eipAllocationID": zone1EipAllocID,
					},
					{
						"name": zone2Name,
						"cidr": map[string]interface{}{
							"workers": zone2Worker,
						},
					},
				},
				"podCIDR": podCIDR,
				"outputKeys": map[string]interface{}{
					"vpcID":              TerraformerOutputKeyVPCID,
					"vpcCIDR":            TerraformerOutputKeyVPCCIDR,
					"securityGroupID":    TerraformerOutputKeySecurityGroupID,
					"vswitchNodesPrefix": TerraformerOutputKeyVSwitchNodesPrefix,
				},
				"dualStack": map[string]interface{}{
					"enabled":            true,
					"zone_a":             Zone_A,
					"zone_a_cidr":        Zone_A_CIDR,
					"zone_a_ipv6_subnet": Zone_A_IPV6_SUBNET,
					"zone_b":             Zone_B,
					"zone_b_cidr":        Zone_B_CIDR,
					"zone_b_ipv6_subnet": Zone_B_IPV6_SUBNET,
				},
			}))
		})
	})
})
