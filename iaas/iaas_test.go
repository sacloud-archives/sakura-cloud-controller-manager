package iaas

import (
	"errors"
	"log"
	"os"
	"strconv"
	"testing"
)

var testClient *client

func TestMain(m *testing.M) {

	accessToken := os.Getenv("SAKURACLOUD_ACCESS_TOKEN")
	accessTokenSecret := os.Getenv("SAKURACLOUD_ACCESS_TOKEN_SECRET")

	if accessToken == "" || accessTokenSecret == "" {
		log.Println("Please Set ENV 'SAKURACLOUD_ACCESS_TOKEN' and 'SAKURACLOUD_ACCESS_TOKEN_SECRET'")
		os.Exit(0) // exit normal
	}

	zone := os.Getenv("SAKURACLOUD_ZONE")
	if zone == "" {
		zone = "is1b"
	}

	acceptLanguage := os.Getenv("SAKURACLOUD_ACCEPT_LANGUAGE")

	retryMax := 0
	strRetryMax := os.Getenv("SAKURACLOUD_RETRY_MAX")
	if strRetryMax != "" {
		retryMax, _ = strconv.Atoi(strRetryMax)
	}

	retryInterval := 0
	strInterval := os.Getenv("SAKURACLOUD_RETRY_INTERVAL")
	if strInterval != "" {
		retryInterval, _ = strconv.Atoi(strInterval)
	}

	apiRootURL := os.Getenv("SAKURACLOUD_API_ROOT_URL")

	traceMode := false
	if os.Getenv("SAKURACLOUD_TRACE_MODE") != "" {
		traceMode = true
	}

	c, err := NewClient(&Config{
		AccessToken:       accessToken,
		AccessTokenSecret: accessTokenSecret,
		Zone:              zone,
		AcceptLanguage:    acceptLanguage,
		RetryMax:          retryMax,
		RetryIntervalSec:  retryInterval,
		APIRootURL:        apiRootURL,
		TraceMode:         traceMode,
	})
	if err != nil {
		panic(err)
	}
	tc, ok := c.(*client)
	if !ok {
		panic(errors.New("Invalid client"))
	}
	testClient = tc

	ret := m.Run()
	os.Exit(ret)
}
