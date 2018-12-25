resource sakuracloud_switch "kubernetes_internal" {
  name = "kubernetes-internal"
  tags = ["${var.other_resource_tags}"]
}

resource sakuracloud_vpc_router "vpc" {
  name = "kubernetes-vpc"
  plan = "standard"
  tags = ["${var.other_resource_tags}"]
}

resource sakuracloud_vpc_router_interface "eth1" {
  vpc_router_id = "${sakuracloud_vpc_router.vpc.id}"

  index       = 1
  switch_id   = "${sakuracloud_switch.kubernetes_internal.id}"
  ipaddress   = ["${local.vpc_router_internal_ip}"]
  nw_mask_len = 16
}

resource sakuracloud_vpc_router_static_route "worker_pod_network_routes" {
  vpc_router_id           = "${sakuracloud_vpc_router.vpc.id}"
  vpc_router_interface_id = "${sakuracloud_vpc_router_interface.eth1.id}"
  prefix                  = "${cidrsubnet(local.pod_cidr, 8, local.worker_ip_start_index + count.index)}"
  next_hop                = "${cidrhost(local.kube_internal_cidr, local.worker_ip_start_index + count.index)}"

  count = "${local.worker_node_count}"
}

resource sakuracloud_vpc_router_static_route "master_pod_network_routes" {
  vpc_router_id           = "${sakuracloud_vpc_router.vpc.id}"
  vpc_router_interface_id = "${sakuracloud_vpc_router_interface.eth1.id}"
  prefix                  = "${cidrsubnet(local.pod_cidr, 8, local.master_ip_start_index + count.index)}"
  next_hop                = "${cidrhost(local.kube_internal_cidr, local.master_ip_start_index + count.index)}"

  count = "${local.master_node_count}"
}

resource sakuracloud_vpc_router_l2tp "l2tp" {
  vpc_router_id           = "${sakuracloud_vpc_router.vpc.id}"
  vpc_router_interface_id = "${sakuracloud_vpc_router_interface.eth1.id}"

  pre_shared_secret = "${var.vpc_pre_shared_secret}"
  range_start       = "${cidrhost(local.kube_internal_cidr, local.management_ip_start_index)}"
  range_stop        = "${cidrhost(local.kube_internal_cidr, local.management_ip_start_index+5)}"
}

resource sakuracloud_vpc_router_user "admin" {
  vpc_router_id = "${sakuracloud_vpc_router.vpc.id}"

  name     = "${var.vpc_username}"
  password = "${var.vpc_password}"
}

# ポートフォワーディング
resource sakuracloud_vpc_router_port_forwarding "forward1" {
  vpc_router_id           = "${sakuracloud_vpc_router.vpc.id}"
  vpc_router_interface_id = "${sakuracloud_vpc_router_interface.eth1.id}"

  protocol        = "tcp"
  global_port     = "${10022 + count.index}"
  private_address = "${cidrhost(local.kube_internal_cidr, local.master_ip_start_index + count.index)}"
  private_port    = 22
  count           = "${local.master_count}"
}
