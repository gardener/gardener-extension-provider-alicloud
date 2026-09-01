<p>Packages:</p>
<ul>
<li>
<a href="#alicloud.provider.extensions.gardener.cloud%2fv1alpha1">alicloud.provider.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="alicloud.provider.extensions.gardener.cloud/v1alpha1">alicloud.provider.extensions.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="backupbucketconfig">BackupBucketConfig
</h3>


<p>
BackupBucketConfig represents the configuration for a backup bucket.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>immutability</code></br>
<em>
<a href="#immutableconfig">ImmutableConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutability defines the immutability configuration for the backup bucket.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="csi">CSI
</h3>


<p>
(<em>Appears on:</em><a href="#controlplaneconfig">ControlPlaneConfig</a>)
</p>

<p>
CSI is csi components configuration.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>enableADController</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableADController enables disks to be attached/detached from controller server of CSI Plugin.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="cloudcontrollermanagerconfig">CloudControllerManagerConfig
</h3>


<p>
(<em>Appears on:</em><a href="#controlplaneconfig">ControlPlaneConfig</a>)
</p>

<p>
CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>featureGates</code></br>
<em>
object (keys:string, values:boolean)
</em>
</td>
<td>
<em>(Optional)</em>
<p>FeatureGates contains information about enabled feature gates.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="cloudprofileconfig">CloudProfileConfig
</h3>


<p>
CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
resource.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#machineimages">MachineImages</a> array
</em>
</td>
<td>
<p>MachineImages is the list of machine images that are understood by the controller. It maps<br />logical names and versions to provider-specific identifiers.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="controlplaneconfig">ControlPlaneConfig
</h3>


<p>
ControlPlaneConfig contains configuration settings for the control plane.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>cloudControllerManager</code></br>
<em>
<a href="#cloudcontrollermanagerconfig">CloudControllerManagerConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CloudControllerManager contains configuration settings for the cloud-controller-manager.</p>
</td>
</tr>
<tr>
<td>
<code>csi</code></br>
<em>
<a href="#csi">CSI</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CSI is the config for CSI plugin components.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="dualstack">DualStack
</h3>


<p>
(<em>Appears on:</em><a href="#infrastructureconfig">InfrastructureConfig</a>)
</p>

<p>
DualStack specifies whether dual-stack or IPv4-only should be supported.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>enabled</code></br>
<em>
boolean
</em>
</td>
<td>
<p>Enabled specifies if dual-stack is enabled or not.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="immutableconfig">ImmutableConfig
</h3>


<p>
(<em>Appears on:</em><a href="#backupbucketconfig">BackupBucketConfig</a>)
</p>

<p>
ImmutableConfig represents the immutability configuration for a backup bucket.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>retentionType</code></br>
<em>
<a href="#retentiontype">RetentionType</a>
</em>
</td>
<td>
<p>RetentionType specifies the type of retention for the backup bucket.<br />Currently allowed value is:<br />- "bucket": retention policy applies on the entire bucket.</p>
</td>
</tr>
<tr>
<td>
<code>retentionPeriod</code></br>
<em>
integer
</em>
</td>
<td>
<p>RetentionPeriod specifies the immutability retention period for the backup bucket.<br />Alicloud only supports immutability duration in days, therefore this field must be set as integer.<br />The minimum retention period is 1 day.</p>
</td>
</tr>
<tr>
<td>
<code>locked</code></br>
<em>
boolean
</em>
</td>
<td>
<p>Locked indicates whether the immutable retention policy is locked for the backup bucket.<br />If set to true, the retention policy can't be removed and retention period can't be reduced.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="infrastructureconfig">InfrastructureConfig
</h3>


<p>
InfrastructureConfig infrastructure configuration resource
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>dualStack</code></br>
<em>
<a href="#dualstack">DualStack</a>
</em>
</td>
<td>
<p>DualStack specifies whether dual-stack or IPv4-only should be supported.</p>
</td>
</tr>
<tr>
<td>
<code>networks</code></br>
<em>
<a href="#networks">Networks</a>
</em>
</td>
<td>
<p>Networks specifies the networks for an infrastructure.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="infrastructurestatus">InfrastructureStatus
</h3>


<p>
InfrastructureStatus contains information about created infrastructure resources.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>vpc</code></br>
<em>
<a href="#vpcstatus">VPCStatus</a>
</em>
</td>
<td>
<p></p>
</td>
</tr>
<tr>
<td>
<code>keyPairName</code></br>
<em>
string
</em>
</td>
<td>
<p>KeyPairName is the key pair name.<br />Deprecated: the field should not be used anymore as the SSH key pair is no longer generated by terraform.</p>
</td>
</tr>
<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#machineimage">MachineImage</a> array
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineImages is a list of machine images that have been used in this infrastructure. Usually, the extension controller<br />gets the mapping from name/version to the provider-specific machine image data in its componentconfig. However, if<br />a version that is still in use gets removed from this componentconfig and Shoot's access to the this version is revoked,<br />it cannot reconcile anymore existing `Infrastructure` resources that are still using this version. Hence, it stores<br />the used versions in the provider status to ensure reconciliation is possible.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="machineimage">MachineImage
</h3>


<p>
(<em>Appears on:</em><a href="#infrastructurestatus">InfrastructureStatus</a>, <a href="#workerstatus">WorkerStatus</a>)
</p>

<p>
MachineImage is a mapping from logical names and versions to provider-specific machine image data.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the logical name of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the logical version of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the id of the image.</p>
</td>
</tr>
<tr>
<td>
<code>encrypted</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>Encrypted is a flag to specify whether this image is encrypted or not</p>
</td>
</tr>
<tr>
<td>
<code>capabilities</code></br>
<em>
<a href="#capabilities">Capabilities</a>
</em>
</td>
<td>
<p>Capabilities of the machine image.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="machineimageflavor">MachineImageFlavor
</h3>


<p>
(<em>Appears on:</em><a href="#machineimageversion">MachineImageVersion</a>)
</p>

<p>
MachineImageFlavor groups all RegionIDMappings for a specific set of capabilities.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>regions</code></br>
<em>
<a href="#regionidmapping">RegionIDMapping</a> array
</em>
</td>
<td>
<p>Regions is a mapping to the correct ID for the machine image in the supported regions.</p>
</td>
</tr>
<tr>
<td>
<code>capabilities</code></br>
<em>
<a href="#capabilities">Capabilities</a>
</em>
</td>
<td>
<p>Capabilities that are supported by the machine image IDs in this set.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="machineimageversion">MachineImageVersion
</h3>


<p>
(<em>Appears on:</em><a href="#machineimages">MachineImages</a>)
</p>

<p>
MachineImageVersion contains a version and a provider-specific identifier.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the version of the image.</p>
</td>
</tr>
<tr>
<td>
<code>regions</code></br>
<em>
<a href="#regionidmapping">RegionIDMapping</a> array
</em>
</td>
<td>
<p>Regions is a mapping to the correct ID for the machine image in the supported regions.</p>
</td>
</tr>
<tr>
<td>
<code>capabilityFlavors</code></br>
<em>
<a href="#machineimageflavor">MachineImageFlavor</a> array
</em>
</td>
<td>
<p>CapabilityFlavors is grouping of region machine image IDs by capabilities.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="machineimages">MachineImages
</h3>


<p>
(<em>Appears on:</em><a href="#cloudprofileconfig">CloudProfileConfig</a>)
</p>

<p>
MachineImages is a mapping from logical names and versions to provider-specific identifiers.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the logical name of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>versions</code></br>
<em>
<a href="#machineimageversion">MachineImageVersion</a> array
</em>
</td>
<td>
<p>Versions contains versions and a provider-specific identifier.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="natgatewayconfig">NatGatewayConfig
</h3>


<p>
(<em>Appears on:</em><a href="#zone">Zone</a>)
</p>

<p>
NatGatewayConfig specifies configuration for the NAT gateway in this zone.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>eipAllocationID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>EIPAllocationID specifies the EIP id to bind on NatGateway.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="networks">Networks
</h3>


<p>
(<em>Appears on:</em><a href="#infrastructureconfig">InfrastructureConfig</a>)
</p>

<p>
Networks specifies the networks for an infrastructure.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>vpc</code></br>
<em>
<a href="#vpc">VPC</a>
</em>
</td>
<td>
<p>VPC contains information about whether to create a new or use an existing VPC.</p>
</td>
</tr>
<tr>
<td>
<code>zones</code></br>
<em>
<a href="#zone">Zone</a> array
</em>
</td>
<td>
<p>Zones are the network zones for an infrastructure.</p>
</td>
</tr>
<tr>
<td>
<code>nodesSecurityGroupID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>NodesSecurityGroupID optionally specifies an existing security group ID for worker nodes.<br />When set, Gardener will not create a nodes security group and will not manage any security<br />group rules. The user is fully responsible for rule correctness.<br />Requires vpc.id to be set. Immutable after creation.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="purpose">Purpose
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#securitygroup">SecurityGroup</a>, <a href="#vswitch">VSwitch</a>)
</p>

<p>
Purpose is a purpose of a subnet.
</p>


<h3 id="regionidmapping">RegionIDMapping
</h3>


<p>
(<em>Appears on:</em><a href="#machineimageflavor">MachineImageFlavor</a>, <a href="#machineimageversion">MachineImageVersion</a>)
</p>

<p>
RegionIDMapping is a mapping to the correct ID for the machine image in the given region.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the region.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the id of the image.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="retentiontype">RetentionType
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#immutableconfig">ImmutableConfig</a>)
</p>

<p>
RetentionType defines the level at which immutability properties are applied on objects.
</p>


<h3 id="securitygroup">SecurityGroup
</h3>


<p>
(<em>Appears on:</em><a href="#vpcstatus">VPCStatus</a>)
</p>

<p>
SecurityGroup contains information about a security group.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>purpose</code></br>
<em>
<a href="#purpose">Purpose</a>
</em>
</td>
<td>
<p>Purpose is the purpose for which the security group was created.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the id of the security group.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="vpc">VPC
</h3>


<p>
(<em>Appears on:</em><a href="#networks">Networks</a>)
</p>

<p>
VPC contains information about whether to create a new or use an existing VPC.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ID is the ID of an existing VPC.</p>
</td>
</tr>
<tr>
<td>
<code>cidr</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CIDR is the CIDR of a VPC to create.</p>
</td>
</tr>
<tr>
<td>
<code>gardenerManagedNATGateway</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>GardenerManagedNATGateway indicates whether Gardener should create NATGateway in the VPC.<br />This will only take effect if VPC ID is set.</p>
</td>
</tr>
<tr>
<td>
<code>useCustomRouteTable</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>UseCustomRouteTable indicates whether Gardener should create a custom route table for this shoot.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="vpcstatus">VPCStatus
</h3>


<p>
(<em>Appears on:</em><a href="#infrastructurestatus">InfrastructureStatus</a>)
</p>

<p>
VPCStatus contains output information about the VPC.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the ID of the VPC.</p>
</td>
</tr>
<tr>
<td>
<code>vswitches</code></br>
<em>
<a href="#vswitch">VSwitch</a> array
</em>
</td>
<td>
<p>VSwitches is a list of vswitches.</p>
</td>
</tr>
<tr>
<td>
<code>securityGroups</code></br>
<em>
<a href="#securitygroup">SecurityGroup</a> array
</em>
</td>
<td>
<p>SecurityGroups is a list of security groups.</p>
</td>
</tr>
<tr>
<td>
<code>routeTableID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>RouteTableID is the ID of the custom route table created for this shoot cluster.<br />This will only take effect if vpc.useCustomRouteTable is set true.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="vswitch">VSwitch
</h3>


<p>
(<em>Appears on:</em><a href="#vpcstatus">VPCStatus</a>)
</p>

<p>
VSwitch contains information about a vswitch.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>purpose</code></br>
<em>
<a href="#purpose">Purpose</a>
</em>
</td>
<td>
<p>Purpose is the purpose for which the vswitch was created.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the id of the vswitch.</p>
</td>
</tr>
<tr>
<td>
<code>zone</code></br>
<em>
string
</em>
</td>
<td>
<p>Zone is the name of the zone.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="workerstatus">WorkerStatus
</h3>


<p>
WorkerStatus contains information about created worker resources.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#machineimage">MachineImage</a> array
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineImages is a list of machine images that have been used in this worker. Usually, the extension controller<br />gets the mapping from name/version to the provider-specific machine image data in its componentconfig. However, if<br />a version that is still in use gets removed from this componentconfig it cannot reconcile anymore existing `Worker`<br />resources that are still using this version. Hence, it stores the used versions in the provider status to ensure<br />reconciliation is possible.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="zone">Zone
</h3>


<p>
(<em>Appears on:</em><a href="#networks">Networks</a>)
</p>

<p>
Zone is a zone with a name and worker CIDR.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of a zone.</p>
</td>
</tr>
<tr>
<td>
<code>worker</code></br>
<em>
string
</em>
</td>
<td>
<p>Worker specifies the worker CIDR to use.<br />Deprecated - use `workers` instead.</p>
</td>
</tr>
<tr>
<td>
<code>workers</code></br>
<em>
string
</em>
</td>
<td>
<p>Workers specifies the worker CIDR to use.<br />Mutually exclusive with WorkersVSwitchID.</p>
</td>
</tr>
<tr>
<td>
<code>workersVSwitchID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>WorkersVSwitchID specifies an existing VSwitch ID for worker nodes.<br />When set, Gardener will not create a VSwitch, NAT Gateway, EIP, or SNAT entries.<br />Mutually exclusive with Workers CIDR.<br />Requires vpc.id to be set. Immutable after creation.</p>
</td>
</tr>
<tr>
<td>
<code>ipv6CidrBlock</code></br>
<em>
integer
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ipv6CidrBlock specifies the worker ipv6 CIDR block to use 0-255.<br />This will only take effect if dualStack.enabled is true.<br />Not required when WorkersVSwitchID is set (user pre-configures IPv6 on the VSwitch).</p>
</td>
</tr>
<tr>
<td>
<code>natGateway</code></br>
<em>
<a href="#natgatewayconfig">NatGatewayConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NatGateway specifies configuration for the NAT gateway in this zone.<br />Forbidden when WorkersVSwitchID is set.</p>
</td>
</tr>

</tbody>
</table>


