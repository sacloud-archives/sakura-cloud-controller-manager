module "kubernetes" {
  source             = "./kubernetes"
  kubernetes_version = "1.12.3"

  password              = "${var.password}"
  vpc_pre_shared_secret = "${var.pre_shared_secret}"
  vpc_username          = "${var.vpn_username}"
  vpc_password          = "${var.vpn_password}"
  worker_count          = 2
  use_cloud_provider    = true
}

variable password {
  type = "string"
}

variable vpn_username {
  type    = "string"
  default = "admin"
}

variable vpn_password {
  type = "string"
}

variable pre_shared_secret {
  type = "string"
}
