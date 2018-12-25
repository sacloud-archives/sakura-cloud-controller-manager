locals {
  token = "${var.kubeadm_token == "" ? data.template_file.kubeadm_token.rendered : var.kubeadm_token}"
}

resource "random_shuffle" "part1" {
  input        = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "a", "b", "c", "d", "e", "f", "g", "h", "i", "t", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"]
  result_count = 6
}

resource "random_shuffle" "part2" {
  input        = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "a", "b", "c", "d", "e", "f", "g", "h", "i", "t", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"]
  result_count = 16
}

locals {
  tokens = [
    "${join("", random_shuffle.part1.result)}",
    "${join("", random_shuffle.part2.result)}",
  ]
}

/******************************************************************************
 * kubeadm token
 *****************************************************************************/
data "template_file" "kubeadm_token" {
  template = "$${token}"

  vars {
    token = "${join(".",local.tokens)}"
  }
}
