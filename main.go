package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/sacloud/sakura-cloud-controller-manager/sakura"
	"github.com/sacloud/sakura-cloud-controller-manager/version"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app"
	_ "k8s.io/kubernetes/pkg/client/metrics/prometheus" // for client metric registration
	_ "k8s.io/kubernetes/pkg/version/prometheus"        // for version metric registration
)

func init() {
	healthz.DefaultHealthz()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewCloudControllerManagerCommand()

	logs.InitLogs()
	defer logs.FlushLogs()

	klog.V(1).Infof("sakura-cloud-controller-manager version: %s", version.FullVersion())

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
