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

	pb "deephealth/build/gen"
	dt "deephealth/types"
	du "deephealth/util"
)

const (
	portstart = 10000
	portend   = 30000
)

var client pb.HealthServiceClient
var handle uint64

var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var (
	faddr  = flag.String("addr", "", "use this address instead of localhost")
	create = flag.Bool("create", true, "whether to create service ourselves or not")
)

func TestSubmitReport(t *testing.T) {
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	_, err := client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Handle: handle, Report: report})
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

func BenchmarkGetInference(b *testing.B) {
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	for i := 0; i < b.N; i++ {
		client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: report})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetInference(context.Background(), &pb.GetInferenceRequest{Subject: "TS_3"})
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	var addr string

	var config *dt.HealthServerConfig
	if len(*faddr) == 0 {
		port := portstart + int(r.Intn(portend-portstart))
		addr = fmt.Sprintf("localhost:%d", port)
	} else {
		addr = *faddr
	}

	if *create {
		subjects := []string{"TS_1", "TS_2", "TS_3", "TS_4"}
		config = &dt.HealthServerConfig{
			Addr:     addr,
			Id:       "XFE_1",
			Subjects: subjects,
		}
		du.SetLogLevel(du.ErrorLevel)
		gs := NewHealthGServer(config)
		errch := make(chan error)
		gs.Start(errch)
		time.Sleep(3)
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Could not connect to %s: %v", addr, err))
	}
	defer conn.Close()
	client = pb.NewHealthServiceClient(conn)
	reply, err := client.Register(context.Background(), &pb.RegisterRequest{Module: "DeepHealth", Observer: "self"})
	if err != nil {
		panic(fmt.Sprintf("Fail to register with the health service: %v", err))
	}
	handle = reply.Handle
	os.Exit(m.Run())
}
