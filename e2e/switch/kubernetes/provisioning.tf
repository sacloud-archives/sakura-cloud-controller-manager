#################################
# Startup-Script
#################################
data template_file "master_provisioning" {
  template = "${file("${path.module}/templates/provisioning.tpl")}"

  vars {
    startup_script_headers = "${data.template_file.master_script_header.*.rendered[count.index]}"
    node_prepare           = "${data.template_file.master_pod_network_script.*.rendered[count.index]}"
    kubeadm_prepare        = "${data.template_file.kubeadm_prepare_master.rendered}"
    kubeadm_action         = "${data.template_file.kubeadm_init.*.rendered[count.index]}"
  }

  count = "${local.master_node_count}"
}

data template_file "worker_provisioning" {
  template = "${file("${path.module}/templates/provisioning.tpl")}"

  vars {
    startup_script_headers = "${data.template_file.worker_script_header.*.rendered[count.index]}"
    node_prepare           = "${data.template_file.worker_pod_network_script.*.rendered[count.index]}"
    kubeadm_prepare        = "${data.template_file.kubeadm_prepare_worker.rendered}"
    kubeadm_action         = "${data.template_file.kubeadm_join.*.rendered[count.index]}"
  }

  count = "${local.worker_node_count}"
}

#################################
# Script Headers
#################################
data template_file "master_script_header" {
  template = "${file("${path.module}/templates/startup_script_headers.tpl")}"

  vars {
    name        = "kubernetes-master-provision-${count.index}}"
    once        = true
    description = "Kubernetes master provisioning"
  }

  count = "${local.master_node_count}"
}

data template_file "worker_script_header" {
  template = "${file("${path.module}/templates/startup_script_headers.tpl")}"

  vars {
    name        = "kubernetes-worker-provision-${count.index}}"
    once        = true
    description = "Kubernetes worker provisioning"
  }

  count = "${local.worker_node_count}"
}

#################################
# pod network
#################################
data template_file "master_pod_network_script" {
  template = "${file("${path.module}/templates/pod_network.tpl")}"

  vars {
    ip           = "${cidrhost(local.kube_internal_cidr, local.master_ip_start_index + count.index)}"
    mask         = "16"
    gateway      = "${local.vpc_router_internal_ip}"
    pod_cidr     = "${local.pod_cidr}"
    service_cidr = "${local.service_cidr}"
  }

  count = "${local.master_node_count}"
}

data template_file "worker_pod_network_script" {
  template = "${file("${path.module}/templates/pod_network.tpl")}"

  vars {
    ip           = "${cidrhost(local.kube_internal_cidr, local.worker_ip_start_index + count.index)}"
    mask         = "16"
    gateway      = "${local.vpc_router_internal_ip}"
    pod_cidr     = "${local.pod_cidr}"
    service_cidr = "${local.service_cidr}"
  }

  count = "${local.worker_node_count}"
}

data template_file "kubeadm_prepare_master" {
  template = "${file("${path.module}/templates/kubeadm_prepare_master.tpl")}"

  vars {
    kubernetes_version = "${local.kubernetes_version_with_prefix}"
  }
}

data template_file "kubeadm_prepare_worker" {
  template = "${file("${path.module}/templates/kubeadm_prepare_worker.tpl")}"

  vars {
    service_node_port_range = "${replace(var.service_node_port_range, "-",":")}"
    kubernetes_version      = "${local.kubernetes_version_with_prefix}"
  }
}

##################################
# kubeadm init
##################################
data template_file "kubeadm_init" {
  template = "${file("${path.module}/templates/kubeadm_init.tpl")}"

  vars {
    token                   = "${local.token}"
    pod_cidr                = "${cidrsubnet(local.pod_cidr, 8, local.master_ip_start_index + count.index)}"
    enable_master_isolation = "${local.enable_master_isolation}"
    service_cidr            = "${local.service_cidr}"
    service_node_port_range = "${replace(var.service_node_port_range, ":","-")}"
    cloud_provider          = "${local.cloud_provider}"
    kubernetes_version      = "${var.kubernetes_version}"
  }

  count = "${local.master_node_count}"
}

##################################
# kubeadm join
##################################
data template_file "kubeadm_join" {
  template = "${file("${path.module}/templates/kubeadm_join.tpl")}"

  vars {
    token          = "${local.token}"
    master_url     = "${cidrhost(local.kube_internal_cidr, local.master_ip_start_index)}:6443"
    pod_cidr       = "${cidrsubnet(local.pod_cidr, 8, local.worker_ip_start_index + count.index)}"
    cloud_provider = "${local.cloud_provider}"
  }

  count = "${local.worker_node_count}"
}
