package sakura

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/sakura-cloud-controller-manager/iaas"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/util/strings"
)

// TagsLoadBalancerServiceName is tag name indicating that resource is part of k8s cluster service
var TagsLoadBalancerServiceName = fmt.Sprintf("%s.Service", TagsKubernetesResource)

const (
	// annRouterSelector is the annotation used to specify the condition
	// for finding upstream Router resource.
	// default is empty.
	annRouterSelector = "k8s.usacloud.jp/router-selector"

	// annLoadBalancerHA is the annotation used to specify the flag
	// for setting LoadBalancer high-availability mode.
	annLoadBalancerHA = "k8s.usacloud.jp/load-balancer-ha"

	// annLoadBalancerPlan is the annotation used to specify the plan
	// for setting LoadBalancer plan.
	// Options are standard and premium. Defaults to standard
	annLoadBalancerPlan = "k8s.usacloud.jp/load-balancer-plan"

	// annHealthzInterval is the annotation used to specify check interval(sec)
	// for health check to real nodes.
	// Defaults to 10.
	annHealthzInterval = "k8s.usacloud.jp/load-balancer-healthz-interval"
)

var (
	errLBNotFound = errors.New("loadbalancer not found")
)

type loadbalancers struct {
	sacloudAPI iaas.Client
	config     *Config
}

// newLoadbalancers returns a cloudprovider.LoadBalancer whose concrete type is a *loadbalancer.
func newLoadbalancers(client iaas.Client, config *Config) cloudprovider.LoadBalancer {
	return &loadbalancers{
		sacloudAPI: client,
		config:     config,
	}
}

// GetLoadBalancer returns the *v1.LoadBalancerStatus of service.
//
// GetLoadBalancer will not modify service.
func (l *loadbalancers) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {

	lbName := cloudprovider.GetLoadBalancerName(service)
	lb, err := l.lbByName(lbName)
	if err != nil {
		if err == errLBNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	if lb.IsMigrating() {
		err = l.sacloudAPI.WaitForLBActive(lb.ID)
		if err != nil {
			return nil, true, fmt.Errorf("error waiting for load balancer to be active %v", err)
		}
	}

	if len(lb.Settings.LoadBalancer) == 0 {
		// vip not found
		return nil, false, nil
	}

	ingress := []v1.LoadBalancerIngress{}
	ips := map[string]bool{}
	for _, setting := range lb.Settings.LoadBalancer {
		vip := setting.VirtualIPAddress
		if _, ok := ips[vip]; !ok {
			ingress = append(ingress, v1.LoadBalancerIngress{IP: vip})
		}
		ips[vip] = true
	}

	return &v1.LoadBalancerStatus{
		Ingress: ingress,
	}, true, nil
}

// EnsureLoadBalancer ensures that the cluster is running a load balancer for
// service.
//
// EnsureLoadBalancer will not modify service or nodes.
func (l *loadbalancers) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	_, exists, err := l.GetLoadBalancer(ctx, clusterName, service)
	if err != nil {
		return nil, err
	}

	if !exists {
		var vipParam *iaas.VIPParam
		vipParam, err = l.buildVIPParams(service, nodes)
		if err != nil {
			return nil, err
		}

		lbTags := []string{TagsKubernetesResource}
		if l.config.ClusterID != "" {
			lbTags = append(lbTags, fmt.Sprintf("%s=%s", TagsClusterID, l.config.ClusterID))
		}
		serviceTag := fmt.Sprintf("%s=%s",
			TagsLoadBalancerServiceName,
			strings.ShortenString(service.Name, 18),
		)
		lbTags = append(lbTags, serviceTag)

		lbParam := &iaas.LoadBalancerParam{
			Name:       cloudprovider.GetLoadBalancerName(service),
			Tags:       lbTags,
			RouterTags: []string{TagsKubernetesResource},
		}
		if v, ok := service.Annotations[annRouterSelector]; ok {
			if v != "" {
				lbParam.RouterTags = append(lbParam.RouterTags, v)
			}
		}
		if v, ok := service.Annotations[annLoadBalancerHA]; ok {
			if v != "" {
				lbParam.UseHA = true
			}
		}
		if v, ok := service.Annotations[annLoadBalancerPlan]; ok {
			if v == "premium" {
				lbParam.UseHighSpecPlan = true
			}
		}

		var vips []string
		vips, err = l.sacloudAPI.CreateLoadBalancer(lbParam, vipParam)
		if err != nil {
			return nil, err
		}

		ingress := []v1.LoadBalancerIngress{}
		for _, vip := range vips {
			ingress = append(ingress, v1.LoadBalancerIngress{IP: vip})
		}

		return &v1.LoadBalancerStatus{
			Ingress: ingress,
		}, nil
	}

	err = l.UpdateLoadBalancer(ctx, clusterName, service, nodes)
	if err != nil {
		return nil, err
	}

	var lbStatus *v1.LoadBalancerStatus
	lbStatus, _, err = l.GetLoadBalancer(ctx, clusterName, service)
	if err != nil {
		return nil, err
	}

	return lbStatus, nil
}

// UpdateLoadBalancer updates the load balancer for service to balance across in nodes.
//
// UpdateLoadBalancer will not modify service or nodes.
func (l *loadbalancers) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	lbName := cloudprovider.GetLoadBalancerName(service)
	lb, err := l.lbByName(lbName)
	if err != nil {
		return err
	}

	vipParam, err := l.buildVIPParams(service, nodes)
	if err != nil {
		return err
	}

	_, err = l.sacloudAPI.UpdateLoadBalancer(lb, vipParam)
	return err
}

// EnsureLoadBalancerDeleted deletes the specified loadbalancer if it exists.
// nil is returned if the load balancer for service does not exist or is
// successfully deleted.
//
// EnsureLoadBalancerDeleted will not modify service.
func (l *loadbalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	lbName := cloudprovider.GetLoadBalancerName(service)
	lb, err := l.lbByName(lbName)
	if err != nil {
		if err == errLBNotFound {
			return nil
		}
		return err
	}
	return l.sacloudAPI.DeleteLoadBalancer(lb.ID)
}

// lbByName gets a SAKURA Cloud Load Balancer by name. The returned error will
// be lbNotFound if the Load Balancer does not exist.
func (l *loadbalancers) lbByName(name string) (*sacloud.LoadBalancer, error) {
	lbs, err := l.sacloudAPI.LoadBalancers()
	if err != nil {
		return nil, err
	}

	for _, lb := range lbs {
		if !lb.IsFailed() && lb.Name == name {
			return &lb, nil
		}
	}

	return nil, errLBNotFound
}

func (l *loadbalancers) getAllNodeIPs(nodes []*v1.Node) ([]string, error) {

	ips := []string{}
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeExternalIP {
				ips = append(ips, addr.Address)
			}
		}
	}
	return ips, nil
}

func (l *loadbalancers) buildVIPParams(service *v1.Service, nodes []*v1.Node) (*iaas.VIPParam, error) {
	ports := []int32{}
	for _, port := range service.Spec.Ports {
		ports = append(ports, port.Port)
	}

	nodeIPs, err := l.getAllNodeIPs(nodes)
	if err != nil {
		return nil, err
	}

	healthCheck := &iaas.HealthCheck{
		Protocol:   "ping",
		Path:       "/",
		StatusCode: 200,
		DelayLoop:  10,
	}

	// collect health check spec from annotations
	if v, ok := service.Annotations[annHealthzInterval]; ok {
		if v != "" {
			status, err := strconv.Atoi(v)
			if err != nil {
				healthCheck.StatusCode = int32(status)
			}
		}
	}
	return &iaas.VIPParam{
		Ports:       ports,
		NodeIPs:     nodeIPs,
		HealthCheck: healthCheck,
	}, nil
}
