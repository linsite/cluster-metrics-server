package main

import (
	    "flag"
	    "os"

		  "k8s.io/apimachinery/pkg/util/wait"
		  "k8s.io/component-base/logs"
			"k8s.io/klog/v2"

			basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
			"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

				        // make this the path to the provider that you just wrote
		yourprov "github.com/linsite/k8s-cluster-metrics/internal/pkg"
)


type YourAdapter struct {
    basecmd.AdapterBase

    // the message printed on startup
    Message string
}


func (a *YourAdapter) makeProviderOrDie() provider.CustomMetricsProvider {
    client, err := a.DynamicClient()
    if err != nil {
        klog.Fatalf("unable to construct dynamic client: %v", err)
    }

    mapper, err := a.RESTMapper()
    if err != nil {
        klog.Fatalf("unable to construct discovery REST mapper: %v", err)
    }

    return yourprov.NewProvider(client, mapper)
}

func main() {
    logs.InitLogs()
    defer logs.FlushLogs()

    // initialize the flags, with one custom flag for the message
    cmd := &YourAdapter{}
    cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
    // make sure you get the klog flags
    logs.AddGoFlags(flag.CommandLine)
    cmd.Flags().AddGoFlagSet(flag.CommandLine)
    cmd.Flags().Parse(os.Args)

    provider := cmd.makeProviderOrDie()
    cmd.WithCustomMetrics(provider)
    // you could also set up external metrics support,
    // if your provider supported it:
    // cmd.WithExternalMetrics(provider)

    klog.Infof(cmd.Message)
    if err := cmd.Run(wait.NeverStop); err != nil {
        klog.Fatalf("unable to run custom metrics adapter: %v", err)
    }
}
