// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
)

var _ = Describe("ControlPlaneConfig validation", func() {
	var (
		controlPlane *apisalicloud.ControlPlaneConfig
		fldPath      *field.Path
	)

	BeforeEach(func() {
		controlPlane = &apisalicloud.ControlPlaneConfig{}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for a valid configuration", func() {
			Expect(ValidateControlPlaneConfig(controlPlane, "", fldPath)).To(BeEmpty())
		})

		It("should fail with invalid CCM feature gates", func() {
			controlPlane.CloudControllerManager = &apisalicloud.CloudControllerManagerConfig{
				FeatureGates: map[string]bool{
					"AnyVolumeDataSource": true,
					"Foo":                 true,
				},
			}

			errorList := ValidateControlPlaneConfig(controlPlane, "1.33.0", fldPath)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("cloudControllerManager.featureGates.Foo"),
				})),
			))
		})
	})

	Describe("#ValidateCSIForWorkloadIdentity", func() {
		It("should return no errors when enableLabeler is true and enableADController is nil (defaults to true)", func() {
			cp := &apisalicloud.ControlPlaneConfig{
				CSI: &apisalicloud.CSI{
					EnableLabeler: ptr.To(true),
				},
			}
			Expect(ValidateCSIForWorkloadIdentity(cp, fldPath)).To(BeEmpty())
		})

		It("should return no errors when both enableADController and enableLabeler are true", func() {
			cp := &apisalicloud.ControlPlaneConfig{
				CSI: &apisalicloud.CSI{
					EnableADController: ptr.To(true),
					EnableLabeler:      ptr.To(true),
				},
			}
			Expect(ValidateCSIForWorkloadIdentity(cp, fldPath)).To(BeEmpty())
		})

		It("should return an error when controlPlaneConfig is nil", func() {
			errorList := ValidateCSIForWorkloadIdentity(nil, fldPath)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("csi.enableLabeler"),
				})),
			))
		})

		It("should return an error when CSI is nil", func() {
			cp := &apisalicloud.ControlPlaneConfig{}
			errorList := ValidateCSIForWorkloadIdentity(cp, fldPath)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("csi.enableLabeler"),
				})),
			))
		})

		It("should return an error when enableLabeler is false", func() {
			cp := &apisalicloud.ControlPlaneConfig{
				CSI: &apisalicloud.CSI{
					EnableLabeler: ptr.To(false),
				},
			}
			errorList := ValidateCSIForWorkloadIdentity(cp, fldPath)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("csi.enableLabeler"),
				})),
			))
		})

		It("should return an error when enableLabeler is nil", func() {
			cp := &apisalicloud.ControlPlaneConfig{
				CSI: &apisalicloud.CSI{},
			}
			errorList := ValidateCSIForWorkloadIdentity(cp, fldPath)
			Expect(errorList).NotTo(BeEmpty())
		})

		It("should return an error when enableADController is explicitly false", func() {
			cp := &apisalicloud.ControlPlaneConfig{
				CSI: &apisalicloud.CSI{
					EnableADController: ptr.To(false),
					EnableLabeler:      ptr.To(true),
				},
			}
			errorList := ValidateCSIForWorkloadIdentity(cp, fldPath)
			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("csi.enableADController"),
				})),
			))
		})

		It("should return two errors when both enableADController is false and enableLabeler is not true", func() {
			cp := &apisalicloud.ControlPlaneConfig{
				CSI: &apisalicloud.CSI{
					EnableADController: ptr.To(false),
					EnableLabeler:      ptr.To(false),
				},
			}
			errorList := ValidateCSIForWorkloadIdentity(cp, fldPath)
			Expect(errorList).To(HaveLen(2))
		})
	})
})
