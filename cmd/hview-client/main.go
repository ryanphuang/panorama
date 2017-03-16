package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	dh "deephealth"
	"deephealth/client"
)

const (
	help = "Usage: hview-client <server address> [command <args...>]"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, help)
		os.Exit(1)
	}
	addr := args[0]
	c := client.NewSimpleClient(addr)
	o := dh.NewObservation(time.Now(), "cpu", "disk", "network")
	o.SetMetric("cpu", dh.UNHEALTHY, 30)
	o.SetMetric("disk", dh.HEALTHY, 90)
	o.SetMetric("network", dh.HEALTHY, 95)
	report := &dh.Report{
		Observer:    "HS_2",
		Subject:     "XFE_3",
		Observation: *o,
	}
	var reply int
	fmt.Printf("Calling add report to %s\n", addr)
	c.Call("HealthStorage.AddReport", report, &reply)
}
