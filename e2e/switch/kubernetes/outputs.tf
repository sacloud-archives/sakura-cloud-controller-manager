output "ip_address_masters" {
  description = "Global IP address of master node"
  value       = "${sakuracloud_server.masters.*.ipaddress}"
}

output "ip_address_workers" {
  description = "Global IP addresses of worker nodes"
  value       = "${sakuracloud_server.workers.*.ipaddress}"
}

output "vpc_switch_id" {
  value = "${sakuracloud_switch.kubernetes_internal.id}"
}

output "vpc_router_id" {
  value = "${sakuracloud_vpc_router.vpc.id}"
}

output "vpc_router_external_ip" {
  value = "${sakuracloud_vpc_router.vpc.global_address}"
}

output "vpc_router_internal_ip" {
  value = "${sakuracloud_vpc_router_interface.eth1.ipaddress}"
}

output "vpc_internal_cidr" {
  value = "${local.kube_internal_cidr}"
}

output "pod_cidr" {
  value = "${local.pod_cidr}"
}

output "service_cidr" {
  value = "${local.service_cidr}"
}

output "download_kubeconfig_command" {
  value = "usacloud --zone ${sakuracloud_vpc_router.vpc.zone} server scp -y -i certs/id_rsa ${local.master_node_name_prefix}01:/etc/kubernetes/admin.conf admin.conf\nexport KUBECONFIG=${path.cwd}admin.conf"
}

output "ssh_private_key" {
  sensitive = true
  value     = "${tls_private_key.ssh_key.private_key_pem}"
}
