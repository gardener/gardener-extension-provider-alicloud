// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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
	"github.com/gardener/gardener/pkg/apis/security"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
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
		apiReader *mockclient.MockReader
		mockActor *mockaliclient.MockActor
		ctx       context.Context

		providerSecret *corev1.Secret
		baseShoot      *core.Shoot
		s              *shoot
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ctx = context.TODO()

		apiReader = mockclient.NewMockReader(ctrl)
		mockActor = mockaliclient.NewMockActor(ctrl)

		providerSecret = &corev1.Secret{
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

		s = &shoot{
			apiReader: apiReader,
			newActorFn: func(ak, sk, region string) (aliclient.Actor, error) {
				return mockActor, nil
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	expectSecretBindingLookup := func() {
		secretBinding := &core.SecretBinding{
			SecretRef: corev1.SecretReference{Namespace: shootNamespace, Name: secretName},
		}
		apiReader.EXPECT().
			Get(ctx, client.ObjectKey{Namespace: shootNamespace, Name: bindingName}, gomock.AssignableToTypeOf(&core.SecretBinding{})).
			DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *core.SecretBinding, _ ...client.GetOption) error {
				*obj = *secretBinding
				return nil
			})
		apiReader.EXPECT().
			Get(ctx, client.ObjectKey{Namespace: shootNamespace, Name: secretName}, gomock.AssignableToTypeOf(&corev1.Secret{})).
			DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
				*obj = *providerSecret
				return nil
			})
	}

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
			expectSecretBindingLookup()
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return nil when zone CIDR does not overlap any existing vswitch", func() {
			expectSecretBindingLookup()
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-other", CidrBlock: "192.168.2.0/24"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return field.Invalid when zone CIDR exactly matches an existing vswitch", func() {
			expectSecretBindingLookup()
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
			expectSecretBindingLookup()
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-large", CidrBlock: "192.168.0.0/16"},
			}, nil)

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicts with existing vswitch vsw-large"))
		})

		It("should use startIndex to produce correct field path for newly added zones", func() {
			expectSecretBindingLookup()
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return([]*aliclient.VSwitch{
				{VSwitchId: "vsw-b", CidrBlock: "192.168.2.0/24"},
			}, nil)

			// zones[1] is newly added (startIndex=1)
			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.2.0/24"), 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("networks.zones[1].workers"))
		})

		It("should return InternalError when FindVSwitchesByVPC fails", func() {
			expectSecretBindingLookup()
			mockActor.EXPECT().FindVSwitchesByVPC(ctx, testVPCID).Return(nil, fmt.Errorf("api error"))

			err := s.validateVSwitchCIDRConflict(ctx, baseShoot, testVPCID, zones("192.168.1.0/24"), 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not list vswitches"))
		})

		It("should skip zones with empty CIDR", func() {
			expectSecretBindingLookup()
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

			credentialsBinding := &security.CredentialsBinding{
				CredentialsRef: corev1.ObjectReference{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
					Namespace:  shootNamespace,
					Name:       secretName,
				},
			}
			apiReader.EXPECT().
				Get(ctx, client.ObjectKey{Namespace: shootNamespace, Name: bindingName}, gomock.AssignableToTypeOf(&security.CredentialsBinding{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *security.CredentialsBinding, _ ...client.GetOption) error {
					*obj = *credentialsBinding
					return nil
				})
			apiReader.EXPECT().
				Get(ctx, client.ObjectKey{Namespace: shootNamespace, Name: secretName}, gomock.AssignableToTypeOf(&corev1.Secret{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					*obj = *providerSecret
					return nil
				})
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
