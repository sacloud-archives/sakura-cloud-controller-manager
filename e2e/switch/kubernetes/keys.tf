/******************************************************************************
 * for ssh to nodes
 *****************************************************************************/
resource "tls_private_key" "ssh_key" {
  algorithm = "RSA"
}

resource "local_file" "ssh_key" {
  content  = "${tls_private_key.ssh_key.private_key_pem}"
  filename = "${path.root}/id_rsa"
}
