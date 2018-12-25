cat > /tmp/kubeadm-extra-params.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1beta1
kind: JoinConfiguration
nodeRegistration:
  kubeletExtraArgs:
    cloud-provider: ${cloud_provider}
discovery:
  bootstrapToken:
    token: ${token}
    unsafeSkipCAVerification: true
    apiServerEndpoint: "${master_url}"
EOF

# join
kubeadm join --config /tmp/kubeadm-extra-params.yaml

# Setup CNI plugin(bridge)
mkdir -p /etc/cni/net.d
cat <<EOF > /etc/cni/net.d/10-cbr0.conf
{
	"name": "cbr0",
	"type": "bridge",
	"bridge": "cbr0",
	"isDefaultGateway": true,
	"forceAddress": false,
	"ipMasq": true,
	"ipam": {
		"type": "host-local",
        "ranges": [
          [{"subnet": "${pod_cidr}"}]
        ],
        "routes": [{"dst": "0.0.0.0/0"}]
	}
}
EOF
cat >/etc/cni/net.d/99-loopback.conf <<EOF
{
	"type": "loopback"
}
EOF

