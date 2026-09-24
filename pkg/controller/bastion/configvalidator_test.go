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
	testutils "github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gstruct "github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

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
	credentialsFile = "credentialsFile"
	region          = "region"
	id              = "id"
)

var _ = Describe("ConfigValidator", func() {
	var (
		ctrl                  *gomock.Controller
		mgr                   *testutils.FakeManager
		alicloudClientFactory *mockalicloudclient.MockClientFactory
		ecsClient             *mockalicloudclient.MockECS
		vpcClient             *mockalicloudclient.MockVPC
		ctx                   context.Context
		worker                *extensionsv1alpha1.Worker
		cv                    bastion.ConfigValidator
		bastion               *extensionsv1alpha1.Bastion
		cluster               *extensions.Cluster
		secret                *corev1.Secret
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		defer ctrl.Finish()
		alicloudClientFactory = mockalicloudclient.NewMockClientFactory(ctrl)
		ecsClient = mockalicloudclient.NewMockECS(ctrl)
		vpcClient = mockalicloudclient.NewMockVPC(ctrl)
		ctx = context.TODO()

		bastion = &extensionsv1alpha1.Bastion{}
		cluster = &extensions.Cluster{}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				alicloud.AccessKeyID:     []byte(accessKeyID),
				alicloud.AccessKeySecret: []byte(accessKeySecret),
				alicloud.CredentialsFile: []byte(credentialsFile),
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
			}},
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

			worker.Name = cluster.Shoot.Name
			worker.Namespace = cluster.ObjectMeta.Name

			cpSecret := secret.DeepCopy()
			cpSecret.Name = v1beta1constants.SecretNameCloudProvider
			cpSecret.Namespace = cluster.ObjectMeta.Name

			scheme := runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
			fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(worker, cpSecret).Build()
			mgr = &testutils.FakeManager{Client: fakeClient}
			cv = NewConfigValidator(mgr, alicloudClientFactory)
			alicloudClientFactory.EXPECT().NewECSClientFromAuthConfig(region, gomock.Any(), gomock.Any()).Return(ecsClient, nil)
			alicloudClientFactory.EXPECT().NewVPCClientFromAuthConfig(region, gomock.Any(), gomock.Any()).Return(vpcClient, nil)
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
