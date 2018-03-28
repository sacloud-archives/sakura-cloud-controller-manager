# sakura-cloud-controller-manager

[![Go Report Card](https://goreportcard.com/badge/github.com/sacloud/sakura-cloud-controller-manager)](https://goreportcard.com/report/github.com/sacloud/sakura-cloud-controller-manager)
[![Build Status](https://travis-ci.org/sacloud/sakura-cloud-controller-manager.svg?branch=master)](https://travis-ci.org/sacloud/sakura-cloud-controller-manager)

`sakura-cloud-controller-manager` is the Kubernetes cloud controller manager implementation for the [SAKURA Cloud](https://cloud.sakura.ad.jp/).

> [About Kubernetes cloud controller managers](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)

**this project is a work in progress and may not be production ready.**

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

- All workers must connected to under Switch+Router tagged with `@k8s` tag.
- All workers must have kernel parameters enabled for DSR load balancing

To set the kernel parameters for DSR load balancing, do as follows:

    # add following lines to /etc/sysctl.conf
    net.ipv4.conf.all.arp_ignore = 1
    net.ipv4.conf.all.arp_announce = 2
    # reload 
    sysctl -p

## Deploy

### API Key

To running `sakura-cloud-controller-manager`, you need SAKURA Cloud API Key.  
Please create API Key from [Control Panel](https://secure.sakura.ad.jp/cloud/) if you haven't it.

### Run `sakura-cloud-controller-manager` container

There are two ways to run `sakura-cloud-controller-manager`

- using `helm` 
- manually settting up

### Using `helm`

To deploy by helm, see [sacloud/helm-charts/sakura-cloud-controller-mammager](https://github.com/sacloud/helm-charts/blob/master/sakura-cloud-controller-manager/README.md)

### Manually 

[TODO add documents]

## License

 `sakura-cloud-controller-manager` Copyright (C) 2018 Kazumichi Yamamoto.

  This project is published under [Apache 2.0 License](LICENSE.txt).
  
## Author

  * Kazumichi Yamamoto ([@yamamoto-febc](https://github.com/yamamoto-febc))