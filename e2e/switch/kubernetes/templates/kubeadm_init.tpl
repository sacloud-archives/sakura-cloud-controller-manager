# Initialize cluster by kubeadm
cat > /tmp/kubeadm-extra-params.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1beta1
kind: InitConfiguration
bootstrapTokens:
- groups:
  token: ${token}
nodeRegistration:
  kubeletExtraArgs:
    cloud-provider: ${cloud_provider}
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
kubernetesVersion: "${kubernetes_version}"
networking:
  serviceSubnet: ${service_cidr}
apiServer:
  extraArgs:
    service-node-port-range: ${service_node_port_range}
EOF

kubeadm init --config /tmp/kubeadm-extra-params.yaml

mkdir -p /root/.kube
sudo cp -i /etc/kubernetes/admin.conf /root/.kube/config
sudo chown $(id -u):$(id -g) /root/.kube/config
export KUBECONFIG=/etc/kubernetes/admin.conf

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

# create ClusterRoleBinding for helm
kubectl create clusterrolebinding tiller-cluster-admin \
    --clusterrole=cluster-admin \
    --serviceaccount=kube-system:default

# master isolation(for develop)
if [ "${enable_master_isolation}" != "1" ]; then
  kubectl taint nodes --all node-role.kubernetes.io/master-
fi