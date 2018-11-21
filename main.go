package main

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	_ "github.com/sacloud/sakura-cloud-controller-manager/sakura"
	"github.com/sacloud/sakura-cloud-controller-manager/version"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app/options"
	_ "k8s.io/kubernetes/pkg/client/metrics/prometheus" // for client metric registration
	_ "k8s.io/kubernetes/pkg/version/prometheus"        // for version metric registration
	"k8s.io/kubernetes/pkg/version/verflag"
)

func init() {
	healthz.DefaultHealthz()
}

func main() {
	s, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		glog.Fatalf("failed to create config options: %s", err)
	}

	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()
	glog.V(1).Infof("sakura-cloud-controller-manager version: %s", version.FullVersion())

	verflag.PrintAndExitIfRequested()

	config, err := s.Config()
	if err != nil {
		glog.Fatalf("failed to create component config: %s", err)
	}

	if err := app.Run(config.Complete()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err) // nolint
		os.Exit(1)
	}
}
