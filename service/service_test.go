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
	dt "deephealth/types"
)

const (
	portstart = 10000
	portend   = 30000
)

var client pb.HealthServiceClient

var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var (
	remote = flag.Bool("remote", false, "whether to perform remote service test or not")
	faddr  = flag.String("addr", "localhost:30000", "use this address instead of localhost")
)

func TestSubmitReport(t *testing.T) {
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	_, err := client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: report})
	if err != nil {
		t.Errorf("Fail to submit report: %v", err)
	}
	fmt.Println("Submitted report")
}

func BenchmarkSubmitReport(b *testing.B) {
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	for i := 0; i < b.N; i++ {
		client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: report})
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	var addr string

	var config *dt.HealthServerConfig
	if !*remote {
		port := portstart + int(r.Intn(portend-portstart))
		addr = fmt.Sprintf("localhost:%d", port)
		subjects := []string{"TS_1", "TS_2", "TS_3", "TS_4"}
		config = &dt.HealthServerConfig{
			Addr:     addr,
			Id:       "XFE_1",
			Subjects: subjects,
		}
		dh.SetLogLevel(dh.ErrorLevel)
	} else {
		addr = *faddr
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Could not connect to %s: %v", addr, err))
	}
	defer conn.Close()
	client = pb.NewHealthServiceClient(conn)
	if !*remote {
		gs := NewHealthGServer(config)
		errch := make(chan error)
		gs.Start(errch)
	}
	time.Sleep(3)

	os.Exit(m.Run())
}
