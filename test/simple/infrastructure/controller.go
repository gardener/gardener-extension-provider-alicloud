// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"syscall"

	"os"
	"os/signal"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud"
	alicloudinstall "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/install"
	alicloudv1alpha1 "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	secretName   = "cloudprovider"
	podCIDR      = "100.96.0.0/11"
	imageName    = "ubuntu"
	imageVersion = "20.04"
	space_name   = "provider-alicloud-test"
)

var (
	accessKeyID     = flag.String("access-key-id", "", "Alicloud access key id")
	accessKeySecret = flag.String("access-key-secret", "", "Alicloud access key secret")
	region          = flag.String("region", "", "Alicloud region")
	namespace       *corev1.Namespace
	cluster         *extensionsv1alpha1.Cluster
	c               client.Client
	mgrCancel       context.CancelFunc
)

func main() {

	repoRoot := filepath.Join("..", "..", "..")
	var ctx = context.Background()

	var testEnv = &envtest.Environment{
		UseExistingCluster: pointer.Bool(true),
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_clusters.yaml"),
				filepath.Join(repoRoot, "example", "20-crd-extensions.gardener.cloud_infrastructures.yaml"),
			},
		},
	}

	defer func() {
		if mgrCancel != nil {
			mgrCancel()
		}
		testEnv.Stop()
	}()

	cfg, err := testEnv.Start()
	if err != nil {
		runtimelog.Log.Error(err, "error when testEnv Start")
		panic("env start fail")
	}
	if cfg == nil {
		runtimelog.Log.Info("cfg is nil")
		panic("can not get env cfg")
	}

	mgr, err := manager.New(cfg, manager.Options{
		MetricsBindAddress: "0",
		Logger:             zap.New(zap.UseDevMode(true)),
	})

	if err != nil {
		runtimelog.Log.Error(err, "error when manager New")
		panic("new manager failed")
	}
	if err := extensionsv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		runtimelog.Log.Error(err, "error when extensionsv1alpha1.AddToScheme(mgr.GetScheme())")
		panic("extensionsv1alpha1.AddToScheme(mgr.GetScheme()) failed")
	}
	if err := alicloudinstall.AddToScheme(mgr.GetScheme()); err != nil {
		runtimelog.Log.Error(err, "error when alicloudinstall.AddToScheme(mgr.GetScheme())")
		panic("alicloudinstall.AddToScheme(mgr.GetScheme()) failed")
	}
	if err := infrastructure.AddToManagerWithOptions(ctx, mgr, infrastructure.AddOptions{
		// During testing in testmachinery cluster, there is no gardener-resource-manager to inject the volume mount.
		// Hence, we need to run without projected token mount.
		DisableProjectedTokenMount: true,
		IgnoreOperationAnnotation:  false,
	}); err != nil {
		runtimelog.Log.Error(err, "error when infrastructure.AddToManagerWithOptions")
		panic("infrastructure.AddToManagerWithOptions failed")
	}

	var mgrContext context.Context

	mgrContext, mgrCancel = context.WithCancel(ctx)

	go func() {

		err := mgr.Start(mgrContext)
		if err != nil {
			runtimelog.Log.Error(err, "error when mgr Start")
		}

	}()

	c = mgr.GetClient()
	if c == nil {
		runtimelog.Log.Info("mgr.GetClient is nil")
		panic("mgr.GetClient is nil")
	}

	flag.Parse()
	validateFlags()

	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: space_name,
		},
	}

	if err := c.Create(ctx, namespace); err != nil {
		if !apiErrors.IsAlreadyExists(err) {
			runtimelog.Log.Error(err, "error when create namespace")
			panic("create namespace failed")
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace.Name,
		},
		Data: map[string][]byte{
			alicloud.AccessKeyID:     []byte(*accessKeyID),
			alicloud.AccessKeySecret: []byte(*accessKeySecret),
		},
	}
	if err := c.Create(ctx, secret); err != nil {
		if !apiErrors.IsAlreadyExists(err) {
			runtimelog.Log.Error(err, "error when create secret")
			panic("create secret failed")
		}
	}

	cluster, _ = newCluster(space_name)
	if cluster != nil {
		if err := c.Create(ctx, cluster); err != nil {
			if !apiErrors.IsAlreadyExists(err) {
				runtimelog.Log.Error(err, "error when create cluster")
				panic("create cluster failed")
			}
		}
	}

	fmt.Println("\nCtrl+C pressed in Terminal, exiting.")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	<-ch

}

func newCluster(namespace string) (*extensionsv1alpha1.Cluster, error) {
	providerConfig := &alicloudv1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudProfileConfig",
			APIVersion: alicloudv1alpha1.SchemeGroupVersion.String(),
		},
		MachineImages: []alicloudv1alpha1.MachineImages{
			{
				Name: imageName,
				Versions: []alicloudv1alpha1.MachineImageVersion{
					{
						Version: imageVersion,
						Regions: []alicloudv1alpha1.RegionIDMapping{
							{
								Name: *region,
								ID:   getImageId(*region),
							},
						},
					},
				},
			},
		},
	}
	providerConfigJSON, err := json.Marshal(providerConfig)
	if err != nil {
		return nil, err
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudProfile",
			APIVersion: gardencorev1beta1.SchemeGroupVersion.String(),
		},
		Spec: gardencorev1beta1.CloudProfileSpec{
			ProviderConfig: &runtime.RawExtension{
				Raw: providerConfigJSON,
			},
		},
	}
	cloudProfileJSON, err := json.Marshal(cloudProfile)
	if err != nil {
		return nil, err
	}

	shoot := &gardencorev1beta1.Shoot{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Shoot",
			APIVersion: gardencorev1beta1.SchemeGroupVersion.String(),
		},
		Spec: gardencorev1beta1.ShootSpec{
			Provider: gardencorev1beta1.Provider{
				Type: "alicloud",
				Workers: []gardencorev1beta1.Worker{
					{
						Machine: gardencorev1beta1.Machine{
							Type: "ecs.g6.2xlarge",
							Image: &gardencorev1beta1.ShootMachineImage{
								Name:    imageName,
								Version: pointer.String(imageVersion),
							},
						},
						Volume: &gardencorev1beta1.Volume{
							Name:       pointer.String("workgroup"),
							Type:       pointer.String("cloud_efficiency"),
							VolumeSize: "200Gi",
							Encrypted:  pointer.Bool(false),
						},
					},
				},
			},
			Networking: &gardencorev1beta1.Networking{
				Pods: pointer.String(podCIDR),
			},
		},
	}
	shootJSON, err := json.Marshal(shoot)
	if err != nil {
		return nil, err
	}

	cluster := &extensionsv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: extensionsv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			CloudProfile: runtime.RawExtension{
				Raw: cloudProfileJSON,
			},
			Seed: runtime.RawExtension{
				Raw: []byte("{}"),
			},
			Shoot: runtime.RawExtension{
				Raw: shootJSON,
			},
		},
	}

	return cluster, nil
}

func getImageId(region string) string {
	regionImageMap := map[string]string{
		"cn-shanghai":    "m-uf6a3012pcuemma21nfk",
		"ap-southeast-2": "m-p0w8c5rj528oj84nlise",
		"eu-central-1":   "m-gw83xpc3q3yzpoahhckf",
	}

	return regionImageMap[region]
}

func validateFlags() {
	if len(*accessKeyID) == 0 {
		panic("need an Alicloud access key id")
	}
	if len(*accessKeySecret) == 0 {
		panic("need an Alicloud access key secret")
	}
	if len(*region) == 0 {
		panic("need an Alicloud region")
	}
}
