// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	mockaliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient/mock"
)

const (
	name      = "infrastructure"
	namespace = "shoot--foobar--alicloud"
	region    = "eu-central-1"
	vpcID     = "vpc-123456"
	eipID     = "eip-123456"

	accessKeyID     = "accessKeyID"
	secretAccessKey = "secretAccessKey"
	credentialsFile = "credentialsFile"
)

var _ = Describe("ConfigValidator", func() {
	var (
		ctrl        *gomock.Controller
		c           *mockclient.MockClient
		ctx         context.Context
		logger      logr.Logger
		actor       *mockaliclient.MockActor
		actorFactor *mockaliclient.MockFactory
		cv          infrastructure.ConfigValidator
		infra       *extensionsv1alpha1.Infrastructure
		secret      *corev1.Secret

		mgr *mockmanager.MockManager
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
		ctx = context.TODO()
		logger = log.Log.WithName("test")
		mgr = mockmanager.NewMockManager(ctrl)

		actorFactor = mockaliclient.NewMockFactory(ctrl)
		actor = mockaliclient.NewMockActor(ctrl)
		cv = NewConfigValidator(mgr, logger, actorFactor)

		infra = &extensionsv1alpha1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: extensionsv1alpha1.InfrastructureSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type: alicloud.Type,
					ProviderConfig: &runtime.RawExtension{
						Raw: encode(&apisalicloud.InfrastructureConfig{
							Networks: apisalicloud.Networks{
								VPC: apisalicloud.VPC{
									ID: ptr.To(vpcID),
								},
							},
						}),
					},
				},
				Region: region,
				SecretRef: corev1.SecretReference{
					Name:      name,
					Namespace: namespace,
				},
			},
		}
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(accessKeyID),
				alicloud.AccessKeySecret: []byte(secretAccessKey),
				alicloud.CredentialsFile: []byte(credentialsFile),
			},
		}

		mgr.EXPECT().GetClient().Return(c)
		c.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(
			func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
				*obj = *secret
				return nil
			},
		)
		actorFactor.EXPECT().NewActor(accessKeyID, secretAccessKey, region).Return(actor, nil)

	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should forbid when provide vpc id but call get VPC failed", func() {
		actor.EXPECT().GetVpc(ctx, vpcID).Return(nil, fmt.Errorf("not found"))
		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeInternal),
			"Field": Equal("networks.vpc.id"),
		}))
	})

	It("should forbid when provide vpc id but VPC doesn't exist", func() {
		actor.EXPECT().GetVpc(ctx, vpcID).Return(nil, nil)
		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("networks.vpc.id"),
		}))
	})

	It("should forbid when provide vpc id and gardenerManagedNATGateway is false or null but call get natgateway failed", func() {
		actor.EXPECT().GetVpc(ctx, vpcID).Return(&aliclient.VPC{}, nil)
		actor.EXPECT().FindNatGatewayByVPC(ctx, vpcID).Return(nil, fmt.Errorf("not found"))

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeInternal),
			"Field": Equal("networks.vpc.id"),
		}))
	})

	It("should forbid when provide vpc id and gardenerManagedNATGateway is false or null but natgateway doesn't exist", func() {
		actor.EXPECT().GetVpc(ctx, vpcID).Return(&aliclient.VPC{}, nil)
		actor.EXPECT().FindNatGatewayByVPC(ctx, vpcID).Return(nil, nil)

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":   Equal(field.ErrorTypeInvalid),
			"Field":  Equal("networks.vpc.id"),
			"Detail": Equal("no user natgateway found"),
		}))
	})

	It("should succeed when provide vpc exist and natgateway exist", func() {
		actor.EXPECT().GetVpc(ctx, vpcID).Return(&aliclient.VPC{}, nil)
		actor.EXPECT().FindNatGatewayByVPC(ctx, vpcID).Return(&aliclient.NatGateway{}, nil)

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(BeEmpty())

	})

	It("should forbid when provide eip id but call get EIP failed", func() {
		infra.Spec.ProviderConfig.Raw = encode(&apisalicloud.InfrastructureConfig{
			Networks: apisalicloud.Networks{
				VPC: apisalicloud.VPC{},
				Zones: []apisalicloud.Zone{
					{
						Name: "zone_1",
						NatGateway: &apisalicloud.NatGatewayConfig{
							EIPAllocationID: ptr.To(eipID),
						},
					},
				},
			},
		})
		actor.EXPECT().ListEnhanhcedNatGatewayAvailableZones(ctx, region).Return([]string{
			"zone_1",
			"zone_2",
		}, nil)
		actor.EXPECT().GetEIP(ctx, eipID).Return(nil, fmt.Errorf("not found"))

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeInternal),
			"Field": Equal("networks.zones[].natGateway.eipAllocationID"),
		}))
	})

	It("should forbid when provide eip id but EIP doesn't exist", func() {
		infra.Spec.ProviderConfig.Raw = encode(&apisalicloud.InfrastructureConfig{
			Networks: apisalicloud.Networks{
				VPC: apisalicloud.VPC{},
				Zones: []apisalicloud.Zone{
					{
						Name: "zone_1",
						NatGateway: &apisalicloud.NatGatewayConfig{
							EIPAllocationID: ptr.To(eipID),
						},
					},
				},
			},
		})
		actor.EXPECT().ListEnhanhcedNatGatewayAvailableZones(ctx, region).Return([]string{
			"zone_1",
			"zone_2",
		}, nil)
		actor.EXPECT().GetEIP(ctx, eipID).Return(nil, nil)

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("networks.zones[].natGateway.eipAllocationID"),
		}))
	})

	It("should forbid when provide eip id duplicated", func() {
		infra.Spec.ProviderConfig.Raw = encode(&apisalicloud.InfrastructureConfig{
			Networks: apisalicloud.Networks{
				VPC: apisalicloud.VPC{},
				Zones: []apisalicloud.Zone{
					{
						Name: "zone_1",
						NatGateway: &apisalicloud.NatGatewayConfig{
							EIPAllocationID: ptr.To(eipID),
						},
					},
					{
						Name: "zone_2",
						NatGateway: &apisalicloud.NatGatewayConfig{
							EIPAllocationID: ptr.To(eipID),
						},
					},
				},
			},
		})
		actor.EXPECT().ListEnhanhcedNatGatewayAvailableZones(ctx, region).Return([]string{
			"zone_1",
			"zone_2",
		}, nil)
		actor.EXPECT().GetEIP(ctx, eipID).Return(&aliclient.EIP{}, nil)

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeForbidden),
			"Field": Equal("networks.zones[].natGateway.eipAllocationID"),
		}))
	})

	It("should forbid when the zone is not EnhanhcedNatGateway available zone", func() {
		infra.Spec.ProviderConfig.Raw = encode(&apisalicloud.InfrastructureConfig{
			Networks: apisalicloud.Networks{
				VPC: apisalicloud.VPC{},
				Zones: []apisalicloud.Zone{
					{
						Name: "zone_invalid",
					},
				},
			},
		})
		actor.EXPECT().ListEnhanhcedNatGatewayAvailableZones(ctx, region).Return([]string{
			"zone_1",
			"zone_2",
		}, nil)

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(ConsistOfFields(Fields{
			"Type":  Equal(field.ErrorTypeForbidden),
			"Field": Equal("networks.zones[0].name"),
		}))
	})

	It("should succeed when provide EIP exist and natgateway zone valid", func() {
		infra.Spec.ProviderConfig.Raw = encode(&apisalicloud.InfrastructureConfig{
			Networks: apisalicloud.Networks{
				VPC: apisalicloud.VPC{},
				Zones: []apisalicloud.Zone{
					{
						Name: "zone_1",
						NatGateway: &apisalicloud.NatGatewayConfig{
							EIPAllocationID: ptr.To(eipID),
						},
					},
				},
			},
		})
		actor.EXPECT().ListEnhanhcedNatGatewayAvailableZones(ctx, region).Return([]string{
			"zone_1",
			"zone_2",
		}, nil)
		actor.EXPECT().GetEIP(ctx, eipID).Return(&aliclient.EIP{}, nil)

		errorList := cv.Validate(ctx, infra)
		Expect(errorList).To(BeEmpty())
	})

})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
