package service

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	dh "deephealth"
	pb "deephealth/build/gen"
	"deephealth/client"
	dt "deephealth/types"
)

const (
	portstart = 10000
	portend   = 30000
)

type uclient struct {
	nc *client.NClient
	gc pb.HealthServiceClient
}

var u uclient

var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var (
	g      = flag.Bool("grpc", true, "use grpc service client")
	remote = flag.Bool("remote", false, "whether to perform remote service test or not")
	faddr  = flag.String("addr", "localhost:30000", "use this address instead of localhost")
)

func TestSubmitReport(t *testing.T) {
	metrics := map[string]dt.Value{
		"cpu":     dt.Value{dt.UNHEALTHY, 30},
		"disk":    dt.Value{dt.HEALTHY, 90},
		"network": dt.Value{dt.HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	if u.nc != nil {
		var reply int
		err := u.nc.SubmitReport(report, &reply)
		if err != nil {
			t.Errorf("Fail to submit report: %v", err)
		}
	} else {
		pbr := dt.ReportToPb(report)
		_, err := u.gc.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: pbr})
		if err != nil {
			t.Errorf("Fail to submit report: %v", err)
		}
		fmt.Println("Submitted report")
	}
}

func BenchmarkSubmitReport(b *testing.B) {
	metrics := map[string]dt.Value{
		"cpu":     dt.Value{dt.UNHEALTHY, 30},
		"disk":    dt.Value{dt.HEALTHY, 90},
		"network": dt.Value{dt.HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	var reply int
	if u.nc != nil {
		for i := 0; i < b.N; i++ {
			u.nc.SubmitReport(report, &reply)
		}
	} else {
		pbr := dt.ReportToPb(report)
		for i := 0; i < b.N; i++ {
			u.gc.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: pbr})
		}
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	var addr string

	var config *dt.HealthServerConfig
	if !*remote {
		port := portstart + int(r.Intn(portend-portstart))
		addr = fmt.Sprintf("localhost:%d", port)
		subjects := []dt.EntityId{"TS_1", "TS_2", "TS_3", "TS_4"}
		config = &dt.HealthServerConfig{
			Addr:     addr,
			Owner:    "XFE_1",
			Subjects: subjects,
		}
		dh.SetLogLevel(dh.ErrorLevel)
	} else {
		addr = *faddr
	}

	if *g {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(fmt.Sprintf("Could not connect to %s: %v", addr, err))
		}
		defer conn.Close()
		u.gc = pb.NewHealthServiceClient(conn)
		if !*remote {
			gs := NewHealthGServer(config)
			errch := make(chan error)
			gs.Start(errch)
		}
	} else {
		u.nc = client.NewClient(addr, false)
		if !*remote {
			hs := NewHealthNServer(config)
			hs.Start()
		}
	}
	// time.Sleep(1)

	os.Exit(m.Run())
}
