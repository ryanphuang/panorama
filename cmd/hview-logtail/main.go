package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hpcloud/tail"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "deephealth/build/gen"
	dp "deephealth/plugin"
	dt "deephealth/types"
)

var (
	report = flag.Bool("report", true, "Whether to report events to health service")
	log    = flag.String("log", "", "Log file to watch for (Required)")
	server = flag.String("server", "localhost:6688", "Address of health server to report events to (Required)")
)

const (
	staleSeconds = 5 * 60 // 5 minutes
	mergeSeconds = 5      // merge within 5 seconds
)

var lastReportTime = make(map[string]time.Time)
var ipEntities = make(map[string]string)

func usage() {
	fmt.Printf("Usage: %s OPTIONS <plugin> [PLUGIN OPTIONS]...\n\n", os.Args[0])
	flag.PrintDefaults()
}

func reportEvent(client pb.HealthServiceClient, event *dt.Event) error {
	ts, ok := lastReportTime[event.Subject]
	if ok && time.Now().Sub(ts).Seconds() < mergeSeconds {
		fmt.Printf("report for %s is too frequent, skip\n", event.Subject)
		return nil
	}
	observation := dt.NewObservationSingleMetric(event.Time, event.Context, pb.Status_UNHEALTHY, 20)
	report := &pb.Report{
		Observer:    event.Id,
		Subject:     event.Subject,
		Observation: observation,
	}
	reply, err := client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: report})
	lastReportTime[event.Subject] = time.Now()
	if err != nil {
		return err
	}
	switch reply.Result {
	case pb.SubmitReportReply_ACCEPTED:
		fmt.Printf("Accepted report %s\n", event)
	case pb.SubmitReportReply_IGNORED:
		fmt.Printf("Ignored report %s\n", event)
	case pb.SubmitReportReply_FAILED:
		fmt.Printf("Failed report %s\n", event)
	}
	return nil
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if len(*log) == 0 {
		fmt.Println("Log file argument is empty")
		os.Exit(1)
	}
	var plugin dp.LogTailPlugin
	switch args[0] {
	case "zookeeper":
		plugin = new(dp.ZooKeeperPlugin)
	default:
		fmt.Println("Unsupported plugin " + args[0])
		os.Exit(1)
	}
	plugin.ProvideFlags().Parse(args[1:])
	err := plugin.ValidateFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	plugin.Init()
	parser := plugin.ProvideEventParser()

	var client pb.HealthServiceClient
	if *report {
		conn, err := grpc.Dial(*server, grpc.WithInsecure())
		if err != nil {
			panic(fmt.Sprintf("Could not connect to %s: %v", *server, err))
		}
		defer conn.Close()
		client = pb.NewHealthServiceClient(conn)
	}

	fmt.Println("Start monitoring " + *log)

	t, _ := tail.TailFile(*log, tail.Config{Follow: true})
	for line := range t.Lines {
		event := parser.ParseLine(line.Text)
		if event != nil {
			/*
				if time.Since(event.ts).Seconds() > staleSeconds {
					fmt.Printf("skip stale event: %s\n", event)
					continue
				}
			*/
			fmt.Println(event)
			if *report {
				reportEvent(client, event)
			}
		}
	}
}
