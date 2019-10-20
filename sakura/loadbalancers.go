package sakura

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/sakura-cloud-controller-manager/iaas"
	"k8s.io/api/core/v1"
	"k8s.io/cloud-provider"
	"k8s.io/kubernetes/pkg/util/strings"
)

// TagsLoadBalancerServiceName is tag name indicating that resource is part of k8s cluster service
var TagsLoadBalancerServiceName = fmt.Sprintf("%s.Service", TagsKubernetesResource)

const (
	// annLoadBalancerExternalNetworkType is the annotation used to specify type
	// for setting LoadBalancer's network type.
	// Options are `internet` and `switch`.
	// default is `internet`.
	annLoadBalancerType = "k8s.usacloud.jp/load-balancer-type"

	// annRouterSelector is the annotation used to specify the condition
	// for finding upstream Router resource.
	// default is empty.
	annRouterSelector = "k8s.usacloud.jp/router-selector"

	// annRouterSelector is the annotation used to specify the condition
	// for finding upstream Router resource.
	// default is empty.
	annSwitchSelector = "k8s.usacloud.jp/switch-selector"

	// annLoadBalancerHA is the annotation used to specify the flag
	// for setting LoadBalancer high-availability mode.
	annLoadBalancerHA = "k8s.usacloud.jp/load-balancer-ha"

	// annLoadBalancerPlan is the annotation used to specify the plan
	// for setting LoadBalancer plan.
	// Options are `standard` and `premium`. Defaults to standard
	annLoadBalancerPlan = "k8s.usacloud.jp/load-balancer-plan"

	// annHealthzInterval is the annotation used to specify check interval(sec)
	// for health check to real nodes.
	// default is `10`.
	annHealthzInterval = "k8s.usacloud.jp/load-balancer-healthz-interval"

	// annLoadBalancerIPAddressRange is the annotation used to specify IP address range
	// for assigning to LoadBalancer's VIP and RealServer's IP address.
	// This annotation is used only when load-balancer-type is `switch`
	// default is `192.168.11.0/24`
	annLoadBalancerIPAddressRange = "k8s.usacloud.jp/load-balancer-ip-range"

	// annLoadBalancerSwitchIPAddressRange is the annotation used to specify IP address range
	// for assigning to LoadBalancer's VIP and RealServer's IP address.
	// This annotation is used only when load-balancer-type is `switch`
	// default is `192.168.11.0/24`
	annLoadBalancerAssignIPAddressRange = "k8s.usacloud.jp/load-balancer-assign-ip-range"

	// annLoadBalancerAssignDefaultGateway is the annotation used to specify DefaultGateway IP address.
	// This annotation is used only when load-balancer-type is `switch`
	// default is `192.168.11.1`
	annLoadBalancerAssignDefaultGateway = "k8s.usacloud.jp/load-balancer-assign-default-gateway"
)

var (
	errLBNotFound = errors.New("loadbalancer not found")
)

const (
	defaultLoadBalancerShutdownWait = time.Minute
	defaultLoadBalancerBootWait     = 10 * time.Minute
)

type loadbalancers struct {
	sacloudAPI   iaas.Client
	config       *Config
	shutdownWait time.Duration
	bootWait     time.Duration
}

// newLoadbalancers returns a cloudprovider.LoadBalancer whose concrete type is a *loadbalancer.
func newLoadbalancers(client iaas.Client, config *Config) cloudprovider.LoadBalancer {
	return &loadbalancers{
		sacloudAPI:   client,
		config:       config,
		shutdownWait: defaultLoadBalancerShutdownWait,
		bootWait:     defaultLoadBalancerBootWait,
	}
}

// GetLoadBalancer returns the *v1.LoadBalancerStatus of service.
//
// GetLoadBalancer will not modify service.
func (l *loadbalancers) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {

	lbName := l.GetLoadBalancerName(ctx, clusterName, service)
	lb, err := l.lbByName(lbName)
	if err != nil {
		if err == errLBNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	if lb.IsMigrating() {
		err = l.sacloudAPI.WaitForLBActive(lb.ID, l.bootWait)
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

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (l *loadbalancers) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	// TODO: replace DefaultLoadBalancerName to generate more meaningful loadbalancer names.
	return cloudprovider.DefaultLoadBalancerName(service)
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
		return l.createLoadBalancer(ctx, clusterName, service, nodes)
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

func (l *loadbalancers) createLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {

	loadBalancerType := l.getLoadBalancerType(service)
	switch loadBalancerType {
	case iaas.LoadBalancerTypesInternet, iaas.LoadBalancerTypesSwitch:
		return l.createLoadBalancerByType(ctx, clusterName, service, nodes, loadBalancerType)
	default:
		return nil, fmt.Errorf("%q is specified invalid value %q", annLoadBalancerType, loadBalancerType)
	}
}

func (l *loadbalancers) createLoadBalancerByType(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node, lbType string) (*v1.LoadBalancerStatus, error) {
	var vipParam *iaas.VIPParam
	vipParam, err := l.buildVIPParams(service, nodes, lbType)
	if err != nil {
		return nil, err
	}

	lbParam := l.createLoadBalancerParam(ctx, clusterName, service, lbType)

	vips, err := l.sacloudAPI.CreateLoadBalancer(lbParam, vipParam, l.bootWait)
	if err != nil {
		return nil, err
	}

	var ingress []v1.LoadBalancerIngress
	for _, vip := range vips {
		ingress = append(ingress, v1.LoadBalancerIngress{IP: vip})
	}

	return &v1.LoadBalancerStatus{
		Ingress: ingress,
	}, nil
}

// UpdateLoadBalancer updates the load balancer for service to balance across in nodes.
//
// UpdateLoadBalancer will not modify service or nodes.
func (l *loadbalancers) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	lbName := l.GetLoadBalancerName(ctx, clusterName, service)
	lb, err := l.lbByName(lbName)
	if err != nil {
		return err
	}

	// TODO use loadBalancerIP parameter
	lbType := l.getLoadBalancerType(service)

	lbParam := l.createLoadBalancerParam(ctx, clusterName, service, lbType)

	vipParam, err := l.buildVIPParams(service, nodes, lbType)
	if err != nil {
		return err
	}

	_, err = l.sacloudAPI.UpdateLoadBalancer(lb, lbParam, vipParam)
	return err
}

// EnsureLoadBalancerDeleted deletes the specified loadbalancer if it exists.
// nil is returned if the load balancer for service does not exist or is
// successfully deleted.
//
// EnsureLoadBalancerDeleted will not modify service.
func (l *loadbalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	lbName := l.GetLoadBalancerName(ctx, clusterName, service)
	lb, err := l.lbByName(lbName)
	if err != nil {
		if err == errLBNotFound {
			return nil
		}
		return err
	}
	return l.sacloudAPI.DeleteLoadBalancer(lb.ID, l.shutdownWait)
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

func (l *loadbalancers) getLoadBalancerType(service *v1.Service) string {
	loadBalancerType := iaas.LoadBalancerTypesInternet
	if v, ok := service.Annotations[annLoadBalancerType]; ok {
		loadBalancerType = v
	}
	return loadBalancerType
}

func (l *loadbalancers) getAllNodeIPs(nodes []*v1.Node, lbType string) ([]string, error) {
	ips := []string{}
	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeExternalIP && lbType == iaas.LoadBalancerTypesInternet {
				ips = append(ips, addr.Address)
			}
			if addr.Type == v1.NodeInternalIP && lbType == iaas.LoadBalancerTypesSwitch {
				ips = append(ips, addr.Address)
			}
		}
	}
	return ips, nil
}

func (l *loadbalancers) createLoadBalancerParam(ctx context.Context, clusterName string, service *v1.Service, lbType string) *iaas.LoadBalancerParam {
	var clusterSelector []string
	lbTags := []string{TagsKubernetesResource}
	if l.config.ClusterID != "" {
		clusterIDMarker := fmt.Sprintf("%s=%s", TagsClusterID, l.config.ClusterID)
		lbTags = append(lbTags, clusterIDMarker)
		clusterSelector = lbTags
	}
	serviceTag := fmt.Sprintf("%s=%s",
		TagsLoadBalancerServiceName,
		strings.ShortenString(service.Name, 18),
	)
	lbTags = append(lbTags, serviceTag)

	lbParam := &iaas.LoadBalancerParam{
		ClusterSelector: clusterSelector,
		Name:            l.GetLoadBalancerName(ctx, clusterName, service),
		Tags:            lbTags,
		RouterTags:      []string{TagsKubernetesResource},
		VIP:             service.Spec.LoadBalancerIP,
		Type:            lbType,
	}

	annSelector := annRouterSelector
	switch lbType {
	case iaas.LoadBalancerTypesInternet:
		annSelector = annRouterSelector
	case iaas.LoadBalancerTypesSwitch:
		annSelector = annSwitchSelector
		// Network settings for switch
		if v, ok := service.Annotations[annLoadBalancerIPAddressRange]; ok {
			if v != "" {
				lbParam.IPAddressRange = v
			}
		}
		if v, ok := service.Annotations[annLoadBalancerAssignIPAddressRange]; ok {
			if v != "" {
				lbParam.AssignIPAddressRange = v
			}
		}
		if v, ok := service.Annotations[annLoadBalancerAssignDefaultGateway]; ok {
			if v != "" {
				lbParam.DefaultGateway = v
			}
		}
	}

	if v, ok := service.Annotations[annSelector]; ok {
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

	return lbParam
}

func (l *loadbalancers) buildVIPParams(service *v1.Service, nodes []*v1.Node, lbType string) (*iaas.VIPParam, error) {
	var ports []*iaas.VIPPorts
	for _, port := range service.Spec.Ports {

		hc := &iaas.HealthCheck{
			Protocol:   "ping",
			DelayLoop:  10,
			Path:       "/", // we don't use this(for future)
			StatusCode: 200, // we don't use this(for future)
			Port:       port.Port,
		}

		// **
		// NOTE: currently, SakuraCloud's LB only support using same port between VIP/realServer.
		// **
		//if port.Protocol == "TCP" {
		//	hc.Protocol = "tcp"
		//} else {
		//	klog.Warningf("Protocol %q is not support for health-check, so we will use ping", port.Protocol)
		//}

		// collect health check spec from annotations
		if v, ok := service.Annotations[annHealthzInterval]; ok {
			if v != "" {
				status, err := strconv.Atoi(v)
				if err != nil {
					hc.StatusCode = int32(status)
				}
			}
		}

		ports = append(ports, &iaas.VIPPorts{
			Port:        port.Port,
			HealthCheck: hc,
		})
	}

	nodeIPs, err := l.getAllNodeIPs(nodes, lbType)
	if err != nil {
		return nil, err
	}

	return &iaas.VIPParam{
		Ports:   ports,
		NodeIPs: nodeIPs,
		VIP:     service.Spec.LoadBalancerIP,
		Type:    lbType,
	}, nil
}
