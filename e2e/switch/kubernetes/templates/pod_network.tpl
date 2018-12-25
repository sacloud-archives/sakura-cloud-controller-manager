yum -y install bridge-utils

#nmcli c modify "System eth1"  \
#        ipv4.method manual \
#        ipv4.addresses "${ip}/${mask}" \
#        ipv4.gateway "${gateway}" \
#        +ipv4.routes "${pod_cidr} ${gateway}" \
#        +ipv4.routes "${service_cidr}"
#systemctl restart NetworkManager
#systemctl restart network
