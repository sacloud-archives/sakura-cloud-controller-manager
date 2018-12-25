package iaas

import (
	"fmt"
	"net"
	"time"

	"github.com/sacloud/libsacloud/api"
	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/sakura-cloud-controller-manager/version"
)

const apiFindLimit = 100

// ClientConfig represents config for IaaS API Client
type ClientConfig struct {
	AccessToken       string
	AccessTokenSecret string
	Zone              string
	AcceptLanguage    string
	RetryMax          int
	RetryIntervalSec  int64
	APIRootURL        string
	TraceMode         bool
}

// LoadBalancerParam represents LoadBalancer parameter for IaaS API
type LoadBalancerParam struct {
	ClusterSelector      []string
	Name                 string
	Description          string
	Tags                 []string
	RouterTags           []string
	UseHA                bool
	UseHighSpecPlan      bool
	VIP                  string
	IPAddressRange       string
	AssignIPAddressRange string
	DefaultGateway       string
	Type                 string
}

func (l *LoadBalancerParam) hasVIP() bool {
	return l.VIP == ""
}

func (l *LoadBalancerParam) assignAddresses() (net.IP, int, error) {
	ip, assignNet, err := net.ParseCIDR(l.AssignIPAddressRange)
	if err != nil {
		return nil, -1, err
	}
	ip = ip.To4()
	ip = ip.Mask(assignNet.Mask)
	maskLen, _ := assignNet.Mask.Size()
	return ip, maskLen, nil
}

func (l *LoadBalancerParam) nwMaskLen() (int, error) {
	_, lbNet, err := net.ParseCIDR(l.IPAddressRange)
	if err != nil {
		return -1, err
	}
	maskLen, _ := lbNet.Mask.Size()
	return maskLen, nil
}

// VIPParam represents LoadBalancer VIP parameter for IaaS API
type VIPParam struct {
	Ports   []*VIPPorts
	NodeIPs []string
	VIP     string
	Type    string
}

// VIPPorts represents LoadBalancer VIP port parameter for IaaS API
type VIPPorts struct {
	Port        int32
	HealthCheck *HealthCheck
}

// HealthCheck represents LoadBalancer health check rule
type HealthCheck struct {
	Protocol   string
	Path       string
	StatusCode int32
	DelayLoop  int32
	Port       int32
}

// Client IaaS API Client interface
type Client interface {
	AuthStatus() (*sacloud.AuthStatus, error)
	LoadBalancers(tags ...string) ([]sacloud.LoadBalancer, error)
	WaitForLBActive(id int64, waitTimeout time.Duration) error
	CreateLoadBalancer(*LoadBalancerParam, *VIPParam, time.Duration) ([]string, error)
	UpdateLoadBalancer(*sacloud.LoadBalancer, *LoadBalancerParam, *VIPParam) ([]string, error)
	DeleteLoadBalancer(id int64, waitTimeout time.Duration) error
	Servers() ([]sacloud.Server, error)
	ShutdownServerByID(id int64, shutdownWait time.Duration) error
	CurrentZone() string
}

type client struct {
	apiClient apiClient
}

// Config represents Iaas API Client configuration
type Config struct {
	AccessToken       string
	AccessTokenSecret string
	Zone              string
	AcceptLanguage    string
	RetryMax          int
	RetryIntervalSec  int
	APIRootURL        string
	TraceMode         bool
}

// NewClient returns Iaas API Client instance
func NewClient(c *Config) (Client, error) {

	rawClient := api.NewClient(c.AccessToken, c.AccessTokenSecret, c.Zone)
	rawClient.UserAgent = fmt.Sprintf("k8s-sakura-cloud-controller-manager/v%s", version.Version)

	rawClient.TraceMode = c.TraceMode
	if c.AcceptLanguage != "" {
		rawClient.AcceptLanguage = c.AcceptLanguage
	}
	if c.RetryMax > 0 {
		rawClient.RetryMax = c.RetryMax
	}
	if c.RetryIntervalSec > 0 {
		rawClient.RetryInterval = time.Duration(c.RetryIntervalSec) * time.Second
	}
	if c.APIRootURL != "" {
		api.SakuraCloudAPIRoot = c.APIRootURL
	}

	return &client{apiClient: newDefaultAPIClient(rawClient)}, nil
}

func (c *client) getAPIClient() apiClient {
	return c.apiClient.Clone()
}

// CurrentZone returns current zone name of IaaS API Client
func (c *client) CurrentZone() string {
	return c.apiClient.Zone()
}

type apiClient interface {
	Clone() apiClient
	Zone() string
	ReadAuthStatus() (*sacloud.AuthStatus, error)
	ReadSwitch(id int64) (*sacloud.Switch, error)
	FindServers() ([]sacloud.Server, error)
	FindLoadBalancers() ([]sacloud.LoadBalancer, error)
	FindLoadBalancersByTags(tags ...string) ([]sacloud.LoadBalancer, error)
	FindRoutersByTags(tags ...string) ([]sacloud.Internet, error)
	FindSwitchesByTags(tags ...string) ([]sacloud.Switch, error)
	FindVPCRouters() ([]sacloud.VPCRouter, error)
	FindDatabases() ([]sacloud.Database, error)
	ShutdownServer(id int64, shutdownWait time.Duration) error
	WaitForLBActive(id int64, wait time.Duration) error
	CreateLoadBalancer(value *sacloud.LoadBalancer) (*sacloud.LoadBalancer, error)
	ApplyLoadBalancerConfig(id int64) error
	UpdateLoadBalancer(id int64, value *sacloud.LoadBalancer) (*sacloud.LoadBalancer, error)
	DeleteLoadBalancer(id int64, waitTimeout time.Duration) error
}

type defaultAPIClient struct {
	rawClient *api.Client
}

func newDefaultAPIClient(rawClient *api.Client) *defaultAPIClient {
	return &defaultAPIClient{rawClient: rawClient}
}

func (d *defaultAPIClient) Clone() apiClient {
	return newDefaultAPIClient(d.rawClient.Clone())
}

func (d *defaultAPIClient) Zone() string {
	return d.rawClient.Zone
}

func (d *defaultAPIClient) ReadAuthStatus() (*sacloud.AuthStatus, error) {
	return d.rawClient.AuthStatus.Read()
}

func (d *defaultAPIClient) ReadSwitch(id int64) (*sacloud.Switch, error) {
	return d.rawClient.Switch.Read(id)
}

func (d *defaultAPIClient) FindServers() ([]sacloud.Server, error) {
	res, err := d.rawClient.Server.Reset().Limit(apiFindLimit).Find()
	if err != nil {
		return nil, err
	}
	return res.Servers, nil
}

func (d *defaultAPIClient) FindLoadBalancers() ([]sacloud.LoadBalancer, error) {
	res, err := d.rawClient.LoadBalancer.Reset().Limit(apiFindLimit).Find()
	if err != nil {
		return nil, err
	}
	return res.LoadBalancers, nil
}

func (d *defaultAPIClient) FindLoadBalancersByTags(tags ...string) ([]sacloud.LoadBalancer, error) {
	finder := d.rawClient.LoadBalancer.Reset().Limit(apiFindLimit)
	if len(tags) > 0 {
		finder.WithTags(tags)
	}
	res, err := finder.Find()
	if err != nil {
		return nil, err
	}
	return res.LoadBalancers, nil
}

func (d *defaultAPIClient) FindRoutersByTags(tags ...string) ([]sacloud.Internet, error) {
	finder := d.rawClient.Internet.Reset().Limit(apiFindLimit)
	if len(tags) > 0 {
		finder.WithTags(tags)
	}
	res, err := finder.Find()
	if err != nil {
		return nil, err
	}
	return res.Internet, nil
}

func (d *defaultAPIClient) FindSwitchesByTags(tags ...string) ([]sacloud.Switch, error) {
	finder := d.rawClient.Switch.Reset().Limit(apiFindLimit)
	if len(tags) > 0 {
		finder.WithTags(tags)
	}
	res, err := finder.Find()
	if err != nil {
		return nil, err
	}
	return res.Switches, nil
}

func (d *defaultAPIClient) FindVPCRouters() ([]sacloud.VPCRouter, error) {
	res, err := d.rawClient.VPCRouter.Reset().Limit(apiFindLimit).Find()
	if err != nil {
		return nil, err
	}
	return res.VPCRouters, nil
}

func (d *defaultAPIClient) FindDatabases() ([]sacloud.Database, error) {
	res, err := d.rawClient.Database.Reset().Limit(apiFindLimit).Find()
	if err != nil {
		return nil, err
	}
	return res.Databases, nil
}

func (d *defaultAPIClient) ShutdownServer(id int64, shutdownWait time.Duration) error {
	if _, err := d.rawClient.Server.Shutdown(id); err != nil {
		return err
	}
	if err := d.rawClient.Server.SleepUntilDown(id, shutdownWait); err == nil {
		return nil
	}
	if _, err := d.rawClient.Server.Stop(id); err != nil {
		return err
	}
	return d.rawClient.Server.SleepUntilDown(id, shutdownWait)
}

func (d *defaultAPIClient) WaitForLBActive(id int64, wait time.Duration) error {
	if err := d.rawClient.LoadBalancer.SleepWhileCopying(id, wait, 20); err != nil {
		return err
	}
	if err := d.rawClient.LoadBalancer.SleepUntilUp(id, wait); err != nil {
		return err
	}
	return nil
}

func (d *defaultAPIClient) CreateLoadBalancer(value *sacloud.LoadBalancer) (*sacloud.LoadBalancer, error) {
	return d.rawClient.LoadBalancer.Create(value)
}

func (d *defaultAPIClient) ApplyLoadBalancerConfig(id int64) error {
	_, err := d.rawClient.LoadBalancer.Config(id)
	return err
}

func (d *defaultAPIClient) UpdateLoadBalancer(id int64, value *sacloud.LoadBalancer) (*sacloud.LoadBalancer, error) {
	return d.rawClient.LoadBalancer.Update(id, value)
}

func (d *defaultAPIClient) DeleteLoadBalancer(id int64, waitTimeout time.Duration) error {
	_, err := d.rawClient.LoadBalancer.Stop(id)
	if err != nil {
		return err
	}
	err = d.rawClient.LoadBalancer.SleepUntilDown(id, waitTimeout)
	if err != nil {
		return err
	}

	_, err = d.rawClient.LoadBalancer.Delete(id)
	return err
}
