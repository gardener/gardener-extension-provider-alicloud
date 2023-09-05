//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	alicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*CSI)(nil), (*alicloud.CSI)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CSI_To_alicloud_CSI(a.(*CSI), b.(*alicloud.CSI), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.CSI)(nil), (*CSI)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_CSI_To_v1alpha1_CSI(a.(*alicloud.CSI), b.(*CSI), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*CloudControllerManagerConfig)(nil), (*alicloud.CloudControllerManagerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CloudControllerManagerConfig_To_alicloud_CloudControllerManagerConfig(a.(*CloudControllerManagerConfig), b.(*alicloud.CloudControllerManagerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.CloudControllerManagerConfig)(nil), (*CloudControllerManagerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(a.(*alicloud.CloudControllerManagerConfig), b.(*CloudControllerManagerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*CloudProfileConfig)(nil), (*alicloud.CloudProfileConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CloudProfileConfig_To_alicloud_CloudProfileConfig(a.(*CloudProfileConfig), b.(*alicloud.CloudProfileConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.CloudProfileConfig)(nil), (*CloudProfileConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(a.(*alicloud.CloudProfileConfig), b.(*CloudProfileConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ControlPlaneConfig)(nil), (*alicloud.ControlPlaneConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ControlPlaneConfig_To_alicloud_ControlPlaneConfig(a.(*ControlPlaneConfig), b.(*alicloud.ControlPlaneConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.ControlPlaneConfig)(nil), (*ControlPlaneConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(a.(*alicloud.ControlPlaneConfig), b.(*ControlPlaneConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*InfrastructureConfig)(nil), (*alicloud.InfrastructureConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_InfrastructureConfig_To_alicloud_InfrastructureConfig(a.(*InfrastructureConfig), b.(*alicloud.InfrastructureConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.InfrastructureConfig)(nil), (*InfrastructureConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(a.(*alicloud.InfrastructureConfig), b.(*InfrastructureConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*InfrastructureStatus)(nil), (*alicloud.InfrastructureStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_InfrastructureStatus_To_alicloud_InfrastructureStatus(a.(*InfrastructureStatus), b.(*alicloud.InfrastructureStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.InfrastructureStatus)(nil), (*InfrastructureStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(a.(*alicloud.InfrastructureStatus), b.(*InfrastructureStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MachineImage)(nil), (*alicloud.MachineImage)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MachineImage_To_alicloud_MachineImage(a.(*MachineImage), b.(*alicloud.MachineImage), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.MachineImage)(nil), (*MachineImage)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_MachineImage_To_v1alpha1_MachineImage(a.(*alicloud.MachineImage), b.(*MachineImage), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MachineImageVersion)(nil), (*alicloud.MachineImageVersion)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MachineImageVersion_To_alicloud_MachineImageVersion(a.(*MachineImageVersion), b.(*alicloud.MachineImageVersion), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.MachineImageVersion)(nil), (*MachineImageVersion)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_MachineImageVersion_To_v1alpha1_MachineImageVersion(a.(*alicloud.MachineImageVersion), b.(*MachineImageVersion), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MachineImages)(nil), (*alicloud.MachineImages)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MachineImages_To_alicloud_MachineImages(a.(*MachineImages), b.(*alicloud.MachineImages), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.MachineImages)(nil), (*MachineImages)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_MachineImages_To_v1alpha1_MachineImages(a.(*alicloud.MachineImages), b.(*MachineImages), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NatGatewayConfig)(nil), (*alicloud.NatGatewayConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_NatGatewayConfig_To_alicloud_NatGatewayConfig(a.(*NatGatewayConfig), b.(*alicloud.NatGatewayConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.NatGatewayConfig)(nil), (*NatGatewayConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_NatGatewayConfig_To_v1alpha1_NatGatewayConfig(a.(*alicloud.NatGatewayConfig), b.(*NatGatewayConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Networks)(nil), (*alicloud.Networks)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Networks_To_alicloud_Networks(a.(*Networks), b.(*alicloud.Networks), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.Networks)(nil), (*Networks)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_Networks_To_v1alpha1_Networks(a.(*alicloud.Networks), b.(*Networks), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RegionIDMapping)(nil), (*alicloud.RegionIDMapping)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RegionIDMapping_To_alicloud_RegionIDMapping(a.(*RegionIDMapping), b.(*alicloud.RegionIDMapping), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.RegionIDMapping)(nil), (*RegionIDMapping)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_RegionIDMapping_To_v1alpha1_RegionIDMapping(a.(*alicloud.RegionIDMapping), b.(*RegionIDMapping), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*SecurityGroup)(nil), (*alicloud.SecurityGroup)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_SecurityGroup_To_alicloud_SecurityGroup(a.(*SecurityGroup), b.(*alicloud.SecurityGroup), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.SecurityGroup)(nil), (*SecurityGroup)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_SecurityGroup_To_v1alpha1_SecurityGroup(a.(*alicloud.SecurityGroup), b.(*SecurityGroup), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*VPC)(nil), (*alicloud.VPC)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_VPC_To_alicloud_VPC(a.(*VPC), b.(*alicloud.VPC), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.VPC)(nil), (*VPC)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_VPC_To_v1alpha1_VPC(a.(*alicloud.VPC), b.(*VPC), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*VPCStatus)(nil), (*alicloud.VPCStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_VPCStatus_To_alicloud_VPCStatus(a.(*VPCStatus), b.(*alicloud.VPCStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.VPCStatus)(nil), (*VPCStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_VPCStatus_To_v1alpha1_VPCStatus(a.(*alicloud.VPCStatus), b.(*VPCStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*VSwitch)(nil), (*alicloud.VSwitch)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_VSwitch_To_alicloud_VSwitch(a.(*VSwitch), b.(*alicloud.VSwitch), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.VSwitch)(nil), (*VSwitch)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_VSwitch_To_v1alpha1_VSwitch(a.(*alicloud.VSwitch), b.(*VSwitch), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*WorkerStatus)(nil), (*alicloud.WorkerStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_WorkerStatus_To_alicloud_WorkerStatus(a.(*WorkerStatus), b.(*alicloud.WorkerStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.WorkerStatus)(nil), (*WorkerStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_WorkerStatus_To_v1alpha1_WorkerStatus(a.(*alicloud.WorkerStatus), b.(*WorkerStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Zone)(nil), (*alicloud.Zone)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Zone_To_alicloud_Zone(a.(*Zone), b.(*alicloud.Zone), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*alicloud.Zone)(nil), (*Zone)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_alicloud_Zone_To_v1alpha1_Zone(a.(*alicloud.Zone), b.(*Zone), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_CSI_To_alicloud_CSI(in *CSI, out *alicloud.CSI, s conversion.Scope) error {
	out.EnableADController = (*bool)(unsafe.Pointer(in.EnableADController))
	return nil
}

// Convert_v1alpha1_CSI_To_alicloud_CSI is an autogenerated conversion function.
func Convert_v1alpha1_CSI_To_alicloud_CSI(in *CSI, out *alicloud.CSI, s conversion.Scope) error {
	return autoConvert_v1alpha1_CSI_To_alicloud_CSI(in, out, s)
}

func autoConvert_alicloud_CSI_To_v1alpha1_CSI(in *alicloud.CSI, out *CSI, s conversion.Scope) error {
	out.EnableADController = (*bool)(unsafe.Pointer(in.EnableADController))
	return nil
}

// Convert_alicloud_CSI_To_v1alpha1_CSI is an autogenerated conversion function.
func Convert_alicloud_CSI_To_v1alpha1_CSI(in *alicloud.CSI, out *CSI, s conversion.Scope) error {
	return autoConvert_alicloud_CSI_To_v1alpha1_CSI(in, out, s)
}

func autoConvert_v1alpha1_CloudControllerManagerConfig_To_alicloud_CloudControllerManagerConfig(in *CloudControllerManagerConfig, out *alicloud.CloudControllerManagerConfig, s conversion.Scope) error {
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	return nil
}

// Convert_v1alpha1_CloudControllerManagerConfig_To_alicloud_CloudControllerManagerConfig is an autogenerated conversion function.
func Convert_v1alpha1_CloudControllerManagerConfig_To_alicloud_CloudControllerManagerConfig(in *CloudControllerManagerConfig, out *alicloud.CloudControllerManagerConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_CloudControllerManagerConfig_To_alicloud_CloudControllerManagerConfig(in, out, s)
}

func autoConvert_alicloud_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(in *alicloud.CloudControllerManagerConfig, out *CloudControllerManagerConfig, s conversion.Scope) error {
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	return nil
}

// Convert_alicloud_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig is an autogenerated conversion function.
func Convert_alicloud_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(in *alicloud.CloudControllerManagerConfig, out *CloudControllerManagerConfig, s conversion.Scope) error {
	return autoConvert_alicloud_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(in, out, s)
}

func autoConvert_v1alpha1_CloudProfileConfig_To_alicloud_CloudProfileConfig(in *CloudProfileConfig, out *alicloud.CloudProfileConfig, s conversion.Scope) error {
	out.MachineImages = *(*[]alicloud.MachineImages)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_v1alpha1_CloudProfileConfig_To_alicloud_CloudProfileConfig is an autogenerated conversion function.
func Convert_v1alpha1_CloudProfileConfig_To_alicloud_CloudProfileConfig(in *CloudProfileConfig, out *alicloud.CloudProfileConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_CloudProfileConfig_To_alicloud_CloudProfileConfig(in, out, s)
}

func autoConvert_alicloud_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(in *alicloud.CloudProfileConfig, out *CloudProfileConfig, s conversion.Scope) error {
	out.MachineImages = *(*[]MachineImages)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_alicloud_CloudProfileConfig_To_v1alpha1_CloudProfileConfig is an autogenerated conversion function.
func Convert_alicloud_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(in *alicloud.CloudProfileConfig, out *CloudProfileConfig, s conversion.Scope) error {
	return autoConvert_alicloud_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(in, out, s)
}

func autoConvert_v1alpha1_ControlPlaneConfig_To_alicloud_ControlPlaneConfig(in *ControlPlaneConfig, out *alicloud.ControlPlaneConfig, s conversion.Scope) error {
	out.CloudControllerManager = (*alicloud.CloudControllerManagerConfig)(unsafe.Pointer(in.CloudControllerManager))
	out.CSI = (*alicloud.CSI)(unsafe.Pointer(in.CSI))
	return nil
}

// Convert_v1alpha1_ControlPlaneConfig_To_alicloud_ControlPlaneConfig is an autogenerated conversion function.
func Convert_v1alpha1_ControlPlaneConfig_To_alicloud_ControlPlaneConfig(in *ControlPlaneConfig, out *alicloud.ControlPlaneConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_ControlPlaneConfig_To_alicloud_ControlPlaneConfig(in, out, s)
}

func autoConvert_alicloud_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(in *alicloud.ControlPlaneConfig, out *ControlPlaneConfig, s conversion.Scope) error {
	out.CloudControllerManager = (*CloudControllerManagerConfig)(unsafe.Pointer(in.CloudControllerManager))
	out.CSI = (*CSI)(unsafe.Pointer(in.CSI))
	return nil
}

// Convert_alicloud_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig is an autogenerated conversion function.
func Convert_alicloud_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(in *alicloud.ControlPlaneConfig, out *ControlPlaneConfig, s conversion.Scope) error {
	return autoConvert_alicloud_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(in, out, s)
}

func autoConvert_v1alpha1_InfrastructureConfig_To_alicloud_InfrastructureConfig(in *InfrastructureConfig, out *alicloud.InfrastructureConfig, s conversion.Scope) error {
	if err := Convert_v1alpha1_Networks_To_alicloud_Networks(&in.Networks, &out.Networks, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_InfrastructureConfig_To_alicloud_InfrastructureConfig is an autogenerated conversion function.
func Convert_v1alpha1_InfrastructureConfig_To_alicloud_InfrastructureConfig(in *InfrastructureConfig, out *alicloud.InfrastructureConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_InfrastructureConfig_To_alicloud_InfrastructureConfig(in, out, s)
}

func autoConvert_alicloud_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(in *alicloud.InfrastructureConfig, out *InfrastructureConfig, s conversion.Scope) error {
	if err := Convert_alicloud_Networks_To_v1alpha1_Networks(&in.Networks, &out.Networks, s); err != nil {
		return err
	}
	return nil
}

// Convert_alicloud_InfrastructureConfig_To_v1alpha1_InfrastructureConfig is an autogenerated conversion function.
func Convert_alicloud_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(in *alicloud.InfrastructureConfig, out *InfrastructureConfig, s conversion.Scope) error {
	return autoConvert_alicloud_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(in, out, s)
}

func autoConvert_v1alpha1_InfrastructureStatus_To_alicloud_InfrastructureStatus(in *InfrastructureStatus, out *alicloud.InfrastructureStatus, s conversion.Scope) error {
	if err := Convert_v1alpha1_VPCStatus_To_alicloud_VPCStatus(&in.VPC, &out.VPC, s); err != nil {
		return err
	}
	out.KeyPairName = in.KeyPairName
	out.MachineImages = *(*[]alicloud.MachineImage)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_v1alpha1_InfrastructureStatus_To_alicloud_InfrastructureStatus is an autogenerated conversion function.
func Convert_v1alpha1_InfrastructureStatus_To_alicloud_InfrastructureStatus(in *InfrastructureStatus, out *alicloud.InfrastructureStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_InfrastructureStatus_To_alicloud_InfrastructureStatus(in, out, s)
}

func autoConvert_alicloud_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(in *alicloud.InfrastructureStatus, out *InfrastructureStatus, s conversion.Scope) error {
	if err := Convert_alicloud_VPCStatus_To_v1alpha1_VPCStatus(&in.VPC, &out.VPC, s); err != nil {
		return err
	}
	out.KeyPairName = in.KeyPairName
	out.MachineImages = *(*[]MachineImage)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_alicloud_InfrastructureStatus_To_v1alpha1_InfrastructureStatus is an autogenerated conversion function.
func Convert_alicloud_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(in *alicloud.InfrastructureStatus, out *InfrastructureStatus, s conversion.Scope) error {
	return autoConvert_alicloud_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(in, out, s)
}

func autoConvert_v1alpha1_MachineImage_To_alicloud_MachineImage(in *MachineImage, out *alicloud.MachineImage, s conversion.Scope) error {
	out.Name = in.Name
	out.Version = in.Version
	out.ID = in.ID
	out.Encrypted = (*bool)(unsafe.Pointer(in.Encrypted))
	return nil
}

// Convert_v1alpha1_MachineImage_To_alicloud_MachineImage is an autogenerated conversion function.
func Convert_v1alpha1_MachineImage_To_alicloud_MachineImage(in *MachineImage, out *alicloud.MachineImage, s conversion.Scope) error {
	return autoConvert_v1alpha1_MachineImage_To_alicloud_MachineImage(in, out, s)
}

func autoConvert_alicloud_MachineImage_To_v1alpha1_MachineImage(in *alicloud.MachineImage, out *MachineImage, s conversion.Scope) error {
	out.Name = in.Name
	out.Version = in.Version
	out.ID = in.ID
	out.Encrypted = (*bool)(unsafe.Pointer(in.Encrypted))
	return nil
}

// Convert_alicloud_MachineImage_To_v1alpha1_MachineImage is an autogenerated conversion function.
func Convert_alicloud_MachineImage_To_v1alpha1_MachineImage(in *alicloud.MachineImage, out *MachineImage, s conversion.Scope) error {
	return autoConvert_alicloud_MachineImage_To_v1alpha1_MachineImage(in, out, s)
}

func autoConvert_v1alpha1_MachineImageVersion_To_alicloud_MachineImageVersion(in *MachineImageVersion, out *alicloud.MachineImageVersion, s conversion.Scope) error {
	out.Version = in.Version
	out.Regions = *(*[]alicloud.RegionIDMapping)(unsafe.Pointer(&in.Regions))
	return nil
}

// Convert_v1alpha1_MachineImageVersion_To_alicloud_MachineImageVersion is an autogenerated conversion function.
func Convert_v1alpha1_MachineImageVersion_To_alicloud_MachineImageVersion(in *MachineImageVersion, out *alicloud.MachineImageVersion, s conversion.Scope) error {
	return autoConvert_v1alpha1_MachineImageVersion_To_alicloud_MachineImageVersion(in, out, s)
}

func autoConvert_alicloud_MachineImageVersion_To_v1alpha1_MachineImageVersion(in *alicloud.MachineImageVersion, out *MachineImageVersion, s conversion.Scope) error {
	out.Version = in.Version
	out.Regions = *(*[]RegionIDMapping)(unsafe.Pointer(&in.Regions))
	return nil
}

// Convert_alicloud_MachineImageVersion_To_v1alpha1_MachineImageVersion is an autogenerated conversion function.
func Convert_alicloud_MachineImageVersion_To_v1alpha1_MachineImageVersion(in *alicloud.MachineImageVersion, out *MachineImageVersion, s conversion.Scope) error {
	return autoConvert_alicloud_MachineImageVersion_To_v1alpha1_MachineImageVersion(in, out, s)
}

func autoConvert_v1alpha1_MachineImages_To_alicloud_MachineImages(in *MachineImages, out *alicloud.MachineImages, s conversion.Scope) error {
	out.Name = in.Name
	out.Versions = *(*[]alicloud.MachineImageVersion)(unsafe.Pointer(&in.Versions))
	return nil
}

// Convert_v1alpha1_MachineImages_To_alicloud_MachineImages is an autogenerated conversion function.
func Convert_v1alpha1_MachineImages_To_alicloud_MachineImages(in *MachineImages, out *alicloud.MachineImages, s conversion.Scope) error {
	return autoConvert_v1alpha1_MachineImages_To_alicloud_MachineImages(in, out, s)
}

func autoConvert_alicloud_MachineImages_To_v1alpha1_MachineImages(in *alicloud.MachineImages, out *MachineImages, s conversion.Scope) error {
	out.Name = in.Name
	out.Versions = *(*[]MachineImageVersion)(unsafe.Pointer(&in.Versions))
	return nil
}

// Convert_alicloud_MachineImages_To_v1alpha1_MachineImages is an autogenerated conversion function.
func Convert_alicloud_MachineImages_To_v1alpha1_MachineImages(in *alicloud.MachineImages, out *MachineImages, s conversion.Scope) error {
	return autoConvert_alicloud_MachineImages_To_v1alpha1_MachineImages(in, out, s)
}

func autoConvert_v1alpha1_NatGatewayConfig_To_alicloud_NatGatewayConfig(in *NatGatewayConfig, out *alicloud.NatGatewayConfig, s conversion.Scope) error {
	out.EIPAllocationID = (*string)(unsafe.Pointer(in.EIPAllocationID))
	return nil
}

// Convert_v1alpha1_NatGatewayConfig_To_alicloud_NatGatewayConfig is an autogenerated conversion function.
func Convert_v1alpha1_NatGatewayConfig_To_alicloud_NatGatewayConfig(in *NatGatewayConfig, out *alicloud.NatGatewayConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_NatGatewayConfig_To_alicloud_NatGatewayConfig(in, out, s)
}

func autoConvert_alicloud_NatGatewayConfig_To_v1alpha1_NatGatewayConfig(in *alicloud.NatGatewayConfig, out *NatGatewayConfig, s conversion.Scope) error {
	out.EIPAllocationID = (*string)(unsafe.Pointer(in.EIPAllocationID))
	return nil
}

// Convert_alicloud_NatGatewayConfig_To_v1alpha1_NatGatewayConfig is an autogenerated conversion function.
func Convert_alicloud_NatGatewayConfig_To_v1alpha1_NatGatewayConfig(in *alicloud.NatGatewayConfig, out *NatGatewayConfig, s conversion.Scope) error {
	return autoConvert_alicloud_NatGatewayConfig_To_v1alpha1_NatGatewayConfig(in, out, s)
}

func autoConvert_v1alpha1_Networks_To_alicloud_Networks(in *Networks, out *alicloud.Networks, s conversion.Scope) error {
	if err := Convert_v1alpha1_VPC_To_alicloud_VPC(&in.VPC, &out.VPC, s); err != nil {
		return err
	}
	out.Zones = *(*[]alicloud.Zone)(unsafe.Pointer(&in.Zones))
	return nil
}

// Convert_v1alpha1_Networks_To_alicloud_Networks is an autogenerated conversion function.
func Convert_v1alpha1_Networks_To_alicloud_Networks(in *Networks, out *alicloud.Networks, s conversion.Scope) error {
	return autoConvert_v1alpha1_Networks_To_alicloud_Networks(in, out, s)
}

func autoConvert_alicloud_Networks_To_v1alpha1_Networks(in *alicloud.Networks, out *Networks, s conversion.Scope) error {
	if err := Convert_alicloud_VPC_To_v1alpha1_VPC(&in.VPC, &out.VPC, s); err != nil {
		return err
	}
	out.Zones = *(*[]Zone)(unsafe.Pointer(&in.Zones))
	return nil
}

// Convert_alicloud_Networks_To_v1alpha1_Networks is an autogenerated conversion function.
func Convert_alicloud_Networks_To_v1alpha1_Networks(in *alicloud.Networks, out *Networks, s conversion.Scope) error {
	return autoConvert_alicloud_Networks_To_v1alpha1_Networks(in, out, s)
}

func autoConvert_v1alpha1_RegionIDMapping_To_alicloud_RegionIDMapping(in *RegionIDMapping, out *alicloud.RegionIDMapping, s conversion.Scope) error {
	out.Name = in.Name
	out.ID = in.ID
	return nil
}

// Convert_v1alpha1_RegionIDMapping_To_alicloud_RegionIDMapping is an autogenerated conversion function.
func Convert_v1alpha1_RegionIDMapping_To_alicloud_RegionIDMapping(in *RegionIDMapping, out *alicloud.RegionIDMapping, s conversion.Scope) error {
	return autoConvert_v1alpha1_RegionIDMapping_To_alicloud_RegionIDMapping(in, out, s)
}

func autoConvert_alicloud_RegionIDMapping_To_v1alpha1_RegionIDMapping(in *alicloud.RegionIDMapping, out *RegionIDMapping, s conversion.Scope) error {
	out.Name = in.Name
	out.ID = in.ID
	return nil
}

// Convert_alicloud_RegionIDMapping_To_v1alpha1_RegionIDMapping is an autogenerated conversion function.
func Convert_alicloud_RegionIDMapping_To_v1alpha1_RegionIDMapping(in *alicloud.RegionIDMapping, out *RegionIDMapping, s conversion.Scope) error {
	return autoConvert_alicloud_RegionIDMapping_To_v1alpha1_RegionIDMapping(in, out, s)
}

func autoConvert_v1alpha1_SecurityGroup_To_alicloud_SecurityGroup(in *SecurityGroup, out *alicloud.SecurityGroup, s conversion.Scope) error {
	out.Purpose = alicloud.Purpose(in.Purpose)
	out.ID = in.ID
	return nil
}

// Convert_v1alpha1_SecurityGroup_To_alicloud_SecurityGroup is an autogenerated conversion function.
func Convert_v1alpha1_SecurityGroup_To_alicloud_SecurityGroup(in *SecurityGroup, out *alicloud.SecurityGroup, s conversion.Scope) error {
	return autoConvert_v1alpha1_SecurityGroup_To_alicloud_SecurityGroup(in, out, s)
}

func autoConvert_alicloud_SecurityGroup_To_v1alpha1_SecurityGroup(in *alicloud.SecurityGroup, out *SecurityGroup, s conversion.Scope) error {
	out.Purpose = Purpose(in.Purpose)
	out.ID = in.ID
	return nil
}

// Convert_alicloud_SecurityGroup_To_v1alpha1_SecurityGroup is an autogenerated conversion function.
func Convert_alicloud_SecurityGroup_To_v1alpha1_SecurityGroup(in *alicloud.SecurityGroup, out *SecurityGroup, s conversion.Scope) error {
	return autoConvert_alicloud_SecurityGroup_To_v1alpha1_SecurityGroup(in, out, s)
}

func autoConvert_v1alpha1_VPC_To_alicloud_VPC(in *VPC, out *alicloud.VPC, s conversion.Scope) error {
	out.ID = (*string)(unsafe.Pointer(in.ID))
	out.CIDR = (*string)(unsafe.Pointer(in.CIDR))
	out.Bandwidth = (*int)(unsafe.Pointer(in.Bandwidth))
	out.GardenerManagedNATGateway = (*bool)(unsafe.Pointer(in.GardenerManagedNATGateway))
	return nil
}

// Convert_v1alpha1_VPC_To_alicloud_VPC is an autogenerated conversion function.
func Convert_v1alpha1_VPC_To_alicloud_VPC(in *VPC, out *alicloud.VPC, s conversion.Scope) error {
	return autoConvert_v1alpha1_VPC_To_alicloud_VPC(in, out, s)
}

func autoConvert_alicloud_VPC_To_v1alpha1_VPC(in *alicloud.VPC, out *VPC, s conversion.Scope) error {
	out.ID = (*string)(unsafe.Pointer(in.ID))
	out.CIDR = (*string)(unsafe.Pointer(in.CIDR))
	out.Bandwidth = (*int)(unsafe.Pointer(in.Bandwidth))
	out.GardenerManagedNATGateway = (*bool)(unsafe.Pointer(in.GardenerManagedNATGateway))
	return nil
}

// Convert_alicloud_VPC_To_v1alpha1_VPC is an autogenerated conversion function.
func Convert_alicloud_VPC_To_v1alpha1_VPC(in *alicloud.VPC, out *VPC, s conversion.Scope) error {
	return autoConvert_alicloud_VPC_To_v1alpha1_VPC(in, out, s)
}

func autoConvert_v1alpha1_VPCStatus_To_alicloud_VPCStatus(in *VPCStatus, out *alicloud.VPCStatus, s conversion.Scope) error {
	out.ID = in.ID
	out.VSwitches = *(*[]alicloud.VSwitch)(unsafe.Pointer(&in.VSwitches))
	out.SecurityGroups = *(*[]alicloud.SecurityGroup)(unsafe.Pointer(&in.SecurityGroups))
	return nil
}

// Convert_v1alpha1_VPCStatus_To_alicloud_VPCStatus is an autogenerated conversion function.
func Convert_v1alpha1_VPCStatus_To_alicloud_VPCStatus(in *VPCStatus, out *alicloud.VPCStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_VPCStatus_To_alicloud_VPCStatus(in, out, s)
}

func autoConvert_alicloud_VPCStatus_To_v1alpha1_VPCStatus(in *alicloud.VPCStatus, out *VPCStatus, s conversion.Scope) error {
	out.ID = in.ID
	out.VSwitches = *(*[]VSwitch)(unsafe.Pointer(&in.VSwitches))
	out.SecurityGroups = *(*[]SecurityGroup)(unsafe.Pointer(&in.SecurityGroups))
	return nil
}

// Convert_alicloud_VPCStatus_To_v1alpha1_VPCStatus is an autogenerated conversion function.
func Convert_alicloud_VPCStatus_To_v1alpha1_VPCStatus(in *alicloud.VPCStatus, out *VPCStatus, s conversion.Scope) error {
	return autoConvert_alicloud_VPCStatus_To_v1alpha1_VPCStatus(in, out, s)
}

func autoConvert_v1alpha1_VSwitch_To_alicloud_VSwitch(in *VSwitch, out *alicloud.VSwitch, s conversion.Scope) error {
	out.Purpose = alicloud.Purpose(in.Purpose)
	out.ID = in.ID
	out.Zone = in.Zone
	return nil
}

// Convert_v1alpha1_VSwitch_To_alicloud_VSwitch is an autogenerated conversion function.
func Convert_v1alpha1_VSwitch_To_alicloud_VSwitch(in *VSwitch, out *alicloud.VSwitch, s conversion.Scope) error {
	return autoConvert_v1alpha1_VSwitch_To_alicloud_VSwitch(in, out, s)
}

func autoConvert_alicloud_VSwitch_To_v1alpha1_VSwitch(in *alicloud.VSwitch, out *VSwitch, s conversion.Scope) error {
	out.Purpose = Purpose(in.Purpose)
	out.ID = in.ID
	out.Zone = in.Zone
	return nil
}

// Convert_alicloud_VSwitch_To_v1alpha1_VSwitch is an autogenerated conversion function.
func Convert_alicloud_VSwitch_To_v1alpha1_VSwitch(in *alicloud.VSwitch, out *VSwitch, s conversion.Scope) error {
	return autoConvert_alicloud_VSwitch_To_v1alpha1_VSwitch(in, out, s)
}

func autoConvert_v1alpha1_WorkerStatus_To_alicloud_WorkerStatus(in *WorkerStatus, out *alicloud.WorkerStatus, s conversion.Scope) error {
	out.MachineImages = *(*[]alicloud.MachineImage)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_v1alpha1_WorkerStatus_To_alicloud_WorkerStatus is an autogenerated conversion function.
func Convert_v1alpha1_WorkerStatus_To_alicloud_WorkerStatus(in *WorkerStatus, out *alicloud.WorkerStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_WorkerStatus_To_alicloud_WorkerStatus(in, out, s)
}

func autoConvert_alicloud_WorkerStatus_To_v1alpha1_WorkerStatus(in *alicloud.WorkerStatus, out *WorkerStatus, s conversion.Scope) error {
	out.MachineImages = *(*[]MachineImage)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_alicloud_WorkerStatus_To_v1alpha1_WorkerStatus is an autogenerated conversion function.
func Convert_alicloud_WorkerStatus_To_v1alpha1_WorkerStatus(in *alicloud.WorkerStatus, out *WorkerStatus, s conversion.Scope) error {
	return autoConvert_alicloud_WorkerStatus_To_v1alpha1_WorkerStatus(in, out, s)
}

func autoConvert_v1alpha1_Zone_To_alicloud_Zone(in *Zone, out *alicloud.Zone, s conversion.Scope) error {
	out.Name = in.Name
	out.Worker = in.Worker
	out.Workers = in.Workers
	out.NatGateway = (*alicloud.NatGatewayConfig)(unsafe.Pointer(in.NatGateway))
	return nil
}

// Convert_v1alpha1_Zone_To_alicloud_Zone is an autogenerated conversion function.
func Convert_v1alpha1_Zone_To_alicloud_Zone(in *Zone, out *alicloud.Zone, s conversion.Scope) error {
	return autoConvert_v1alpha1_Zone_To_alicloud_Zone(in, out, s)
}

func autoConvert_alicloud_Zone_To_v1alpha1_Zone(in *alicloud.Zone, out *Zone, s conversion.Scope) error {
	out.Name = in.Name
	out.Worker = in.Worker
	out.Workers = in.Workers
	out.NatGateway = (*NatGatewayConfig)(unsafe.Pointer(in.NatGateway))
	return nil
}

// Convert_alicloud_Zone_To_v1alpha1_Zone is an autogenerated conversion function.
func Convert_alicloud_Zone_To_v1alpha1_Zone(in *alicloud.Zone, out *Zone, s conversion.Scope) error {
	return autoConvert_alicloud_Zone_To_v1alpha1_Zone(in, out, s)
}
