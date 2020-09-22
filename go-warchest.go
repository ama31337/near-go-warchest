package main

import (
	"context"
	"flag"
	"log"

	"github.com/ama31337/near-go-warchest/common"
	"github.com/ama31337/near-go-warchest/near-shell/runner"
	"github.com/ama31337/near-go-warchest/rpc"
	nearapi "github.com/ama31337/near-go-warchest/rpc/client"
	prom "github.com/ama31337/near-go-warchest/services/prometheus"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "delegator ids"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var delegatorIds arrayFlags

func main() {
	log.Println("Go-Warchest started...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url := flag.String("url", "https://rpc.betanet.near.org", "Near JSON-RPC URL")
	addr := flag.String("addr", ":9444", "listen address")
	poolId := flag.String("accountId", "test", "Validator pool account id")
	flag.Var(&delegatorIds, "delegatorId", "Delegator ids.")

	flag.Parse()
	if len(flag.Args()) > 0 {
		flag.Usage()
	}

	client := nearapi.NewClientWithContext(ctx, *url)

	// Prometheus metrics
	promMetrics := prom.NewPromMetrics()
	// Run a metrics service
	go promMetrics.RunMetricsService(*addr)

	rpcMonitor := rpc.NewMonitor(client, *poolId)
	resCh := make(chan *rpc.SubscrResult)
	// Quota for a concurrent rpc requests
	sem := make(common.Sem, 1)
	// Run a remote rpc monitor
	go rpcMonitor.Run(ctx, resCh, sem, promMetrics)

	// Run a near-shell runner
	runner := runner.NewRunner(*poolId, delegatorIds)
	runner.Run(ctx, resCh, promMetrics, sem)
}
