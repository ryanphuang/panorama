package service

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
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
var clients map[string]pb.HealthServiceClient

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

func BenchmarkSubmitReportAsync(b *testing.B) {
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	requestCh := make(chan *pb.Report)
	doneCh := make(chan bool)
	go func() {
		for {
			select {
			case <-requestCh:
				{
					// For async submission benchmarking, we use the channel to approximate a
					// worker queue. The client perceived latency is only the time to insert
					// a request to the queue, but not the processing time. Therefore, we
					// should just count the channel receiving time without doing the submission.
				}
			case <-doneCh:
				break
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		requestCh <- report
	}
	doneCh <- true
}

func BenchmarkSubmitReport(b *testing.B) {
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	for i := 0; i < b.N; i++ {
		client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Handle: handle, Report: report})
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
		client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Handle: handle, Report: report})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetInference(context.Background(), &pb.GetInferenceRequest{Subject: "TS_3"})
	}
}

func BenchmarkPropagate(b *testing.B) {
	myid, err := client.GetId(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Errorf("Failed to get my id\n")
		return
	}
	reply, err := client.GetPeers(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Errorf("Failed to get service peers\n")
		return
	}
	for _, peer := range reply.Peers {
		if peer.Id == myid.Id {
			continue
		}
		_, ok := clients[peer.Id]
		if ok {
			continue
		}
		conn, err := grpc.Dial(peer.Addr, grpc.WithInsecure())
		if err != nil {
			fmt.Errorf("Failed to connect to peer %s at %s\n", peer.Id, peer.Addr)
		}
		clients[peer.Id] = pb.NewHealthServiceClient(conn)
	}
	metrics := map[string]*pb.Value{
		"cpu":     &pb.Value{pb.Status_UNHEALTHY, 30},
		"disk":    &pb.Value{pb.Status_HEALTHY, 90},
		"network": &pb.Value{pb.Status_HEALTHY, 95},
	}
	report := dt.NewReport("XFE_2", "TS_3", metrics)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := &pb.LearnReportRequest{Source: myid, Report: report}
		var wg sync.WaitGroup
		for peer, client := range clients {
			wg.Add(1)
			go func() {
				_, err := client.LearnReport(context.Background(), request)
				if err != nil {
					fmt.Errorf("failed to propagate report about %s to %s\n", report.Subject, peer)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	var addr string

	if *create {
		if len(*faddr) == 0 {
			port := portstart + int(r.Intn(portend-portstart))
			addr = fmt.Sprintf("localhost:%d", port)
		} else {
			addr = *faddr
		}
		var config *dt.HealthServerConfig
		subjects := []string{"TS_1", "TS_2", "TS_3", "TS_4"}
		config = &dt.HealthServerConfig{
			Addr:     addr,
			Id:       "XFE_1",
			Subjects: subjects,
		}
		du.SetLogLevel(du.ErrorLevel)
		fmt.Printf("Creating DH service at %s\n", addr)
		gs := NewHealthGServer(config)
		errch := make(chan error)
		gs.Start(errch)
		time.Sleep(3)
	} else {
		if len(*faddr) == 0 {
			host, err := os.Hostname()
			if err != nil {
				fmt.Printf("Fail to get host name. Use localhost instead")
				host = "localhost"
			} else {
				host = strings.Split(host, ".")[0]
			}
			addr = host + ":6688"
		} else {
			addr = *faddr
		}
		fmt.Printf("Connecting to DH server at %s\n", addr)
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
	clients = make(map[string]pb.HealthServiceClient)
	handle = reply.Handle
	os.Exit(m.Run())
}
