// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
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
	case *securityv1alpha1.WorkloadIdentity:
		*t = *stored.(*securityv1alpha1.WorkloadIdentity)
	default:
		return fmt.Errorf("unsupported type %T", obj)
	}
	return nil
}

func (f *fakeReader) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

var _ = Describe("shoot.validateDNS", func() {
	const (
		wiName = "my-wi"
	)

	var (
		ctx context.Context
		sv  *shoot
	)

	// validWIProviderConfig is a v1alpha1 WorkloadIdentityConfig with valid ARNs.
	validWIProviderConfig := func() *runtime.RawExtension {
		return &runtime.RawExtension{
			Raw: []byte(`{"apiVersion":"alicloud.provider.extensions.gardener.cloud/v1alpha1","kind":"WorkloadIdentityConfig","oidcProviderARN":"acs:ram::123456789012:oidc-provider/test","roleARN":"acs:ram::123456789012:role/testrole"}`),
		}
	}

	invalidWIProviderConfig := func() *runtime.RawExtension {
		return &runtime.RawExtension{
			Raw: []byte(`{"apiVersion":"alicloud.provider.extensions.gardener.cloud/v1alpha1","kind":"WorkloadIdentityConfig","oidcProviderARN":"","roleARN":""}`),
		}
	}

	dnsShoot := func(providers []core.DNSProvider) *core.Shoot {
		return &core.Shoot{
			Spec: core.ShootSpec{
				DNS: &core.DNS{Providers: providers},
			},
		}
	}

	BeforeEach(func() {
		ctx = context.TODO()
		sv = &shoot{apiReader: newFakeReader()}
	})

	It("should return no errors when DNS is nil", func() {
		sh := &core.Shoot{}
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).To(BeEmpty())
	})

	It("should return no errors for non-alicloud-dns provider", func() {
		sh := dnsShoot([]core.DNSProvider{
			{Type: ptr.To("aws-route53"), CredentialsRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Secret", Name: "some-secret",
			}},
		})
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).To(BeEmpty())
	})

	It("should return no errors for alicloud-dns with Secret credentialsRef", func() {
		sh := dnsShoot([]core.DNSProvider{
			{Type: ptr.To("alicloud-dns"), CredentialsRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Secret", Name: "dns-secret",
			}},
		})
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).To(BeEmpty())
	})

	It("should return no errors for alicloud-dns with valid WorkloadIdentity credentialsRef", func() {
		wi := &securityv1alpha1.WorkloadIdentity{
			ObjectMeta: metav1.ObjectMeta{Name: wiName},
			Spec: securityv1alpha1.WorkloadIdentitySpec{
				TargetSystem: securityv1alpha1.TargetSystem{ProviderConfig: validWIProviderConfig()},
			},
		}
		sv.apiReader = newFakeReader(wi)
		sh := dnsShoot([]core.DNSProvider{
			{Type: ptr.To("alicloud-dns"), CredentialsRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: securityv1alpha1.SchemeGroupVersion.String(), Kind: "WorkloadIdentity", Name: wiName,
			}},
		})
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).To(BeEmpty())
	})

	It("should return errors for alicloud-dns with invalid WorkloadIdentity config", func() {
		wi := &securityv1alpha1.WorkloadIdentity{
			ObjectMeta: metav1.ObjectMeta{Name: wiName},
			Spec: securityv1alpha1.WorkloadIdentitySpec{
				TargetSystem: securityv1alpha1.TargetSystem{ProviderConfig: invalidWIProviderConfig()},
			},
		}
		sv.apiReader = newFakeReader(wi)
		sh := dnsShoot([]core.DNSProvider{
			{Type: ptr.To("alicloud-dns"), CredentialsRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: securityv1alpha1.SchemeGroupVersion.String(), Kind: "WorkloadIdentity", Name: wiName,
			}},
		})
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).NotTo(BeEmpty())
	})

	It("should return InternalError when WorkloadIdentity is not found", func() {
		sh := dnsShoot([]core.DNSProvider{
			{Type: ptr.To("alicloud-dns"), CredentialsRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: securityv1alpha1.SchemeGroupVersion.String(), Kind: "WorkloadIdentity", Name: "nonexistent",
			}},
		})
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeInternal))
	})

	It("should return NotSupported for unsupported credentialsRef kind", func() {
		sh := dnsShoot([]core.DNSProvider{
			{Type: ptr.To("alicloud-dns"), CredentialsRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: "example.io/v1", Kind: "ConfigMap", Name: "bad-ref",
			}},
		})
		errs := sv.validateDNS(ctx, sh)
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeNotSupported))
	})
})

var _ = Describe("shoot WorkloadIdentity CSI validation", func() {
	const (
		cbName    = "my-cb"
		shootNS   = "garden-test"
	)

	var (
		ctx context.Context
		sv  *shoot
	)

	BeforeEach(func() {
		ctx = context.TODO()
	})

	wiCredentialsBinding := func() *securityv1alpha1.CredentialsBinding {
		return &securityv1alpha1.CredentialsBinding{
			ObjectMeta: metav1.ObjectMeta{Name: cbName, Namespace: shootNS},
			CredentialsRef: corev1.ObjectReference{Kind: "WorkloadIdentity", Name: "my-wi"},
		}
	}

	akCredentialsBinding := func() *securityv1alpha1.CredentialsBinding {
		return &securityv1alpha1.CredentialsBinding{
			ObjectMeta: metav1.ObjectMeta{Name: cbName, Namespace: shootNS},
			CredentialsRef: corev1.ObjectReference{Kind: "Secret", Name: "my-secret"},
		}
	}

	wiShoot := func() *core.Shoot {
		return &core.Shoot{
			ObjectMeta: metav1.ObjectMeta{Namespace: shootNS},
			Spec: core.ShootSpec{
				CredentialsBindingName: ptr.To(cbName),
			},
		}
	}

	It("should pass when shoot uses WorkloadIdentity and enableLabeler is true", func() {
		sv = &shoot{apiReader: newFakeReader(wiCredentialsBinding())}
		cpConfig := &apisalicloud.ControlPlaneConfig{
			CSI: &apisalicloud.CSI{
				EnableLabeler: ptr.To(true),
			},
		}
		err := sv.validateShoot(ctx, wiShoot(), nil, cpConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail when shoot uses WorkloadIdentity and enableLabeler is false", func() {
		sv = &shoot{apiReader: newFakeReader(wiCredentialsBinding())}
		cpConfig := &apisalicloud.ControlPlaneConfig{
			CSI: &apisalicloud.CSI{
				EnableLabeler: ptr.To(false),
			},
		}
		err := sv.validateShoot(ctx, wiShoot(), nil, cpConfig)
		Expect(err).To(HaveOccurred())
	})

	It("should fail when shoot uses WorkloadIdentity and cpConfig is nil", func() {
		sv = &shoot{apiReader: newFakeReader(wiCredentialsBinding())}
		err := sv.validateShoot(ctx, wiShoot(), nil, nil)
		Expect(err).To(HaveOccurred())
	})

	It("should pass when shoot uses AK/SK (not WorkloadIdentity) regardless of enableLabeler", func() {
		sv = &shoot{apiReader: newFakeReader(akCredentialsBinding())}
		cpConfig := &apisalicloud.ControlPlaneConfig{}
		err := sv.validateShoot(ctx, wiShoot(), nil, cpConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should pass when shoot has no credentialsBindingName", func() {
		sv = &shoot{apiReader: newFakeReader()}
		sh := &core.Shoot{
			ObjectMeta: metav1.ObjectMeta{Namespace: shootNS},
			Spec:       core.ShootSpec{},
		}
		err := sv.validateShoot(ctx, sh, nil, nil)
		Expect(err).NotTo(HaveOccurred())
	})
})
