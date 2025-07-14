// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

const (
	// Name is the name of the Alicloud provider.
	Name = "provider-alicloud"

	// MachineControllerManagerProviderAlicloudImageName is the name of the MachineControllerManagerProviderAlicloud image.
	MachineControllerManagerProviderAlicloudImageName = "machine-controller-manager-provider-alicloud"
	// CloudControllerManagerImageName is the name of the CloudControllerManager image.
	CloudControllerManagerImageName = "alicloud-controller-manager"
	// CSIAttacherImageName is the name of the CSI attacher image.
	CSIAttacherImageName = "csi-attacher"
	// CSIDiskTopologyZoneKey is the CSI topology label name that represents availability by zone.
	// See https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/v1.2.1/pkg/disk/disk.go#L45C1-L45C1
	// See https://www.alibabacloud.com/help/en/elastic-container-instance/latest/mount-a-disk-as-a-statically-provisioned-volume
	CSIDiskTopologyZoneKey = "topology.diskplugin.csi.alibabacloud.com/zone"
	// CSINodeDriverRegistrarImageName is the name of the CSI driver registrar image.
	CSINodeDriverRegistrarImageName = "csi-node-driver-registrar"
	// CSIProvisionerImageName is the name of the CSI provisioner image.
	CSIProvisionerImageName = "csi-provisioner"
	// CSISnapshotterImageName is the name of the CSI snapshotter image.
	CSISnapshotterImageName = "csi-snapshotter"
	// CSISnapshotControllerImageName is the name of the CSI snapshot controller image.
	CSISnapshotControllerImageName = "csi-snapshot-controller"
	// CSIResizerImageName is the name of the CSI resizer image.
	CSIResizerImageName = "csi-resizer"
	// CSILivenessProbeImageName is the name of the CSI liveness probe image.
	CSILivenessProbeImageName = "csi-liveness-probe"

	// CSIPluginImageName is the name of the CSI plugin image.
	CSIPluginImageName = "csi-plugin-alicloud"

	// CSIPluginInitImageName is the name of the CSI plugin init image.
	CSIPluginInitImageName = "csi-plugin-alicloud-init"

	// StorageEndpoint is the data field in a secret where the storage endpoint is stored at.
	StorageEndpoint = "storageEndpoint"
	// CloudControllerManagerName is the a constant for the name of the CloudController.
	CloudControllerManagerName = "cloud-controller-manager"
	// CSIPluginController is the a constant for the name of the csi-plugin-controller Deployment in the Seed.
	CSIPluginController = "csi-plugin-controller"
	// CSISnapshotControllerName is a constant for the name of the csi-snapshot-controller Deployment in the Seed.
	CSISnapshotControllerName = "csi-snapshot-controller"
	// CSISnapshotValidationName is the constant for the name of the csi-snapshot-validation-webhook component.
	CSISnapshotValidationName = "csi-snapshot-validation"

	// CRDVolumeSnapshotClasses is a constant for the name of VolumeSnapshotClasses CRD.
	CRDVolumeSnapshotClasses = "volumesnapshotclasses.snapshot.storage.k8s.io"
	// CRDVolumeSnapshotContents is a constant for the name of VolumeSnapshotContents CRD.
	CRDVolumeSnapshotContents = "volumesnapshotcontents.snapshot.storage.k8s.io"
	// CRDVolumeSnapshots is a constant for the name of CRDVolumeSnapshots CRD.
	CRDVolumeSnapshots = "volumesnapshots.snapshot.storage.k8s.io"

	// ServiceLinkedRoleForNATGateway is a constant for the name of service linked role of NAT gateway.
	ServiceLinkedRoleForNATGateway = "AliyunServiceRoleForNatgw"
	// ServiceForNATGateway is a constant for the name of service of NAT gateway.
	ServiceForNATGateway = "nat.aliyuncs.com"

	// ErrorCodeNoPermission is a constant for the error code of no permission.
	ErrorCodeNoPermission = "NoPermission"
	// ErrorCodeRoleEntityNotExist is a constant for the error code of role entity not exist.
	ErrorCodeRoleEntityNotExist = "EntityNotExist.Role"
	// ErrorCodeDomainRecordNotBelongToUser is a constant for the error code of domain record not belong to user.
	ErrorCodeDomainRecordNotBelongToUser = "DomainRecordNotBelongToUser"

	// DefaultDNSRegion is the default region to be used if a region is not specified in the DNS secret
	// or in the DNSRecord resource.
	DefaultDNSRegion = "cn-shanghai"
	// CloudProviderConfigName is the name of the configmap containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"
)

var (
	// UsernamePrefix is a constant for the username prefix of components deployed by AWS.
	UsernamePrefix = extensionsv1alpha1.SchemeGroupVersion.Group + ":" + Name + ":"
)
