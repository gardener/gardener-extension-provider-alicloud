// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alicloud

import "path/filepath"

const (
	// Name is the name of the Alicloud provider.
	Name = "provider-alicloud"

	// InfraRelease is the name of the alicloud-infra chart.
	InfraRelease = "alicloud-infra"

	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// MachineControllerManagerProviderAlicloudImageName is the name of the MachineControllerManagerProviderAlicloud image.
	MachineControllerManagerProviderAlicloudImageName = "machine-controller-manager-provider-alicloud"
	// CloudControllerManagerImageName is the name of the CloudControllerManager image.
	CloudControllerManagerImageName = "alicloud-controller-manager"
	// CSIAttacherImageName is the name of the CSI attacher image.
	CSIAttacherImageName = "csi-attacher"
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

	// BucketName is a constant for the key in a backup secret that holds the bucket name.
	// The bucket name is written to the backup secret by Gardener as a temporary solution.
	// TODO In the future, the bucket name should come from a BackupBucket resource (see https://github.com/gardener/gardener/blob/master/docs/proposals/02-backupinfra.md)
	BucketName = "bucketName"

	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"
	// MachineControllerManagerVpaName is the name of the VerticalPodAutoscaler of the machine-controller-manager deployment.
	MachineControllerManagerVpaName = "machine-controller-manager-vpa"
	// MachineControllerManagerMonitoringConfigName is the name of the ConfigMap containing monitoring stack configurations for machine-controller-manager.
	MachineControllerManagerMonitoringConfigName = "machine-controller-manager-monitoring-config"
	// BackupSecretName is the name of the secret containing the credentials for storing the backups of Shoot clusters.
	BackupSecretName = "etcd-backup"
	// StorageEndpoint is the data field in a secret where the storage endpoint is stored at.
	StorageEndpoint = "storageEndpoint"
	//CloudControllerManagerName is the a constant for the name of the CloudController.
	CloudControllerManagerName = "cloud-controller-manager"
	// CSIPluginController is the a constant for the name of the csi-plugin-controller Deployment in the Seed.
	CSIPluginController = "csi-plugin-controller"
	// CSISnapshotControllerName is a constant for the name of the csi-snapshot-controller Deployment in the Seed.
	CSISnapshotControllerName = "csi-snapshot-controller"

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

	//ErrorCodeNoPermission is a constant for the error code of no permission.
	ErrorCodeNoPermission = "NoPermission"
	// ErrorCodeRoleEntityNotExist is a constant for the error code of role entity not exist.
	ErrorCodeRoleEntityNotExist = "EntityNotExist.Role"
	// ErrorCodeDomainRecordNotBelongToUser is a constant for the error code of domain record not belong to user.
	ErrorCodeDomainRecordNotBelongToUser = "DomainRecordNotBelongToUser"

	// DefaultDNSRegion is the default region to be used if a region is not specified in the DNS secret
	// or in the DNSRecord resource.
	DefaultDNSRegion = "cn-shanghai"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")
	// InfraChartPath is the path to the alicloud-infra chart.
	InfraChartPath = filepath.Join(InternalChartsPath, "alicloud-infra")
)
