package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
  "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"

	cluster "github.com/linsite/cluster-metrics-server/internal/pkg"
)


type clusterAdapter struct {
    basecmd.AdapterBase

    // the message printed on startup
    Message string
}


func (a *clusterAdapter) makeProviderOrDie() cluster.CustomMetricsProvider{
    client, err := a.DynamicClient()
    if err != nil {
        klog.Fatalf("unable to construct dynamic client: %v", err)
    }

    mapper, err := a.RESTMapper()
    if err != nil {
        klog.Fatalf("unable to construct discovery REST mapper: %v", err)
    }

		cfg, err := a.ClientConfig()
    if err != nil {
        klog.Fatalf("unable to get cfg: %v", err)
    }
		clientSet := kubernetes.NewForConfigOrDie(cfg)
    return cluster.NewProvider(client, mapper, clientSet)
}

func main() {
    logs.InitLogs()
    defer logs.FlushLogs()

    cmd := &clusterAdapter{}
    cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
    logs.AddGoFlags(flag.CommandLine)
    cmd.Flags().AddGoFlagSet(flag.CommandLine)
    cmd.Flags().Parse(os.Args)

    provider := cmd.makeProviderOrDie()
    cmd.WithCustomMetrics(provider)
    cmd.WithExternalMetrics(provider)

    klog.Infof(cmd.Message)
    if err := cmd.Run(wait.NeverStop); err != nil {
        klog.Fatalf("unable to run custom metrics adapter: %v", err)
    }
}
