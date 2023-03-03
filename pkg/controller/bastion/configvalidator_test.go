package bastion

import (
	"context"
	"encoding/json"

	ecs "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gstruct "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	aliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	apialicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	mockalicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/mock/provider-alicloud/alicloud/client"
)

const (
	name            = "foo"
	namespace       = "shoot--foobar--alicloud"
	accessKeyID     = "accessKeyID"
	accessKeySecret = "accessKeySecret"
	region          = "region"
	id              = "id"
)

var _ = Describe("ConfigValidator", func() {
	var (
		ctrl                  *gomock.Controller
		c                     *mockclient.MockClient
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		ecsClient             *mockalicloudclient.MockECS
		vpcClient             *mockalicloudclient.MockVPC
		ctx                   context.Context
		worker                *extensionsv1alpha1.Worker
		cv                    bastion.ConfigValidator
		bastion               *extensionsv1alpha1.Bastion
		cluster               *extensions.Cluster
		cloudProfile          *corev1beta1.CloudProfile
		secretBinding         *corev1beta1.SecretBinding
		secret                *corev1.Secret
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		defer ctrl.Finish()
		c = mockclient.NewMockClient(ctrl)
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		ecsClient = mockalicloudclient.NewMockECS(ctrl)
		vpcClient = mockalicloudclient.NewMockVPC(ctrl)
		ctx = context.TODO()

		cv = NewConfigValidator(alicloudClientFactory)
		err := cv.(inject.Client).InjectClient(c)
		Expect(err).NotTo(HaveOccurred())

		bastion = &extensionsv1alpha1.Bastion{}
		cluster = &extensions.Cluster{}

		secretBinding = &corev1beta1.SecretBinding{
			SecretRef: corev1.SecretReference{
				Name:      name,
				Namespace: namespace,
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
				alicloud.AccessKeySecret: []byte(accessKeySecret),
			},
		}

		infraStatus := &apialicloud.InfrastructureStatus{
			VPC: apialicloud.VPCStatus{
				ID: id,
				VSwitches: []apialicloud.VSwitch{
					{
						ID:   id,
						Zone: "zone",
					},
				},
				SecurityGroups: []apialicloud.SecurityGroup{
					{ID: id},
				},
			},
			MachineImages: []apialicloud.MachineImage{{
				ID: id,
			},
			},
		}

		worker = &extensionsv1alpha1.Worker{
			Spec: extensionsv1alpha1.WorkerSpec{
				InfrastructureProviderStatus: &runtime.RawExtension{
					Raw: encode(infraStatus),
				},
			},
		}

	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Validate", func() {
		BeforeEach(func() {
			cluster = createClusters()
			key := client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}

			c.EXPECT().Get(ctx, key, gomock.AssignableToTypeOf(&extensionsv1alpha1.Worker{})).DoAndReturn(
				func(_ context.Context, namespacedName client.ObjectKey, obj *extensionsv1alpha1.Worker, _ ...client.GetOption) error {
					worker.DeepCopyInto(obj)
					return nil
				})
			c.EXPECT().Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: v1beta1constants.SecretNameCloudProvider}, gomock.AssignableToTypeOf(&corev1beta1.CloudProfile{})).DoAndReturn(clientGet(cloudProfile))
			c.EXPECT().Get(ctx, key, gomock.AssignableToTypeOf(&corev1beta1.SecretBinding{})).DoAndReturn(clientGet(secretBinding))
			c.EXPECT().Get(ctx, key, gomock.AssignableToTypeOf(&corev1.Secret{})).DoAndReturn(clientGet(secret))
			alicloudClientFactory.EXPECT().NewECSClient(region, accessKeyID, accessKeySecret).Return(ecsClient, nil)
			alicloudClientFactory.EXPECT().NewVPCClient(region, accessKeyID, accessKeySecret).Return(vpcClient, nil)
		})

		It("should succeed if there are infrastructureStatus passed", func() {
			vpcClient.EXPECT().GetVPCWithID(ctx, id).Return([]vpc.Vpc{{VpcId: id}}, nil)
			vpcClient.EXPECT().GetVSwitchesInfoByID(id).Return(&aliclient.VSwitchInfo{ZoneID: "zoneid"}, nil)
			ecsClient.EXPECT().CheckIfImageExists(id).Return(true, nil)
			ecsClient.EXPECT().GetSecurityGroupWithID(id).Return(&ecs.DescribeSecurityGroupsResponse{
				SecurityGroups: ecs.SecurityGroups{
					SecurityGroup: []ecs.SecurityGroup{
						{SecurityGroupId: id},
					},
				},
			}, nil)
			errorList := cv.Validate(ctx, bastion, cluster)
			Expect(errorList).To(BeEmpty())
		})

		It("should fail with InternalError if getting vpc failed", func() {
			vpcClient.EXPECT().GetVPCWithID(ctx, id).Return(nil, nil)
			errorList := cv.Validate(ctx, bastion, cluster)
			Expect(errorList).To(ConsistOfFields(
				gstruct.Fields{
					"Type":   Equal(field.ErrorTypeInternal),
					"Field":  Equal("vpc"),
					"Detail": Equal("could not get vpc id from alicloud provider: %!w(<nil>)"),
				}))
		})

		It("should fail with InternalError if getting vSwitch failed", func() {
			vpcClient.EXPECT().GetVPCWithID(ctx, id).Return([]vpc.Vpc{{VpcId: id}}, nil)
			vpcClient.EXPECT().GetVSwitchesInfoByID(id).Return(&aliclient.VSwitchInfo{ZoneID: ""}, nil)
			errorList := cv.Validate(ctx, bastion, cluster)
			Expect(errorList).To(ConsistOfFields(
				gstruct.Fields{
					"Type":   Equal(field.ErrorTypeInternal),
					"Field":  Equal("vswitches"),
					"Detail": Equal("could not get vswitches id from alicloud provider: %!w(<nil>)"),
				}))
		})

		It("should fail with InternalError if getting machineImages id failed", func() {
			vpcClient.EXPECT().GetVPCWithID(ctx, id).Return([]vpc.Vpc{{VpcId: id}}, nil)
			vpcClient.EXPECT().GetVSwitchesInfoByID(id).Return(&aliclient.VSwitchInfo{ZoneID: "zoneid"}, nil)
			ecsClient.EXPECT().CheckIfImageExists(id).Return(false, nil)
			errorList := cv.Validate(ctx, bastion, cluster)
			Expect(errorList).To(ConsistOfFields(
				gstruct.Fields{
					"Type":   Equal(field.ErrorTypeInternal),
					"Field":  Equal("machineImages"),
					"Detail": Equal("could not get machineImages id from alicloud provider: %!w(<nil>)"),
				}))
		})

		It("should fail with InternalError if getting securityGroup id failed", func() {
			vpcClient.EXPECT().GetVPCWithID(ctx, id).Return([]vpc.Vpc{{VpcId: id}}, nil)
			vpcClient.EXPECT().GetVSwitchesInfoByID(id).Return(&aliclient.VSwitchInfo{ZoneID: "zoneid"}, nil)
			ecsClient.EXPECT().CheckIfImageExists(id).Return(true, nil)
			ecsClient.EXPECT().GetSecurityGroupWithID(id).Return(&ecs.DescribeSecurityGroupsResponse{
				SecurityGroups: ecs.SecurityGroups{
					SecurityGroup: []ecs.SecurityGroup{
						{SecurityGroupId: ""},
					},
				},
			}, nil)
			errorList := cv.Validate(ctx, bastion, cluster)
			Expect(errorList).To(ConsistOfFields(
				gstruct.Fields{
					"Type":   Equal(field.ErrorTypeInternal),
					"Field":  Equal("securityGroup"),
					"Detail": Equal("could not get shoot security group id from alicloud provider: %!w(<nil>)"),
				}))
		})

	})

})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

func createClusters() *extensions.Cluster {
	return &extensions.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Shoot: &corev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name: v1beta1constants.SecretNameCloudProvider,
			},
			Spec: corev1beta1.ShootSpec{
				Region: region,
			},
		},
	}
}

func clientGet(result client.Object) interface{} {
	return func(ctx context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
		switch obj.(type) {
		case *corev1.Secret:
			*obj.(*corev1.Secret) = *result.(*corev1.Secret)
		case *corev1beta1.CloudProfile:
			*obj.(*corev1beta1.CloudProfile) = *result.(*corev1beta1.CloudProfile)
		case *corev1beta1.SecretBinding:
			*obj.(*corev1beta1.SecretBinding) = *result.(*corev1beta1.SecretBinding)
		}
		return nil
	}
}
