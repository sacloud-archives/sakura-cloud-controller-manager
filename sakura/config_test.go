package sakura

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var dummyYamlBody = `---
accessToken: "token"
accessTokenSecret: "secret"
zone: "zone"
clusterID: "id"
`

func TestParseConfig(t *testing.T) {

	testCases := []struct {
		caseName     string
		yaml         string
		environments map[string]string
		expect       *Config
	}{
		{
			caseName: "env only",
			environments: map[string]string{
				"SAKURACLOUD_ACCESS_TOKEN":        "token",
				"SAKURACLOUD_ACCESS_TOKEN_SECRET": "secret",
				"SAKURACLOUD_ZONE":                "zone",
				"SAKURACLOUD_CLUSTER_ID":          "id",
			},
			expect: &Config{
				AccessToken:       "token",
				AccessTokenSecret: "secret",
				Zone:              "zone",
				ClusterID:         "id",
			},
		},
		{
			caseName: "yaml only",
			yaml:     dummyYamlBody,
			expect: &Config{
				AccessToken:       "token",
				AccessTokenSecret: "secret",
				Zone:              "zone",
				ClusterID:         "id",
			},
		}, {
			caseName: "env and yaml",
			yaml:     dummyYamlBody,
			environments: map[string]string{
				"SAKURACLOUD_ACCESS_TOKEN":        "dummy",
				"SAKURACLOUD_ACCESS_TOKEN_SECRET": "dummy",
				"SAKURACLOUD_ZONE":                "dummy",
				"SAKURACLOUD_CLUSTER_ID":          "dummy",
			},
			expect: &Config{
				AccessToken:       "token",
				AccessTokenSecret: "secret",
				Zone:              "zone",
				ClusterID:         "id",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {

			var reader io.Reader
			if testCase.yaml != "" {
				reader = strings.NewReader(testCase.yaml)
			}
			for k, v := range testCase.environments {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg, err := parseConfig(reader)
			assert.NoError(t, err)
			assert.EqualValues(t, testCase.expect, cfg)
		})
	}

}
