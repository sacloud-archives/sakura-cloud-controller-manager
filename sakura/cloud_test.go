package sakura

import (
	"time"

	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/sakura-cloud-controller-manager/iaas"
)

type testSacloudClient struct {
	authStatus *sacloud.AuthStatus
	authError  error

	loadBalancers      []sacloud.LoadBalancer
	loadBalancersError error

	waitForLBActive error

	createdVIPs             []string
	createLoadBalancerError error

	updatedVIPs             []string
	updateLoadBalancerError error

	deleteLoadBalancerError error

	servers      []sacloud.Server
	serversError error

	shutdownServerError error

	currentZone string
}

func (t *testSacloudClient) AuthStatus() (*sacloud.AuthStatus, error) {
	return t.authStatus, t.authError
}

func (t *testSacloudClient) LoadBalancers(tags ...string) ([]sacloud.LoadBalancer, error) {
	return t.loadBalancers, t.loadBalancersError
}

func (t *testSacloudClient) WaitForLBActive(id int64, wait time.Duration) error {
	return t.waitForLBActive
}
func (t *testSacloudClient) CreateLoadBalancer(*iaas.LoadBalancerParam, *iaas.VIPParam, time.Duration) ([]string, error) {
	return t.createdVIPs, t.createLoadBalancerError
}
func (t *testSacloudClient) UpdateLoadBalancer(*sacloud.LoadBalancer, *iaas.LoadBalancerParam, *iaas.VIPParam) ([]string, error) {
	return t.updatedVIPs, t.updateLoadBalancerError
}
func (t *testSacloudClient) DeleteLoadBalancer(id int64, wait time.Duration) error {
	return t.deleteLoadBalancerError
}
func (t *testSacloudClient) Servers() ([]sacloud.Server, error) {
	return t.servers, t.serversError
}
func (t *testSacloudClient) ShutdownServerByID(id int64, shutdownWait time.Duration) error {
	return t.shutdownServerError
}
func (t *testSacloudClient) CurrentZone() string {
	return t.currentZone
}
