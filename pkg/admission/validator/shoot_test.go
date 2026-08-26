// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

// Tests for validateVSwitchCIDRConflict and getCredentials are in package validator (white-box)
// so they can call unexported methods directly, avoiding the need to satisfy all static validation
// constraints that ValidateWorkers/ValidateInfrastructureConfig impose on a full Shoot.

package validator

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	provideralicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	mockaliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient/mock"
)

// fakeReader implements client.Reader by storing objects keyed by namespace/name.
type fakeReader struct {
	objects map[string]client.Object
}

func newFakeReader(objs ...client.Object) *fakeReader {
	r := &fakeReader{objects: make(map[string]client.Object)}
	for _, o := range objs {
		key := o.GetNamespace() + "/" + o.GetName()
		r.objects[key] = o
	}
	return r
}

func (f *fakeReader) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	stored, ok := f.objects[key.Namespace+"/"+key.Name]
	if !ok {
		return fmt.Errorf("object %s/%s not found", key.Namespace, key.Name)
	}
	switch t := obj.(type) {
	case *core.SecretBinding:
		*t = *stored.(*core.SecretBinding)
	case *securityv1alpha1.CredentialsBinding:
		*t = *stored.(*securityv1alpha1.CredentialsBinding)
	case *corev1.Secret:
		*t = *stored.(*corev1.Secret)
	default:
		return fmt.Errorf("unsupported type %T", obj)
	}
	return nil
}

func (f *fakeReader) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

var _ = Describe("shoot.validateVSwitchCIDRConflict", func() {
	const (
		shootNamespace = "shoot--project--test"
		shootRegion    = "cn-hangzhou"
		testVPCID      = "vpc-abc123"
		secretName     = "my-provider-secret"
		bindingName    = "my-binding"
		akID           = "AKID1234567890123456"
		akSecret       = "secretsecretsecretsecretsecretsecr"
	)

	var (
		ctrl      *gomock.Controller
		mockActor *mockaliclient.MockActor
		ctx       context.Context

		providerSecret *corev1.Secret
		baseShoot      *core.Shoot
		s              *shoot
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.TODO()

		mockActor = mockaliclient.NewMockActor(ctrl)

		providerSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: shootNamespace, Name: secretName},
			Data: map[string][]byte{
				provideralicloud.AccessKeyID:     []byte(akID),
				provideralicloud.AccessKeySecret: []byte(akSecret),
			},
		}

		baseShoot = &core.Shoot{
			ObjectMeta: metav1.ObjectMeta{Namespace: shootNamespace},
			Spec: core.ShootSpec{
				Region:            shootRegion,
				SecretBindingName: ptr.To(bindingName),
			},
		}

		secretBinding := &core.SecretBinding{
			ObjectMeta: metav1.ObjectMeta{Namespace: shootNamespace, Name: bindingName},
			SecretRef:  corev1.SecretReference{Namespace: shootNamespace, Name: secretName},
		}

		s = &shoot{
			apiReader: newFakeReader(secretBinding, providerSecret),
			newActorFn: func(_, _, _ string) (aliclient.Actor, error) {
				return mockActor, nil
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	zones := func(cidrs ...string) []apisalicloud.Zone {
		var zs []apisalicloud.Zone
		for i, c := range cidrs {
			zs = append(zs, apisalicloud.Zone{
				Name:    fmt.Sprintf("cn-hangzhou-%c", 'a'+i),
				Workers: c,
			})
		}
		return zs
	}

	Describe("CIDR conflict detection", func() {
		It("should return nil when no existing vswitches in VPC", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return nil when zone CIDR does not overlap any existing vswitch", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-other", CidrBlock: "192.168.2.0/24"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return field.Invalid when zone CIDR exactly matches an existing vswitch", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-existing", CidrBlock: "192.168.1.0/24"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("networks.zones[0].workers"))
			Expect(err.Error()).To(ContainSubstring("conflicts with existing vswitch vsw-existing"))
			Expect(err.Error()).To(ContainSubstring("192.168.1.0/24"))
		})

		It("should return field.Invalid when zone CIDR is a subset of an existing vswitch CIDR", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-large", CidrBlock: "192.168.0.0/16"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicts with existing vswitch vsw-large"))
		})

		It("should use startIndex to produce correct field path for newly added zones", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-b", CidrBlock: "192.168.2.0/24"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.2.0/24"), 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("networks.zones[1].workers"))
		})

		It("should return InternalError when FindVSwitchesByVPC fails", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return(nil, fmt.Errorf("api error"))

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not list vswitches"))
		})

		It("should skip zones with empty CIDR", func() {
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-x", CidrBlock: "192.168.1.0/24"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID,
				[]apisalicloud.Zone{{Name: "cn-hangzhou-a"}}, 0)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("getCredentials via CredentialsBinding", func() {
		It("should resolve credentials via CredentialsBinding when SecretBindingName is nil", func() {
			sh := baseShoot.DeepCopy()
			sh.Spec.SecretBindingName = nil
			sh.Spec.CredentialsBindingName = ptr.To(bindingName)

			credentialsBinding := &securityv1alpha1.CredentialsBinding{
				ObjectMeta: metav1.ObjectMeta{Namespace: shootNamespace, Name: bindingName},
				CredentialsRef: corev1.ObjectReference{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
					Namespace:  shootNamespace,
					Name:       secretName,
				},
			}
			s.apiReader = newFakeReader(credentialsBinding, providerSecret)
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, sh, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when neither SecretBindingName nor CredentialsBindingName is set", func() {
			sh := baseShoot.DeepCopy()
			sh.Spec.SecretBindingName = nil
			sh.Spec.CredentialsBindingName = nil

			err := s.validateVSwitchCIDRConflict(ctx, sh, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("neither secretBindingName nor credentialsBindingName"))
		})
	})
})
