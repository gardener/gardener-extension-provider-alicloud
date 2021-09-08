provider "alicloud" {
  access_key = var.ACCESS_KEY_ID
  secret_key = var.ACCESS_KEY_SECRET
  region = "{{ .alicloud.region }}"
}

{{ if .vpc.create -}}
resource "alicloud_vpc" "vpc" {
  vpc_name   = "{{ .clusterName }}-vpc"
  cidr_block = "{{ .vpc.cidr }}"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}

resource "alicloud_nat_gateway" "nat_gateway" {
  vpc_id            = {{ .vpc.id }}
  specification     = "Small"
  name              = "{{ .clusterName }}-natgw"
  nat_type          = "Enhanced"
  vswitch_id        = alicloud_vswitch.vsw_z0.id
  depends_on        = [alicloud_vswitch.vsw_z0]
}
{{- end }}

// Loop zones
{{- range $index, $zone := .zones }}
resource "alicloud_vswitch" "vsw_z{{ $index }}" {
  vswitch_name      = "{{ $.clusterName }}-{{ $zone.name }}-vsw"
  vpc_id            = {{ $.vpc.id }}
  cidr_block        = "{{ $zone.cidr.workers }}"
  zone_id           = "{{ $zone.name }}"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}

{{ if $zone.eipAllocationID -}}
// specify EIP ID
data "alicloud_eips" "eip_natgw_ds{{ $index }}" {
  ids = ["{{ $zone.eipAllocationID }}"]
}

resource "alicloud_eip_association" "eip_natgw_asso_z{{ $index }}" {
  allocation_id = data.alicloud_eips.eip_natgw_ds{{ $index }}.eips.0.id
  instance_id   = {{ $.natGateway.id }}
}

resource "alicloud_snat_entry" "snat_z{{ $index }}" {
  snat_table_id     = {{ $.natGateway.sNatTableIDs }}
  source_vswitch_id = alicloud_vswitch.vsw_z{{ $index }}.id
  snat_ip           = data.alicloud_eips.eip_natgw_ds{{ $index }}.eips.0.ip_address
  depends_on        = [alicloud_eip_association.eip_natgw_asso_z{{ $index }}]
}
{{ else -}}
// Create a new EIP.
resource "alicloud_eip" "eip_natgw_z{{ $index }}" {
  name                 = "{{ $.clusterName }}-eip-natgw-z{{ $index }}"
  bandwidth            = "100"
  instance_charge_type = "PostPaid"
  internet_charge_type = "{{ $.eip.internetChargeType }}"
}

resource "alicloud_eip_association" "eip_natgw_asso_z{{ $index }}" {
  allocation_id = alicloud_eip.eip_natgw_z{{ $index }}.id
  instance_id   = {{ $.natGateway.id }}
}

resource "alicloud_snat_entry" "snat_z{{ $index }}" {
  snat_table_id     = {{ $.natGateway.sNatTableIDs }}
  source_vswitch_id = alicloud_vswitch.vsw_z{{ $index }}.id
  snat_ip           = alicloud_eip.eip_natgw_z{{ $index }}.ip_address
  depends_on        = [alicloud_eip_association.eip_natgw_asso_z{{ $index }}]
}
{{- end }}

// Output
output "{{ $.outputKeys.vswitchNodesPrefix }}{{ $index }}" {
  value = alicloud_vswitch.vsw_z{{ $index }}.id
}

{{ end }}
// End of loop zones

resource "alicloud_security_group" "sg" {
  name   = "{{ .clusterName }}-sg"
  vpc_id = {{ .vpc.id }}
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
  cidr_ip           = "{{ .vpc.cidr }}"
}

resource "alicloud_security_group_rule" "allow_all_internal_udp_in" {
  type              = "ingress"
  ip_protocol       = "udp"
  policy            = "accept"
  port_range        = "1/65535"
  priority          = 1
  security_group_id = alicloud_security_group.sg.id
  cidr_ip           = "{{ .vpc.cidr }}"
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

output "{{ .outputKeys.securityGroupID }}" {
  value = alicloud_security_group.sg.id
}

output "{{ .outputKeys.vpcID }}" {
  value = {{ .vpc.id }}
}

output "{{ .outputKeys.vpcCIDR }}" {
  value = "{{ .vpc.cidr }}"
}
