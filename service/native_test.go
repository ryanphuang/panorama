package service

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	dh "deephealth"
	"deephealth/client"
	dt "deephealth/types"
)

const (
	portstart = 10000
	portend   = 30000
)

var c *client.NClient
var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var (
	remote = flag.Bool("remote", false, "whether to perform remote service test or not")
	faddr  = flag.String("addr", "localhost:30000", "use this address instead of localhost")
)

func BenchmarkAddReport(b *testing.B) {
	o := dt.NewObservation(time.Now(), "cpu", "disk", "network")
	o.SetMetric("cpu", dt.UNHEALTHY, 30)
	o.SetMetric("disk", dt.HEALTHY, 90)
	o.SetMetric("network", dt.HEALTHY, 95)
	report := &dt.Report{
		Observer:    "XFE_2",
		Subject:     "TS_2",
		Observation: *o,
	}
	var reply int
	for i := 0; i < b.N; i++ {
		c.SubmitReport(report, &reply)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	var addr string
	if !*remote {
		port := portstart + int(r.Intn(portend-portstart))
		addr = fmt.Sprintf("localhost:%d", port)
		subjects := []dt.EntityId{"TS_1", "TS_2", "TS_3", "TS_4"}
		config := &dt.HealthServerConfig{
			Addr:     addr,
			Owner:    "XFE_1",
			Subjects: subjects,
		}
		dh.SetLogLevel(dh.ErrorLevel)
		hs := NewHealthNServer(config)
		hs.Start()
	} else {
		addr = *faddr
	}
	c = client.NewClient(addr, false)
	os.Exit(m.Run())
}
