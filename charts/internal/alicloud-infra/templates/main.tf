provider "alicloud" {
  access_key = var.ACCESS_KEY_ID
  secret_key = var.ACCESS_KEY_SECRET
  region = "{{ required "alicloud.region is required" .Values.alicloud.region }}"
}

// Import an existing public key to build a alicloud key pair
resource "alicloud_key_pair" "publickey" {
  key_name = "{{ required "clusterName is required" .Values.clusterName }}-ssh-publickey"
  public_key = "{{ required "sshPublicKey is required" .Values.sshPublicKey }}"
}

{{ if .Values.vpc.create -}}
resource "alicloud_vpc" "vpc" {
  name       = "{{ required "clusterName is required" .Values.clusterName }}-vpc"
  cidr_block = "{{ required "vpc.cidr is required" .Values.vpc.cidr }}"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}

resource "alicloud_nat_gateway" "nat_gateway" {
  vpc_id            = {{ required "vpc.id is required" .Values.vpc.id }}
  specification     = "Small"
  name              = "{{ required "clusterName is required" .Values.clusterName }}-natgw"
  nat_type          = "Enhanced"
  vswitch_id        = alicloud_vswitch.vsw_z0.id
  depends_on        = [alicloud_vswitch.vsw_z0]
}
{{- end }}

// Loop zones
{{- range $index, $zone := .Values.zones }}
resource "alicloud_vswitch" "vsw_z{{ $index }}" {
  name              = "{{ required "clusterName is required" $.Values.clusterName }}-{{ required "zone.name is required" $zone.name }}-vsw"
  vpc_id            = {{ required "vpc.id is required" $.Values.vpc.id }}
  cidr_block        = "{{ required "zone.cidr.workers is required" $zone.cidr.workers }}"
  availability_zone = "{{ required "zone.name is required" $zone.name }}"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}

{{ if $zone.eipAllocationID -}}
// specify EIP ID
data "alicloud_eips" "eip_natgw_ds{{ $index }}" {
  ids = ["{{ required "$zone.eipAllocationID is required" $zone.eipAllocationID }}"]
}

resource "alicloud_eip_association" "eip_natgw_asso_z{{ $index }}" {
  allocation_id = data.alicloud_eips.eip_natgw_ds{{ $index }}.eips.0.id
  instance_id   = {{ required "natGateway.id is required" $.Values.natGateway.id }}
}

resource "alicloud_snat_entry" "snat_z{{ $index }}" {
  snat_table_id     = {{ required "natGateway.sNatTableIDs is required" $.Values.natGateway.sNatTableIDs }}
  source_vswitch_id = alicloud_vswitch.vsw_z{{ $index }}.id
  snat_ip           = data.alicloud_eips.eip_natgw_ds{{ $index }}.eips.0.ip_address
  depends_on        = [alicloud_eip_association.eip_natgw_asso_z{{ $index }}]
}
{{ else -}}
// Create a new EIP.
resource "alicloud_eip" "eip_natgw_z{{ $index }}" {
  name                 = "{{ required "clusterName is required" $.Values.clusterName }}-eip-natgw-z{{ $index }}"
  bandwidth            = "100"
  instance_charge_type = "PostPaid"
  internet_charge_type = "{{ required "eip.internetChargeType is required" $.Values.eip.internetChargeType }}"
}

resource "alicloud_eip_association" "eip_natgw_asso_z{{ $index }}" {
  allocation_id = alicloud_eip.eip_natgw_z{{ $index }}.id
  instance_id   = {{ required "natGateway.id is required" $.Values.natGateway.id }}
}

resource "alicloud_snat_entry" "snat_z{{ $index }}" {
  snat_table_id     = {{ required "natGateway.sNatTableIDs is required" $.Values.natGateway.sNatTableIDs }}
  source_vswitch_id = alicloud_vswitch.vsw_z{{ $index }}.id
  snat_ip           = alicloud_eip.eip_natgw_z{{ $index }}.ip_address
  depends_on        = [alicloud_eip_association.eip_natgw_asso_z{{ $index }}]
}
{{- end }}

// Output
output "{{ $.Values.outputKeys.vswitchNodesPrefix }}{{ $index }}" {
  value = alicloud_vswitch.vsw_z{{ $index }}.id
}

{{ end }}
// End of loop zones

resource "alicloud_security_group" "sg" {
  name   = "{{ required "clusterName is required" .Values.clusterName }}-sg"
  vpc_id = {{ required "vpc.id is required" .Values.vpc.id }}
}

resource "alicloud_security_group_rule" "allow_k8s_tcp_in" {
  type              = "ingress"
  ip_protocol       = "tcp"
  policy            = "accept"
  port_range        = "30000/32767"
  priority          = 1
  security_group_id = alicloud_security_group.sg.id
  cidr_ip           = "0.0.0.0/0"
}

resource "alicloud_security_group_rule" "allow_all_internal_tcp_in" {
  type              = "ingress"
  ip_protocol       = "tcp"
  policy            = "accept"
  port_range        = "1/65535"
  priority          = 1
  security_group_id = alicloud_security_group.sg.id
  cidr_ip           = "{{ required "vpc.cidr is required" .Values.vpc.cidr }}"
}

resource "alicloud_security_group_rule" "allow_all_internal_udp_in" {
  type              = "ingress"
  ip_protocol       = "udp"
  policy            = "accept"
  port_range        = "1/65535"
  priority          = 1
  security_group_id = alicloud_security_group.sg.id
  cidr_ip           = "{{ required "vpc.cidr is required" .Values.vpc.cidr }}"
}

// We have introduced new output variables. However, they are not applied for
// existing clusters as Terraform won't detect a diff when we run `terraform plan`.
// Workaround: Providing a null-resource for letting Terraform think that there are
// differences, enabling the Gardener to start an actual `terraform apply` job.
resource "null_resource" "outputs" {
  triggers = {
    recompute = "outputs"
  }
}

//=====================================================================
//= Output variables
//=====================================================================

output "{{ .Values.outputKeys.securityGroupID }}" {
  value = alicloud_security_group.sg.id
}

output "{{ .Values.outputKeys.vpcID }}" {
  value = {{ required "vpc.id is required" .Values.vpc.id }}
}

output "{{ .Values.outputKeys.vpcCIDR }}" {
  value = "{{ required "vpc.cidr is required" .Values.vpc.cidr }}"
}

output "{{ .Values.outputKeys.keyPairName }}" {
  value = alicloud_key_pair.publickey.key_name
}
