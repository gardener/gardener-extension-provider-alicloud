// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package bastion

import (
	"encoding/json"
	"errors"

	alicloudclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	alicloudApi "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// sshPort is the default SSH Port used for bastion ingress firewall rule
	sshPort = "22"
)

type actuator struct {
	client           client.Client
	logger           logr.Logger
	newClientFactory alicloudclient.ClientFactory
}

func newActuator() bastion.Actuator {
	return &actuator{
		logger:           logger,
		newClientFactory: alicloudclient.NewClientFactory(),
	}
}

func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

func unmarshalClusterProviderConfig(bytes []byte) (*alicloudApi.CloudProfileConfig, error) {
	cloudProfileConfig := &alicloudApi.CloudProfileConfig{}

	err := json.Unmarshal(bytes, cloudProfileConfig)
	if err != nil {
		return nil, errors.New("failed to parse json for cloud ProfileConfig")
	}
	return cloudProfileConfig, nil
}

func getClusterImageID(cloudProfileConfig alicloudApi.CloudProfileConfig, name, version, regionName string) (string, error) {
	var fallbackImageID string
	for _, machineImage := range cloudProfileConfig.MachineImages {
		if machineImage.Name != name {
			continue
		}

		for _, machine := range machineImage.Versions {
			if machine.Version != version {
				continue
			}

			for _, region := range machine.Regions {
				if region.Name == regionName {
					return region.ID, nil
				} else {
					fallbackImageID = machine.Regions[0].ID
				}
			}

		}

	}

	if fallbackImageID == "" {
		return "", errors.New("fall back ImageID must be not empty")
	}
	return fallbackImageID, nil
}

func getImageID(cluster *controller.Cluster, opt *Options) (string, error) {
	if cluster.CloudProfile.Spec.ProviderConfig == nil || cluster.CloudProfile.Spec.ProviderConfig.Raw == nil {
		return "", errors.New("providerconfig Raw must be not empty")
	}

	clusterProviderConfig, err := unmarshalClusterProviderConfig(cluster.CloudProfile.Spec.ProviderConfig.Raw)
	if err != nil {
		return "", err
	}

	if len(cluster.Shoot.Spec.Provider.Workers) == 0 {
		return "", errors.New("shoot worker must not be empty")
	}

	workerVersion := cluster.Shoot.Spec.Provider.Workers[0].Machine.Image.Version
	imageName := cluster.Shoot.Spec.Provider.Workers[0].Machine.Image.Name

	if clusterProviderConfig == nil {
		return "", errors.New("clusterProviderConfig must be not empty")
	}

	if workerVersion == nil || *workerVersion == "" {
		return "", errors.New("workerVersion must be not empty")
	}

	imageID, err := getClusterImageID(*clusterProviderConfig, imageName, *workerVersion, opt.Region)
	if err != nil {
		return "", err
	}

	return imageID, nil
}
