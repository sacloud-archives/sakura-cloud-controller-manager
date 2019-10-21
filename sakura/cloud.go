package sakura

import (
	"fmt"
	"io"

	"github.com/sacloud/sakura-cloud-controller-manager/iaas"
	"k8s.io/cloud-provider"
)

const (
	// ProviderName is name of CCM provider
	ProviderName string = "sakuracloud"

	// ControllerName is name of CCM for logging
	ControllerName string = "sakura-cloud-controller-manager"
)

const (
	// TagsKubernetesResource is a marker tag indicating that resource is part of k8s cluster
	TagsKubernetesResource = "@k8s"

	// TagsClusterID is tag name for mark ClusterID
	TagsClusterID = TagsKubernetesResource + ".ClusterID"
)

type cloud struct {
	sacloudAPI iaas.Client
	config     *Config

	instances     cloudprovider.Instances
	loadBalancers cloudprovider.LoadBalancer
	zones         cloudprovider.Zones
}

func newCloud(configReader io.Reader) (cloudprovider.Interface, error) {

	config, err := parseConfig(configReader)
	if err != nil {
		return nil, fmt.Errorf("initializing cloud provider %q is failed: %s", ProviderName, err)
	}
	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("initializing cloud provider %q is failed: %s", ProviderName, err)
	}

	var client iaas.Client
	client, err = iaas.NewClient(&iaas.Config{
		AccessToken:       config.AccessToken,
		AccessTokenSecret: config.AccessTokenSecret,
		Zone:              config.Zone,
		AcceptLanguage:    "en", // must be "en", see https://github.com/sacloud/sakura-cloud-controller-manager/issues/4
		RetryMax:          config.RetryMax,
		RetryIntervalSec:  config.RetryIntervalSec,
		APIRootURL:        config.APIRootURL,
		TraceMode:         config.TraceMode,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing cloud provider %q is failed: %s", ProviderName, err)
	}

	return &cloud{
		sacloudAPI:    client,
		config:        config,
		instances:     newInstances(client),
		loadBalancers: newLoadbalancers(client, config),
		zones:         newZones(client),
	}, nil
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	})
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping activities within the cloud provider.
func (c *cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	// TODO changed k8s 1.13
}

// LoadBalancer returns a balancer interface. Also returns true if the interface is supported, false otherwise.
func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	if c.config.DisableLoadBalancer {
		return nil, false
	}
	return c.loadBalancers, true
}

// Instances returns an instances interface. Also returns true if the interface is supported, false otherwise.
func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	return c.instances, true
}

// Zones returns a zones interface. Also returns true if the interface is supported, false otherwise.
func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	return c.zones, true
}

// Clusters returns a clusters interface.  Also returns true if the interface is supported, false otherwise.
func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false // not supported
}

// Routes returns a routes interface along with whether the interface is supported.
func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false // not supported
}

// ProviderName returns the cloud provider ID.
func (c *cloud) ProviderName() string {
	return ProviderName
}

// ScrubDNS provides an opportunity for cloud-provider-specific code to process DNS settings for pods.
func (c *cloud) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nil, nil // not supported
}

// HasClusterID returns true if a ClusterID is required and set
func (c *cloud) HasClusterID() bool {
	return true
}
