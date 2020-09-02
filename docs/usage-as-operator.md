# Using the Alicloud provider extension with Gardener as operator

The [`core.gardener.cloud/v1beta1.CloudProfile` resource](https://github.com/gardener/gardener/blob/master/example/30-cloudprofile.yaml) declares a `providerConfig` field that is meant to contain provider-specific configuration.
The [`core.gardener.cloud/v1beta1.Seed` resource](https://github.com/gardener/gardener/blob/master/example/50-seed.yaml) is structured similarly.
Additionally, it allows configuring settings for the backups of the main etcds' data of shoot clusters control planes running in this seed cluster.

This document explains the necessary configuration for this provider extension. In addition, this document also describes how to enable the use of customized machine images for Alicloud.

## `CloudProfile` resource

This section describes, how the configuration for `CloudProfile` looks like for Alicloud by providing an example `CloudProfile` manifest with minimal configuration that can be used to allow the creation of Alicloud shoot clusters.

### `CloudProfileConfig`

The cloud profile configuration contains information about the real machine image IDs in the Alicloud environment (AMIs).
You have to map every version that you specify in `.spec.machineImages[].versions` here such that the Alicloud extension knows the AMI for every version you want to offer.

An example `CloudProfileConfig` for the Alicloud extension looks as follows:

```yaml
apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
kind: CloudProfileConfig
machineImages:
- name: coreos
  versions:
  - version: 2023.4.0
    regions:
    - name: eu-central-1
      id: coreos_2023_4_0_64_30G_alibase_20190319.vhd
```

### Example `CloudProfile` manifest

Please find below an example `CloudProfile` manifest:

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: CloudProfile
metadata:
  name: alicloud
spec:
  type: alicloud
  kubernetes:
    versions:
    - version: 1.16.1
    - version: 1.16.0
      expirationDate: "2020-04-05T01:02:03Z"
  machineImages:
  - name: coreos
    versions:
    - version: 2023.4.0
  machineTypes:
  - name: ecs.sn2ne.large
    cpu: "2"
    gpu: "0"
    memory: 8Gi
  volumeTypes:
  - name: cloud_efficiency
    class: standard
  - name: cloud_ssd
    class: premium
  regions:
  - name: eu-central-1
    zones:
    - name: eu-central-1a
    - name: eu-central-1b
  providerConfig:
    apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
    kind: CloudProfileConfig
    machineImages:
    - name: coreos
      versions:
      - version: 2023.4.0
        regions:
        - name: eu-central-1
          id: coreos_2023_4_0_64_30G_alibase_20190319.vhd
```

## Enable customized machine images for the Alicloud extension

Customized machine images can be created for an Alicloud account and shared with other Alicloud accounts. The same customized machine image has different image ID in different regions on Alicloud. Administrators/Operators need to explicitly declare them per imageID per region as below:

```yaml
machineImages:
- name: customized_coreos
  regions:
  - imageID: <image_id_in_eu_central_1>
    region: eu-central-1
  - imageID: <image_id_in_cn_shanghai>
    region: cn-shanghai
  ...
  version: 2191.4.1
...
```

End-users have to have the permission to use the customized image from its creator Alicloud account. To enable end-users to use customized images, the images are shared from Alicloud account of Seed operator with end-users' Alicloud accounts. Administrators/Operators need to explicitly provide Seed operator's Alicloud account access credentials (base64 encoded) as below:

```yaml
machineImageOwnerSecret:
  name: machine-image-owner
  accessKeyID: <base64_encoded_access_key_id>
  accessKeySecret: <base64_encoded_access_key_secret>
```

As a result, a Secret named `machine-image-owner` by default will be created in namespace of Alicloud provider extension.

Operators can also configure a whitelist of machine image IDs that are not to be shared with end-users as below:

```yaml
whitelistedImageIDs:
- <image_id_1>
- <image_id_2>
- <image_id_3>
```

### Example `ControllerRegistration` manifest for enabling customized machine images

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: ControllerRegistration
metadata:
  name: extension-provider-alicloud
spec:
  deployment:
    type: helm
    providerConfig:
      chart: |
        H4sIFAAAAAAA/yk...
      values:
        config:
          machineImageOwnerSecret:
            accessKeyID: <base64_encoded_access_key_id>
            accessKeySecret: <base64_encoded_access_key_secret>
          whitelistedImageIDs:
          - <image_id_1>
          - <image_id_2>
          ...
          machineImages:
          - name: customized_coreos
            regions:
            - imageID: <image_id_in_eu_central_1>
              region: eu-central-1
            - imageID: <image_id_in_cn_shanghai>
              region: cn-shanghai
            ...
            version: 2191.4.1
          ...
        resources:
          limits:
            cpu: 500m
            memory: 1Gi
          requests:
            memory: 128Mi
  resources:
  - kind: BackupBucket
    type: alicloud
  - kind: BackupEntry
    type: alicloud
  - kind: ControlPlane
    type: alicloud
  - kind: Infrastructure
    type: alicloud
  - kind: Worker
    type: alicloud
```

## `Seed` resource

This provider extension does not support any provider configuration for the `Seed`'s `.spec.provider.providerConfig` field.
However, it supports to managing of backup infrastructure, i.e., you can specify a configuration for the `.spec.backup` field.

### Backup configuration

A Seed of type `alicloud` can be configured to perform backups for the main etcds' of the shoot clusters control planes using Alicloud [Object Storage Service](https://www.alibabacloud.com/help/doc-detail/31817.htm).

The location/region where the backups will be stored defaults to the region of the Seed (`spec.provider.region`).

Please find below an example `Seed` manifest (partly) that configures backups using Alicloud Object Storage Service.

```yaml
---
apiVersion: core.gardener.cloud/v1beta1
kind: Seed
metadata:
  name: my-seed
spec:
  provider:
    type: alicloud
    region: cn-shanghai
  backup:
    provider: alicloud
    secretRef:
      name: backup-credentials
      namespace: garden
  ...
```
An example of the referenced secret containing the credentials for the Alicloud Object Storage Service can be found in the [example folder](https://github.com/gardener/gardener-extension-provider-alicloud/blob/master/example/30-etcd-backup-secret.yaml).

#### Permissions for Alicloud Object Storage Service

Please make sure the RAM user associated with the provided AccessKey pair has the following permission. 
- AliyunOSSFullAccess
