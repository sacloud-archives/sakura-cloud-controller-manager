package sakura

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
	"github.com/imdario/mergo"
	"github.com/kelseyhightower/envconfig"
	"io"
	"io/ioutil"
)

// Config represents CCM configuration includes sacloud API client configuration
type Config struct {
	AccessToken       string `json:"accessToken" yaml:"accessToken" split_words:"true"`
	AccessTokenSecret string `json:"accessTokenSecret" yaml:"accessTokenSecret" split_words:"true"`
	Zone              string `json:"zone" yaml:"zone" split_words:"true"`
	RetryMax          int    `json:"retryMax" yaml:"retryMax" split_words:"true"`
	RetryIntervalSec  int    `json:"retryIntervalSec" yaml:"retryIntervalSec" split_words:"true"`
	APIRootURL        string `json:"apiRootURL" yaml:"apiRootURL" split_words:"true"`
	TraceMode         bool   `json:"traceMode" yaml:"traceMode" split_words:"true"`

	ClusterID string `json:"clusterID" yaml:"clusterID" split_words:"true"`
}

// parseConfig returns a parsed configuration for an SAKURA Cloud cloudprovider config file
func parseConfig(configReader io.Reader) (*Config, error) {
	var envConfig Config
	var err error

	if err = envconfig.Process("sakuracloud", &envConfig); err != nil {
		return nil, err
	}
	if configReader == nil {
		return &envConfig, nil
	}

	var yamlConfig Config
	var configContents []byte
	configContents, err = ioutil.ReadAll(configReader)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(configContents, &yamlConfig)
	if err != nil {
		return nil, err
	}

	if err = mergo.Merge(&yamlConfig, envConfig); err != nil {
		return nil, err
	}

	return &yamlConfig, nil
}

// Validate returns error if CCM config is invalid
func (c *Config) Validate() error {

	var err error
	require := map[string]string{
		"accessToken":       c.AccessToken,
		"accessTokenSecret": c.AccessTokenSecret,
		"zone":              c.Zone,
	}
	for k, v := range require {
		if v == "" {
			err = multierror.Append(err, fmt.Errorf("%q is required", k))
		}
	}

	if c.ClusterID != "" {
		if len(c.ClusterID) > 18 {
			err = multierror.Append(err, fmt.Errorf("%q string length must be less equal 18", "clusterID"))
		}
	}

	return err
}
