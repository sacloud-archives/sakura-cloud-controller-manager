package iaas

import (
	"fmt"
	"time"

	"github.com/sacloud/libsacloud/api"
	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/sakura-cloud-controller-manager/version"
)

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
	Name            string
	Description     string
	Tags            []string
	RouterTags      []string
	UseHA           bool
	UseHighSpecPlan bool
}

// VIPParam represents LoadBalancer VIP parameter for IaaS API
type VIPParam struct {
	Ports       []int32
	HealthCheck *HealthCheck // TCP/HTTP/HTTPS
	NodeIPs     []string
}

// HealthCheck represents LoadBalancer health check rule
type HealthCheck struct {
	Protocol   string
	Path       string
	StatusCode int32
	DelayLoop  int32
}

// Client IaaS API Client interface
type Client interface {
	AuthStatus() (*sacloud.AuthStatus, error)
	LoadBalancers(tags ...string) ([]sacloud.LoadBalancer, error)
	WaitForLBActive(id int64) error
	CreateLoadBalancer(*LoadBalancerParam, *VIPParam) ([]string, error)
	UpdateLoadBalancer(*sacloud.LoadBalancer, *VIPParam) ([]string, error)
	DeleteLoadBalancer(id int64) error
	Servers() ([]sacloud.Server, error)
	CurrentZone() string
}

type client struct {
	rawClient *api.Client
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

	return &client{rawClient: rawClient}, nil
}

func (c *client) getRawClient() *api.Client {
	return c.rawClient.Clone()
}

// CurrentZone returns current zone name of IaaS API Client
func (c *client) CurrentZone() string {
	return c.rawClient.Zone
}
