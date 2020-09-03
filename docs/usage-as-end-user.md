# Using the Alicloud provider extension with Gardener as end-user

The [`core.gardener.cloud/v1beta1.Shoot` resource](https://github.com/gardener/gardener/blob/master/example/90-shoot.yaml) declares a few fields that are meant to contain provider-specific configuration.

This document describes the configurable options for Alicloud and provides an example `Shoot` manifest with minimal configuration that can be used to create an Alicloud cluster (modulo the landscape-specific information like cloud profile names, secret binding names, etc.).

## Alicloud Provider Credentials

In order for Gardener to create a Kubernetes cluster using Alicloud infrastructure components, a Shoot has to provide credentials with sufficient permissions to the desired Alicloud project.
Every shoot cluster references a `SecretBinding` which itself references a `Secret`, and this `Secret` contains the provider credentials of the Alicloud project.
The `SecretBinding` is configurable in the [Shoot cluster](https://github.com/gardener/gardener/blob/master/example/90-shoot.yaml) with the field `secretBindingName`.

The required credentials for the Alicloud project are an [AccessKey Pair](https://www.alibabacloud.com/help/doc-detail/29009.htm) associated with a [Resource Access Management (RAM) User](https://www.alibabacloud.com/help/doc-detail/28627.htm).
A RAM user is a special account that can be used by services and applications to interact with Alicloud Cloud Platform APIs. 
Applications can use AccessKey pair to authorize themselves to a set of APIs and perform actions within the permissions granted to the RAM user.

Make sure to [create a Resource Access Management User](https://www.alibabacloud.com/help/doc-detail/93720.htm), and [create an AccessKey Pair](https://partners-intl.aliyun.com/help/doc-detail/116401.htm) that shall be used for the Shoot cluster.

[Grant at least the following permissions](https://partners-intl.aliyun.com/help/doc-detail/116146.htm) to the Resource Access Management User.
- AliyunECSFullAccess
- AliyunVPCFullAccess
- AliyunSLBFullAccess
- AliyunNATGatewayFullAccess
- AliyunEIPFullAccess

Provide AccessKey Pair in the `Secret` (base64 encoded for fields `accessKeyID` and `accessKeySecret`, respectively), that is being referenced by the `SecretBinding` in the Shoot cluster configuration.

This `Secret` must look as follows:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: core-alicloud
  namespace: garden-dev
type: Opaque
data:
  accessKeyID: base64(access-key-id)
  accessKeySecret: base64(access-key-secret)
```

## `InfrastructureConfig`

The infrastructure configuration mainly describes how the network layout looks like in order to create the shoot worker nodes in a later step, thus, prepares everything relevant to create VMs, load balancers, volumes, etc.

An example `InfrastructureConfig` for the Alicloud extension looks as follows:

```yaml
apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
kind: InfrastructureConfig
networks:
  vpc: # specify either 'id' or 'cidr'
  # id: my-vpc
    cidr: 10.250.0.0/16
  zones:
  - name: eu-central-1a
    workers: 10.250.1.0/24
  # natGateway:
    # eipAllocationID: eip-ufxsdg122elmszcg
```

The `networks.vpc` section describes whether you want to create the shoot cluster in an already existing VPC or whether to create a new one:

* If `networks.vpc.id` is given then you have to specify the VPC ID of the existing VPC that was created by other means (manually, other tooling, ...).
* If `networks.vpc.cidr` is given then you have to specify the VPC CIDR of a new VPC that will be created during shoot creation.
You can freely choose a private CIDR range.
* Either `networks.vpc.id` or `networks.vpc.cidr` must be present, but not both at the same time.

The `networks.zones` section describes which subnets you want to create in availability zones.
For every zone, the Alicloud extension creates one subnet:

* The `workers` subnet is used for all shoot worker nodes, i.e., VMs which later run your applications.

For every subnet, you have to specify a CIDR range contained in the VPC CIDR specified above, or the VPC CIDR of your already existing VPC.
You can freely choose these CIDR and it is your responsibility to properly design the network layout to suit your needs.

If you want to use multiple availability zones then add a second, third, ... entry to the `networks.zones[]` list and properly specify the AZ name in `networks.zones[].name`.

Apart from the VPC and the subnets the Alicloud extension will also create a NAT gateway (only if a new VPC is created), a key pair, elastic IPs, VSwitches, a SNAT table entry, and security groups.

By default, the Alicloud extension will create a corresponding Elastic IP that it attaches to this NAT gateway and which is used for egress traffic. 
The `networks.zones[].natGateway.eipAllocationID` field allows you to specify the Elastic IP Allocation ID of an existing Elastic IP allocation in case you want to bring your own.
If provided, no new Elastic IP will be created and, instead, the Elastic IP specified by you will be used.

⚠️ If you change this field for an already existing infrastructure then it will disrupt egress traffic while Alicloud applies this change, because the NAT gateway must be recreated with the new Elastic IP association.
Also, please note that the existing Elastic IP will be permanently deleted if it was earlier created by the Alicloud extension. 

## `ControlPlaneConfig`

The control plane configuration mainly contains values for the Alicloud-specific control plane components.
Today, the Alicloud extension deploys the `cloud-controller-manager` and the CSI controllers.

An example `ControlPlaneConfig` for the Alicloud extension looks as follows:

```yaml
apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
kind: ControlPlaneConfig
cloudControllerManager:
  featureGates:
    CustomResourceValidation: true
```

The `cloudControllerManager.featureGates` contains a map of explicitly enabled or disabled feature gates.
For production usage it's not recommend to use this field at all as you can enable alpha features or disable beta/stable features, potentially impacting the cluster stability.
If you don't want to configure anything for the `cloudControllerManager` simply omit the key in the YAML specification.

## `WorkerConfig`

The Alicloud extension does not support a specific `WorkerConfig` yet, however, it supports additional data volumes (plus encryption) per machine.
By default (if not stated otherwise), all the disks are unencrypted.
Please note that it is currently only possible to encrypt data disks (system disk is unsupported).
For each data volume, you have to specify a name.
The following YAML is a snippet of a `Shoot` resource:

```yaml
spec:
  provider:
    workers:
    - name: cpu-worker
      ...
      volume:
        type: cloud_efficiency
        size: 20Gi
      dataVolumes:
      - name: kubelet-dir
        type: cloud_efficiency
        size: 25Gi
        encrypted: true
```

## Example `Shoot` manifest (one availability zone)

Please find below an example `Shoot` manifest for one availability zone:

```yaml
apiVersion: core.gardener.cloud/v1alpha1
kind: Shoot
metadata:
  name: johndoe-alicloud
  namespace: garden-dev
spec:
  cloudProfileName: alicloud
  region: eu-central-1
  secretBindingName: core-alicloud
  provider:
    type: alicloud
    infrastructureConfig:
      apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
      kind: InfrastructureConfig
      networks:
        vpc:
          cidr: 10.250.0.0/16
        zones:
        - name: eu-central-1a
          workers: 10.250.0.0/19
    controlPlaneConfig:
      apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
      kind: ControlPlaneConfig
    workers:
    - name: worker-xoluy
      machine:
        type: ecs.sn2ne.large
      minimum: 2
      maximum: 2
      volume:
        size: 50Gi
        type: cloud_efficiency
      zones:
      - eu-central-1a
  networking:
    nodes: 10.250.0.0/16
    type: calico
  kubernetes:
    version: 1.16.1
  maintenance:
    autoUpdate:
      kubernetesVersion: true
      machineImageVersion: true
  addons:
    kubernetes-dashboard:
      enabled: true
    nginx-ingress:
      enabled: true
```

## Example `Shoot` manifest (two availability zones)

Please find below an example `Shoot` manifest for two availability zones:

```yaml
apiVersion: core.gardener.cloud/v1alpha1
kind: Shoot
metadata:
  name: johndoe-alicloud
  namespace: garden-dev
spec:
  cloudProfileName: alicloud
  region: eu-central-1
  secretBindingName: core-alicloud
  provider:
    type: alicloud
    infrastructureConfig:
      apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
      kind: InfrastructureConfig
      networks:
        vpc:
          cidr: 10.250.0.0/16
        zones:
        - name: eu-central-1a
          workers: 10.250.0.0/26
        - name: eu-central-1b
          workers: 10.250.0.64/26
    controlPlaneConfig:
      apiVersion: alicloud.provider.extensions.gardener.cloud/v1alpha1
      kind: ControlPlaneConfig
      zone: eu-central-1a
    workers:
    - name: worker-xoluy
      machine:
        type: ecs.sn2ne.large
      minimum: 2
      maximum: 4
      volume:
        size: 50Gi
        type: cloud_efficiency
      zones:
      - eu-central-1a
      - eu-central-1b
  networking:
    nodes: 10.250.0.0/16
    type: calico
  kubernetes:
    version: 1.16.1
  maintenance:
    autoUpdate:
      kubernetesVersion: true
      machineImageVersion: true
  addons:
    kubernetes-dashboard:
      enabled: true
    nginx-ingress:
      enabled: true
```
