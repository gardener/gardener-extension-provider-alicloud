<p>Packages:</p>
<ul>
<li>
<a href="#alicloud.provider.extensions.config.gardener.cloud%2fv1alpha1">alicloud.provider.extensions.config.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="alicloud.provider.extensions.config.gardener.cloud/v1alpha1">alicloud.provider.extensions.config.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="csi">CSI
</h3>


<p>
(<em>Appears on:</em><a href="#controllerconfiguration">ControllerConfiguration</a>)
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
<p>EnableADController enables disks to be attached/detached from csi-provisioner<br />Deprecated</p>
</td>
</tr>

</tbody>
</table>


<h3 id="controllerconfiguration">ControllerConfiguration
</h3>


<p>
ControllerConfiguration defines the configuration for the Alicloud provider.
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
<code>clientConnection</code></br>
<em>
<a href="#clientconnectionconfiguration">ClientConnectionConfiguration</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClientConnection specifies the kubeconfig file and client connection<br />settings for the proxy server to use when communicating with the apiserver.</p>
</td>
</tr>
<tr>
<td>
<code>machineImageOwnerSecretRef</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#secretreference-v1-core">SecretReference</a>
</em>
</td>
<td>
<p>MachineImageOwnerSecretRef is the secret reference which contains credential of AliCloud subaccount for customized images.<br />We currently assume multiple customized images should always be under this account.</p>
</td>
</tr>
<tr>
<td>
<code>toBeSharedImageIDs</code></br>
<em>
string array
</em>
</td>
<td>
<p>ToBeSharedImageIDs specifies custom image IDs which need to be shared by shoots</p>
</td>
</tr>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#service">Service</a>
</em>
</td>
<td>
<p>Service is the service configuration</p>
</td>
</tr>
<tr>
<td>
<code>etcd</code></br>
<em>
<a href="#etcd">ETCD</a>
</em>
</td>
<td>
<p>ETCD is the etcd configuration.</p>
</td>
</tr>
<tr>
<td>
<code>healthCheckConfig</code></br>
<em>
<a href="#healthcheckconfig">HealthCheckConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HealthCheckConfig is the config for the health check controller</p>
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
<p>CSI is the config for CSI plugin components</p>
</td>
</tr>

</tbody>
</table>


<h3 id="etcd">ETCD
</h3>


<p>
(<em>Appears on:</em><a href="#controllerconfiguration">ControllerConfiguration</a>)
</p>

<p>
ETCD is an etcd configuration.
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
<code>storage</code></br>
<em>
<a href="#etcdstorage">ETCDStorage</a>
</em>
</td>
<td>
<p>ETCDStorage is the etcd storage configuration.</p>
</td>
</tr>
<tr>
<td>
<code>backup</code></br>
<em>
<a href="#etcdbackup">ETCDBackup</a>
</em>
</td>
<td>
<p>ETCDBackup is the etcd backup configuration.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="etcdbackup">ETCDBackup
</h3>


<p>
(<em>Appears on:</em><a href="#etcd">ETCD</a>)
</p>

<p>
ETCDBackup is an etcd backup configuration.
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
<code>schedule</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Schedule is the etcd backup schedule.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="etcdstorage">ETCDStorage
</h3>


<p>
(<em>Appears on:</em><a href="#etcd">ETCD</a>)
</p>

<p>
ETCDStorage is an etcd storage configuration.
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
<code>className</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClassName is the name of the storage class used in etcd-main volume claims.</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#quantity-resource-api">Quantity</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Capacity is the storage capacity used in etcd-main volume claims.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="service">Service
</h3>


<p>
(<em>Appears on:</em><a href="#controllerconfiguration">ControllerConfiguration</a>)
</p>

<p>
Service is a load balancer service configuration.
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
<code>backendLoadBalancerSpec</code></br>
<em>
string
</em>
</td>
<td>
<p>BackendLoadBalancerSpec specifies the type of backend Alicloud load balancer, default is slb.s1.small.</p>
</td>
</tr>

</tbody>
</table>


