// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	extensionssecretmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-alicloud/charts"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/helper"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/config"
)

// Object names
const (
	caNameControlPlane               = "ca-" + alicloud.Name + "-controlplane"
	cloudControllerManagerServerName = "cloud-controller-manager-server"
)

func secretConfigsFunc(namespace string) []extensionssecretmanager.SecretConfigWithOptions {
	return []extensionssecretmanager.SecretConfigWithOptions{
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:       caNameControlPlane,
				CommonName: caNameControlPlane,
				CertType:   secretutils.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        cloudControllerManagerServerName,
				CommonName:                  alicloud.CloudControllerManagerName,
				DNSNames:                    kutil.DNSNamesForService(alicloud.CloudControllerManagerName, namespace),
				CertType:                    secretutils.ServerCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
	}
}

func shootAccessSecretsFunc(namespace string) []*gutil.AccessSecret {
	return []*gutil.AccessSecret{
		gutil.NewShootAccessSecret("cloud-controller-manager", namespace),
		gutil.NewShootAccessSecret("csi-controller-ali-plugin", namespace),
		gutil.NewShootAccessSecret("csi-attacher", namespace),
		gutil.NewShootAccessSecret("csi-provisioner", namespace),
		gutil.NewShootAccessSecret("csi-snapshotter", namespace),
		gutil.NewShootAccessSecret("csi-resizer", namespace),
		gutil.NewShootAccessSecret("csi-snapshot-controller", namespace),
	}
}

var controlPlaneChart = &chart.Chart{
	Name:       "seed-controlplane",
	EmbeddedFS: charts.InternalChart,
	Path:       filepath.Join(charts.InternalChartsPath, "seed-controlplane"),
	SubCharts: []*chart.Chart{
		{
			Name:   "alicloud-cloud-controller-manager",
			Images: []string{alicloud.CloudControllerManagerImageName},
			Objects: []*chart.Object{
				{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
				{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},
				{Type: &corev1.Secret{}, Name: "cloud-provider-config"},
				{Type: &monitoringv1.ServiceMonitor{}, Name: "shoot-cloud-controller-manager"},
				{Type: &monitoringv1.PrometheusRule{}, Name: "shoot-cloud-controller-manager"},
				{Type: &vpaautoscalingv1.VerticalPodAutoscaler{}, Name: "cloud-controller-manager-vpa"},
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
				alicloud.CSILivenessProbeImageName,
				alicloud.CSISnapshotControllerImageName,
			},
			Objects: []*chart.Object{
				{Type: &appsv1.Deployment{}, Name: "csi-plugin-controller"},
				{Type: &vpaautoscalingv1.VerticalPodAutoscaler{}, Name: "csi-plugin-controller-vpa"},
				{Type: &appsv1.Deployment{}, Name: "csi-snapshot-controller"},
				{Type: &vpaautoscalingv1.VerticalPodAutoscaler{}, Name: "csi-snapshot-controller-vpa"},
			},
		},
	},
}

var controlPlaneShootChart = &chart.Chart{
	Name:       "shoot-system-components",
	EmbeddedFS: charts.InternalChart,
	Path:       filepath.Join(charts.InternalChartsPath, "shoot-system-components"),
	SubCharts: []*chart.Chart{
		{
			Name: "alicloud-cloud-controller-manager",
			Objects: []*chart.Object{
				{Type: &rbacv1.ClusterRole{}, Name: "system:controller:cloud-node-controller"},
				{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:controller:cloud-node-controller"},
			},
		},
		{
			Name: "csi-alicloud",
			Images: []string{
				alicloud.CSINodeDriverRegistrarImageName,
				alicloud.CSIPluginImageName,
				alicloud.CSIPluginInitImageName,
				alicloud.CSILivenessProbeImageName,
			},
			Objects: []*chart.Object{
				// csi-disk-plugin-alicloud
				{Type: &appsv1.DaemonSet{}, Name: "csi-disk-plugin-alicloud"},
				{Type: &corev1.Secret{}, Name: "csi-diskplugin-alicloud"},
				{Type: &corev1.ServiceAccount{}, Name: "csi-disk-plugin-alicloud"},
				// csi-controller-ali-plugin
				{Type: &corev1.ServiceAccount{}, Name: "csi-controller-ali-plugin"},
				{Type: &rbacv1.ClusterRole{}, Name: extensionsv1alpha1.SchemeGroupVersion.Group + ":kube-system:csi-controller-ali-plugin"},
				{Type: &corev1.ServiceAccount{}, Name: "csi-controller-ali-plugin"},
				{Type: &rbacv1.Role{}, Name: "csi-controller-ali-plugin"},
				{Type: &rbacv1.RoleBinding{}, Name: "csi-controller-ali-plugin"},
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

var controlPlaneShootCRDsChart = &chart.Chart{
	Name:       "shoot-crds",
	EmbeddedFS: charts.InternalChart,
	Path:       filepath.Join(charts.InternalChartsPath, "shoot-crds"),
	SubCharts: []*chart.Chart{
		{
			Name: "volumesnapshots",
			Objects: []*chart.Object{
				{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: alicloud.CRDVolumeSnapshotClasses},
				{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: alicloud.CRDVolumeSnapshotContents},
				{Type: &apiextensionsv1.CustomResourceDefinition{}, Name: alicloud.CRDVolumeSnapshots},
			},
		},
	},
}

var storageClassChart = &chart.Chart{
	Name:       "shoot-storageclasses",
	EmbeddedFS: charts.InternalChart,
	Path:       filepath.Join(charts.InternalChartsPath, "shoot-storageclasses"),
}

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(mgr manager.Manager, csi config.CSI) genericactuator.ValuesProvider {
	return &valuesProvider{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		csi:     csi,
	}
}

// valuesProvider is a ValuesProvider that provides Alicloud-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	genericactuator.NoopValuesProvider
	client  client.Client
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	csi     config.CSI
}

// GetControlPlaneChartValues returns the values for the control plane chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	_ secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	cpConfig, err := vp.decodeControlPlaneConfig(cp)
	if err != nil {
		return nil, err
	}

	if err := cleanupSeedLegacyCSISnapshotValidation(ctx, vp.client, cp.Namespace); err != nil {
		return nil, err
	}

	// Get control plane chart values
	return vp.getControlPlaneChartValues(ctx, cpConfig, cp, cluster, checksums, scaledDown)
}

// GetControlPlaneShootChartValues returns the values for the control plane shoot chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneShootChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	_ *extensionscontroller.Cluster,
	_ secretsmanager.Reader,
	_ map[string]string,
) (map[string]interface{}, error) {
	// Get credentials from the referenced secret
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, vp.client, &cp.Spec.SecretRef)
	if err != nil {
		return nil, fmt.Errorf("could not read credentials from secret referred by controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
	}

	cpConfig, err := vp.decodeControlPlaneConfig(cp)
	if err != nil {
		return nil, err
	}

	// Get control plane shoot chart values
	return vp.getControlPlaneShootChartValues(cpConfig, credentials)
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

func (vp *valuesProvider) decodeControlPlaneConfig(cp *extensionsv1alpha1.ControlPlane) (*apisalicloud.ControlPlaneConfig, error) {
	cpConfig := &apisalicloud.ControlPlaneConfig{}

	// The custom decoder which disables the strict mode.
	// "Zone" field in not required in "ControlPlaneConfig" any more, and it has already been removed in the struct, but
	// still exists in shoots' yaml. So it should also be removed in the shoots' yaml.
	// Here we leverage one custom decoder (disabled the strict mode) to make migration more smooth.
	// TODO: should be removed in next release.
	decoder := serializer.NewCodecFactory(vp.scheme).UniversalDecoder()
	if cp.Spec.ProviderConfig != nil {
		if _, _, err := decoder.Decode(cp.Spec.ProviderConfig.Raw, nil, cpConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
		}
	}
	return cpConfig, nil
}

func (vp *valuesProvider) getCloudControllerManagerConfigFileContent(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
) (string, error) {
	// Decode infrastructureProviderStatus
	infraStatus := &apisalicloud.InfrastructureStatus{}
	if _, _, err := vp.decoder.Decode(cp.Spec.InfrastructureProviderStatus.Raw, nil, infraStatus); err != nil {
		return "", fmt.Errorf("could not decode infrastructureProviderStatus of controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
	}

	// Get credentials from the referenced secret
	credentials, err := alicloud.ReadCredentialsFromSecretRef(ctx, vp.client, &cp.Spec.SecretRef)
	if err != nil {
		return "", fmt.Errorf("could not read credentials from secret referred by controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
	}

	// Find first vswitch with purpose "nodes"
	vswitch, err := helper.FindVSwitchForPurpose(infraStatus.VPC.VSwitches, apisalicloud.PurposeNodes)
	if err != nil {
		return "", fmt.Errorf("could not determine vswitch from infrastructureProviderStatus of controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
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
		return "", fmt.Errorf("could not marshal cloud config to JSON for controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
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
		return nil, fmt.Errorf("could not build cloud controller config file content for controlplane '%s': %w", client.ObjectKeyFromObject(cp), err)
	}

	ccmNetworkFalg := "public"
	if cluster.Seed != nil && cluster.Seed.Spec.Provider.Type == alicloud.Type && cluster.Shoot != nil && cluster.Seed.Spec.Provider.Region == cluster.Shoot.Spec.Region {
		ccmNetworkFalg = "vpc"
	}
	values := map[string]interface{}{
		"global": map[string]interface{}{
			"genericTokenKubeconfigSecretName": extensionscontroller.GenericTokenKubeconfigSecretNameFromCluster(cluster),
		},
		"alicloud-cloud-controller-manager": map[string]interface{}{
			"replicas":    extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"clusterName": cp.Namespace,
			"podNetwork":  strings.Join(extensionscontroller.GetPodNetwork(cluster), ","),
			"podLabels": map[string]interface{}{
				v1beta1constants.LabelPodMaintenanceRestart: "true",
			},
			"cloudConfig":    ccmConfig,
			"ccmNetworkFalg": ccmNetworkFalg,
		},
		"csi-alicloud": map[string]interface{}{
			"regionID":           cp.Spec.Region,
			"replicas":           extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"enableADController": vp.enableCSIADController(cpConfig),
			"csiPluginController": map[string]interface{}{
				"snapshotPrefix":         cluster.Shoot.Name,
				"persistentVolumePrefix": cluster.Shoot.Name,
				"podAnnotations": map[string]interface{}{
					"checksum/secret-cloudprovider": checksums[v1beta1constants.SecretNameCloudProvider],
				},
			},
			"csiSnapshotController": map[string]interface{}{},
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["alicloud-cloud-controller-manager"].(map[string]interface{})["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	return values, nil
}

func (vp *valuesProvider) enableCSIADController(cpConfig *apisalicloud.ControlPlaneConfig) bool {
	return vp.csi.EnableADController != nil && *vp.csi.EnableADController || cpConfig.CSI != nil && cpConfig.CSI.EnableADController != nil && *cpConfig.CSI.EnableADController
}

// getControlPlaneShootChartValues collects and returns the control plane shoot chart values.
func (vp *valuesProvider) getControlPlaneShootChartValues(
	cpConfig *apisalicloud.ControlPlaneConfig,
	credentials *alicloud.Credentials,

) (map[string]interface{}, error) {
	values := map[string]interface{}{
		"csi-alicloud": map[string]interface{}{
			"credential": map[string]interface{}{
				"credentialsFile": base64.StdEncoding.EncodeToString([]byte(credentials.CredentialsFile)),
			},
			"enableADController": vp.enableCSIADController(cpConfig),
		},
	}

	return values, nil
}

func cleanupSeedLegacyCSISnapshotValidation(
	ctx context.Context,
	client client.Client,
	namespace string,
) error {
	if err := kutil.DeleteObjects(ctx, client,
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: alicloud.CSISnapshotValidationName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: alicloud.CSISnapshotValidationName, Namespace: namespace}},
		&vpaautoscalingv1.VerticalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "csi-snapshot-webhook-vpa", Namespace: namespace}},
		&policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: alicloud.CSISnapshotValidationName, Namespace: namespace}},
	); err != nil {
		return fmt.Errorf("failed to delete legacy csi-snapshot-validation resources: %w", err)
	}

	return nil
}
