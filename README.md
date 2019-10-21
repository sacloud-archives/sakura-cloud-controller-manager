# sakura-cloud-controller-manager

[![Go Report Card](https://goreportcard.com/badge/github.com/sacloud/sakura-cloud-controller-manager)](https://goreportcard.com/report/github.com/sacloud/sakura-cloud-controller-manager)
[![Build Status](https://travis-ci.org/sacloud/sakura-cloud-controller-manager.svg?branch=master)](https://travis-ci.org/sacloud/sakura-cloud-controller-manager)

`sakura-cloud-controller-manager` is the Kubernetes cloud controller manager implementation for the [SAKURA Cloud](https://cloud.sakura.ad.jp/).

> [About Kubernetes cloud controller managers](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)

## Features

#### NodeController

Updates nodes with cloud provider specific labels and addresses, also deletes kubernetes nodes when deleted on the cloud provider.

#### ServiceController

Responsible for creating LoadBalancers when a service of `Type: LoadBalancer` is created in Kubernetes.  
Using SAKURA Cloud's LoadBalancer appliance.

## Requirements

At the current state of Kubernetes, running cloud controller manager requires a few things.  
Please read through the requirements carefully as they are critical to running cloud controller manager on a Kubernetes cluster on SAKURA Cloud.

### --cloud-provider=external
All `kubelet`s in your cluster **MUST** set the flag `--cloud-provider=external`.
`kube-apiserver` and `kube-controller-manager` must **NOT** set the flag `--cloud-provider` 
which will default them to use no cloud provider natively.

**WARNING**: 
Setting `--cloud-provider=external` will taint all nodes in a cluster with `node.cloudprovider.kubernetes.io/uninitialized`, 
it is the responsibility of cloud controller managers to untaint those nodes once it has finished initializing them.
This means that most pods will be left unscheduable until the cloud controller manager is running.

In the future, `--cloud-provider=external` will be the default. 
Learn more about the future of cloud providers in Kubernetes [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/cloud-provider/cloud-provider-refactoring.md).

### Kubernetes node names must match the server name

By default, the kubelet will name nodes based on the node's hostname. 
If you decide to override the hostname on kubelets with `--hostname-override`, this will also override the node name in Kubernetes.
It is important that the node name on Kubernetes matches either the server name, otherwise cloud controller manager cannot find the corresponding server to nodes.

### All servers must have unique names

All server names in kubernetes must be unique since node names in kubernetes must be unique.

## Requirements(only when using LoadBalancer)

If you want to use service with `type: LoadBalancer`, the following settings are required.

- All workers must connected to under Switch or Switch+Router tagged with `@k8s` tag.
  (This setting can change by annotations. see [#LoadBalancer Service Annotations](#loadbalancer-service-annotations) section.)

## Compatibility for Kubernetes and CCM

| Kubernetes | sakura-cloud-controller-manager | 
| ------- | -------- |
|  v1.13  |  v0.3, v0.4+  |
|  v1.14  |  v0.4+  |
|  v1.15  |  v0.4+  |
|  v1.16  |  v0.4+  |

## Deploy

### API Key

To running `sakura-cloud-controller-manager`, you need SAKURA Cloud API Key.  
Please create API Key from [Control Panel](https://secure.sakura.ad.jp/cloud/) if you haven't it.

Then, create the Secret resource by followings:

```bash
# set API keys to env
export SAKURACLOUD_ACCESS_TOKEN=<your-token>
export SAKURACLOUD_ACCESS_TOKEN_SECRET=<your-secret>
export SAKURACLOUD_ZONE=<your-zone> # is1a or is1b or tk1a

# create Secret resource
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: ccm-api-token
  namespace: kube-system
type: Opaque
data:
  token: '$(echo -n $SAKURACLOUD_ACCESS_TOKEN | base64)'
  secret: '$(echo -n $SAKURACLOUD_ACCESS_TOKEN_SECRET | base64)'
  zone: '$(echo -n $SAKURACLOUD_ZONE | base64)'
EOF

```

### Deploy `sakura-cloud-controller-manager` 

```bash
kubectl apply -f https://raw.githubusercontent.com/sacloud/sakura-cloud-controller-manager/0.4.1/manifests/cloud-controller-manager.yaml
```

## Usage (with Router+Switch)

Example for service with Router+Switch and `type:LoadBalancer`:

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    run: load-balancer-example
  name: hello-world
  namespace: default
spec:
  ports:
    - port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    run: load-balancer-example
  type: LoadBalancer
```

To see full manifests, see [examples/services/with-router.yaml](examples/services/with-router.yaml).

## Usage (with Switch)

Example for service with Switch and `type:LoadBalancer`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: load-balancer-example
  namespace: default
  labels:
    run: load-balancer-example
  annotations:
    k8s.usacloud.jp/load-balancer-type: "switch"
    k8s.usacloud.jp/load-balancer-ip-range: "10.240.0.0/16"
    k8s.usacloud.jp/load-balancer-assign-ip-range: "10.240.100.0/24"
    k8s.usacloud.jp/load-balancer-assign-default-gateway: "10.240.0.1"
spec:
  ports:
    - port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    run: load-balancer-example
  type: LoadBalancer
```

To see full manifests, see [examples/services/with-switch.yaml](examples/services/with-switch.yaml).

## Usage (others)

- [Use loadBalancerIP example](examples/services/with-switch-loadBalancerIP.yaml)
- [High-Availability LoadBalancer example](examples/services/with-switch-HA.yaml)

## LoadBalancer Service Annotations

`sakura-cloud-controller-manager` supports annotations as follows:

#### LoadBalancer's settings

- `k8s.usacloud.jp/load-balancer-type`: (optional) LoadBalancer type. Options are `internet` and `switch`. Default is `internet`.  
- `k8s.usacloud.jp/load-balancer-ha`: (optional) Flag of use High-Availability LoadBalancer. Default is `false`  
- `k8s.usacloud.jp/load-balancer-plan`: (optional) LoadBalancer Plan. Options are `standard` and `premium`. Default is `standard`  
- `k8s.usacloud.jp/load-balancer-healthz-interval`: (optional) Interval seconds to check real-server's health. Default is `10`  

#### Router+Switch or Switch Selector settings

- `k8s.usacloud.jp/router-selector`: (optional) Additional tags for finding upstream Router+Switch. Default is `[]`  
This annotation only used when `k8s.usacloud.jp/load-balancer-type` is set to `internet`.  
- `k8s.usacloud.jp/switch-selector`: (optional) Additional tags for finding upstream Switch. Default is `[]`  
This annotation only used when `k8s.usacloud.jp/load-balancer-type` is set to `switch`.  

#### LoadBalancer's switched network settings

There annotations are required when `k8s.usacloud.jp/load-balancer-type` is set to `switch`.

- `k8s.usacloud.jp/load-balancer-ip-range`: IP address range for calculate LoadBalancer's IP/VIP network mask length. CIDR format(`192.2.0.1/24`) required.
- `k8s.usacloud.jp/load-balancer-assign-ip-range`: IP address range for assign LoadBalancer's IP/VIP. CIDR format(`192.2.0.1/24`) required.
- `k8s.usacloud.jp/load-balancer-assign-default-gateway`: Default gateway address for assign LoadBalancer.

## License

 `sakura-cloud-controller-manager` Copyright (C) 2018-2019 Kazumichi Yamamoto.

  This project is published under [Apache 2.0 License](LICENSE.txt).
  
## Author

  * Kazumichi Yamamoto ([@yamamoto-febc](https://github.com/yamamoto-febc))
