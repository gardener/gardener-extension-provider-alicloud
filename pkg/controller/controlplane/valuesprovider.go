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

package controlplane

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authentication/user"
)

var controlPlaneSecrets = &secrets.Secrets{
	CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
		v1beta1constants.SecretNameCACluster: {
			Name:       v1beta1constants.SecretNameCACluster,
			CommonName: "kubernetes",
			CertType:   secrets.CACert,
		},
	},
	SecretConfigsFunc: func(cas map[string]*secrets.Certificate, clusterName string) []secrets.ConfigInterface {
		return []secrets.ConfigInterface{
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         "cloud-controller-manager",
					CommonName:   "system:cloud-controller-manager",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         "csi-attacher",
					CommonName:   "system:csi-attacher",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         "csi-provisioner",
					CommonName:   "system:csi-provisioner",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         "csi-snapshotter",
					CommonName:   "system:csi-snapshotter",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         "csi-resizer",
					CommonName:   "system:csi-resizer",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         "csi-snapshot-controller",
					CommonName:   "system:csi-snapshot-controller",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1beta1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1beta1constants.DeploymentNameKubeAPIServer,
				},
			},
		}
	},
}

var controlPlaneChart = &chart.Chart{
	Name: "seed-controlplane",
	Path: filepath.Join(alicloud.InternalChartsPath, "seed-controlplane"),
	SubCharts: []*chart.Chart{
		{
			Name:   "alicloud-cloud-controller-manager",
			Images: []string{alicloud.CloudControllerManagerImageName},
			Objects: []*chart.Object{
				{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
				{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},
				{Type: &corev1.Secret{}, Name: "cloud-provider-config"},
				{Type: &corev1.ConfigMap{}, Name: "cloud-controller-manager-monitoring-config"},
			},
		},
		{
			Name: "csi-alicloud",
			Images: []string{
				alicloud.CSIAttacherImageName,
				alicloud.CSIProvisionerImageName,
				alicloud.CSISnapshotterImageName,
				alicloud.CSIResizerImageName,
				alicloud.CSIPluginImageName,
				alicloud.CSISnapshotControllerImageName,
			},
			Objects: []*chart.Object{
				{Type: &appsv1.Deployment{}, Name: "csi-plugin-controller"},
				{Type: &appsv1.Deployment{}, Name: "csi-snapshot-controller"},
			},
		},
	},
}

var controlPlaneShootChart = &chart.Chart{
	Name: "shoot-system-components",
	Path: filepath.Join(alicloud.InternalChartsPath, "shoot-system-components"),
	SubCharts: []*chart.Chart{
		{
			Name: "alicloud-cloud-controller-manager",
			Objects: []*chart.Object{
				{Type: &rbacv1.ClusterRole{}, Name: "system:controller:cloud-node-controller"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:controller:cloud-node-controller"},
			},
		},
		{
			Name:   "csi-alicloud",
			Images: []string{alicloud.CSINodeDriverRegistrarImageName, alicloud.CSIPluginImageName},
			Objects: []*chart.Object{
				// csi-disk-plugin-alicloud
				{Type: &appsv1.DaemonSet{}, Name: "csi-disk-plugin-alicloud"},
				{Type: &corev1.Secret{}, Name: "csi-diskplugin-alicloud"},
				{Type: &corev1.ServiceAccount{}, Name: "csi-disk-plugin-alicloud"},
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":psp:kube-system:csi-disk-plugin-alicloud"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":psp:csi-disk-plugin-alicloud"},
				{Type: &policyv1beta1.PodSecurityPolicy{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ".kube-system.csi-disk-plugin-alicloud"},
				{Type: extensionscontroller.GetVerticalPodAutoscalerObject(), Name: "csi-diskplugin-alicloud"},
				// csi-attacher
				{Type: &corev1.ServiceAccount{}, Name: "csi-attacher"},
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":kube-system:csi-attacher"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":csi-attacher"},
				{Type: &rbacv1.Role{}, Name: "csi-attacher"},
				{Type: &rbacv1.RoleBinding{}, Name: "csi-attacher"},
				// csi-provisioner
				{Type: &corev1.ServiceAccount{}, Name: "csi-provisioner"},
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":kube-system:csi-provisioner"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":csi-provisioner"},
				{Type: &rbacv1.Role{}, Name: "csi-provisioner"},
				{Type: &rbacv1.RoleBinding{}, Name: "csi-provisioner"},
				// csi-snapshotter
				{Type: &apiextensionsv1beta1.CustomResourceDefinition{}, Name: alicloud.CRDVolumeSnapshotClasses},
				{Type: &apiextensionsv1beta1.CustomResourceDefinition{}, Name: alicloud.CRDVolumeSnapshotContents},
				{Type: &apiextensionsv1beta1.CustomResourceDefinition{}, Name: alicloud.CRDVolumeSnapshots},
				{Type: &corev1.ServiceAccount{}, Name: "csi-snapshotter"},
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":kube-system:csi-snapshotter"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":csi-snapshotter"},
				{Type: &rbacv1.Role{}, Name: "csi-snapshotter"},
				{Type: &rbacv1.RoleBinding{}, Name: "csi-snapshotter"},
				// csi-snapshot-controller
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":kube-system:csi-snapshot-controller"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":csi-snapshot-controller"},
				{Type: &rbacv1.Role{}, Name: "csi-snapshot-controller"},
				{Type: &rbacv1.RoleBinding{}, Name: "csi-snapshot-controller"},
				// csi-resizer
				{Type: &corev1.ServiceAccount{}, Name: "csi-resizer"},
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":kube-system:csi-resizer"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":csi-resizer"},
				{Type: &rbacv1.Role{}, Name: "csi-resizer"},
				{Type: &rbacv1.RoleBinding{}, Name: "csi-resizer"},
			},
		},
	},
}

var storageClassChart = &chart.Chart{
	Name: "shoot-storageclasses",
	Path: filepath.Join(alicloud.InternalChartsPath, "shoot-storageclasses"),
}

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(logger logr.Logger) genericactuator.ValuesProvider {
	return &valuesProvider{
		logger: logger.WithName("alicloud-values-provider"),
	}
}

// valuesProvider is a ValuesProvider that provides Alicloud-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	genericactuator.NoopValuesProvider
	common.ClientContext
	logger logr.Logger
}

// GetControlPlaneChartValues returns the values for the control plane chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	// Decode providerConfig
	cpConfig := &apisalicloud.ControlPlaneConfig{}
	if cp.Spec.ProviderConfig != nil {
		if _, _, err := vp.Decoder().Decode(cp.Spec.ProviderConfig.Raw, nil, cpConfig); err != nil {
			return nil, errors.Wrapf(err, "could not decode providerConfig of controlplane '%s'", kutil.ObjectName(cp))
		}
	}
	// TODO: Remove this code in next version. Delete old config
	if err := vp.deleteCloudProviderConfig(ctx, cp.Namespace); err != nil {
		return nil, err
	}
	// Get control plane chart values
	return vp.getControlPlaneChartValues(ctx, cpConfig, cp, cluster, checksums, scaledDown)
}

// GetControlPlaneShootChartValues returns the values for the control plane shoot chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneShootChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
) (map[string]interface{}, error) {
	// Get credentials from the referenced secret
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, vp.Client(), &cp.Spec.SecretRef)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read credentials from secret referred by controlplane '%s'", kutil.ObjectName(cp))
	}

	// Get control plane shoot chart values
	return getControlPlaneShootChartValues(cluster, credentials)
}

// cloudConfig wraps the settings for the Alicloud provider.
// See https://github.com/kubernetes/cloud-provider-alibaba-cloud/blob/master/cloud-controller-manager/alicloud.go
type cloudConfig struct {
	Global struct {
		KubernetesClusterTag string
		ClusterID            string `json:"clusterID"`
		UID                  string `json:"uid"`
		VpcID                string `json:"vpcid"`
		Region               string `json:"region"`
		ZoneID               string `json:"zoneid"`
		VswitchID            string `json:"vswitchid"`

		AccessKeyID     string `json:"accessKeyID"`
		AccessKeySecret string `json:"accessKeySecret"`
	}
}

func (vp *valuesProvider) getCloudControllerManagerConfigFileContent(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
) (string, error) {
	// Decode infrastructureProviderStatus
	infraStatus := &apisalicloud.InfrastructureStatus{}
	if _, _, err := vp.Decoder().Decode(cp.Spec.InfrastructureProviderStatus.Raw, nil, infraStatus); err != nil {
		return "", errors.Wrapf(err, "could not decode infrastructureProviderStatus of controlplane '%s'", kutil.ObjectName(cp))
	}

	// Get credentials from the referenced secret
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, vp.Client(), &cp.Spec.SecretRef)
	if err != nil {
		return "", errors.Wrapf(err, "could not read credentials from secret referred by controlplane '%s'", kutil.ObjectName(cp))
	}

	// Find first vswitch with purpose "nodes"
	vswitch, err := helper.FindVSwitchForPurpose(infraStatus.VPC.VSwitches, apisalicloud.PurposeNodes)
	if err != nil {
		return "", errors.Wrapf(err, "could not determine vswitch from infrastructureProviderStatus of controlplane '%s'", kutil.ObjectName(cp))
	}

	// Initialize cloud config
	cfg := &cloudConfig{}
	cfg.Global.KubernetesClusterTag = cp.Namespace
	cfg.Global.ClusterID = cp.Namespace
	cfg.Global.VpcID = infraStatus.VPC.ID
	cfg.Global.ZoneID = vswitch.Zone
	cfg.Global.VswitchID = vswitch.ID
	cfg.Global.AccessKeyID = base64.StdEncoding.EncodeToString([]byte(credentials.AccessKeyID))
	cfg.Global.AccessKeySecret = base64.StdEncoding.EncodeToString([]byte(credentials.AccessKeySecret))
	cfg.Global.Region = cp.Spec.Region

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal cloud config to JSON for controlplane '%s'", kutil.ObjectName(cp))
	}

	return string(cfgJSON), nil
}

// getControlPlaneChartValues collects and returns the control plane chart values.
func (vp *valuesProvider) getControlPlaneChartValues(
	ctx context.Context,
	cpConfig *apisalicloud.ControlPlaneConfig,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	ccmConfig, err := vp.getCloudControllerManagerConfigFileContent(ctx, cp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not build cloud controller config file content for controlplain '%s", kutil.ObjectName(cp))
	}
	values := map[string]interface{}{
		"alicloud-cloud-controller-manager": map[string]interface{}{
			"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"clusterName":       cp.Namespace,
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
			"podNetwork":        extensionscontroller.GetPodNetwork(cluster),
			"podAnnotations": map[string]interface{}{
				"checksum/secret-cloud-controller-manager": checksums["cloud-controller-manager"],
				"checksum/secret-cloud-provider-config":    checksums["cloud-provider-config"],
			},
			"podLabels": map[string]interface{}{
				v1beta1constants.LabelPodMaintenanceRestart: "true",
			},
			"cloudConfig": ccmConfig,
		},
		"csi-alicloud": map[string]interface{}{
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
			"regionID":          cp.Spec.Region,
			"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"csiPluginController": map[string]interface{}{
				"snapshotPrefix":         cluster.Shoot.Name,
				"persistentVolumePrefix": cluster.Shoot.Name,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-csi-attacher":    checksums["csi-attacher"],
					"checksum/secret-csi-provisioner": checksums["csi-provisioner"],
					"checksum/secret-csi-snapshotter": checksums["csi-snapshotter"],
					"checksum/secret-csi-resizer":     checksums["csi-resizer"],
					"checksum/secret-cloudprovider":   checksums[v1beta1constants.SecretNameCloudProvider],
				},
			},
			"csiSnapshotController": map[string]interface{}{
				"podAnnotations": map[string]interface{}{
					"checksum/secret-csi-snapshot-controller": checksums["csi-snapshot-controller"],
				},
			},
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["alicloud-cloud-controller-manager"].(map[string]interface{})["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	return values, nil
}

// getControlPlaneShootChartValues collects and returns the control plane shoot chart values.
func getControlPlaneShootChartValues(
	cluster *extensionscontroller.Cluster,
	credentials *alicloud.Credentials,
) (map[string]interface{}, error) {
	values := map[string]interface{}{
		"csi-alicloud": map[string]interface{}{
			"credential": map[string]interface{}{
				"accessKeyID":     base64.StdEncoding.EncodeToString([]byte(credentials.AccessKeyID)),
				"accessKeySecret": base64.StdEncoding.EncodeToString([]byte(credentials.AccessKeySecret)),
			},
			"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
			"vpaEnabled":        gardencorev1beta1helper.ShootWantsVerticalPodAutoscaler(cluster.Shoot),
		},
	}

	return values, nil
}
