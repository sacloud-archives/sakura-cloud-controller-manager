# Disable selinux
setenforce 0
cat << EOF > /etc/selinux/config
SELINUX=disabled
SELINUXTYPE=targeted
EOF

# Disable swap
swapoff -a
sudo sed -i '/ swap / s/^/#/' /etc/fstab

# Open ports
yum install -y iptables-services
systemctl stop firewalld.service
systemctl disable firewalld.service
systemctl mask firewalld.service
systemctl start iptables
systemctl enable iptables
systemctl unmask iptables
iptables -F
iptables -t filter -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -t filter -A INPUT -p icmp -j ACCEPT
iptables -t filter -A INPUT -i lo -j ACCEPT
iptables -t filter -A INPUT -m state --state NEW -m tcp -p tcp --dport 22 -j ACCEPT
iptables -t filter -A INPUT -m tcp -p tcp --dport 10250 -j ACCEPT
iptables -t filter -A INPUT -m tcp -p tcp --dport 10255 -j ACCEPT
iptables -t filter -A INPUT -m tcp -p tcp --dport ${service_node_port_range} -j ACCEPT
iptables -t filter -A INPUT -j REJECT --reject-with icmp-host-prohibited
iptables -t filter -A FORWARD -o cbr0 -j ACCEPT
iptables -t filter -A FORWARD -o cbr0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -t filter -A FORWARD -i cbr0 ! -o cbr0 -j ACCEPT
iptables -t filter -A FORWARD -i cbr0 -o cbr0 -j ACCEPT
iptables -t filter -A FORWARD -o eth0 -j ACCEPT
iptables -t filter -A FORWARD -o eth0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -t filter -A FORWARD -i eth0 ! -o eth0 -j ACCEPT
iptables -t filter -A FORWARD -i eth0 -o eth0 -j ACCEPT
service iptables save

# Install docker
yum install -y docker
systemctl enable docker && systemctl start docker

# Install kubelet/kubeadm/kubectl
cat <<EOF >  /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
sysctl --system

cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF

yum install -y kubelet${kubernetes_version} kubeadm kubectl
systemctl enable kubelet && systemctl start kubelet
