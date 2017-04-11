package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "deephealth/build/gen"
	dt "deephealth/types"
)

const (
	cmdHelp = `Command list:
	 me observer
	 report subject [<metric:status:score...>]
	 get [report|view|panorama] [observer] subject 
	 ping
	 help
	 exit
`
)

func logError(e error) {
	fmt.Fprintln(os.Stderr, e)
}

var observer string

var client pb.HealthServiceClient

func parseReport(args []string) *pb.Report {
	var score float64
	var metric string
	var status pb.Status
	var err error

	subject := args[1]
	observation := dt.NewObservation(time.Now())
	for i := 2; i < len(args); i++ {
		parts := strings.Split(args[i], ":")
		if len(parts) == 3 {
			metric = parts[0]
			status = dt.StatusFromStr(parts[1])
			if status == pb.Status_INVALID {
				logError(fmt.Errorf("invalid health metric %s\n", args[i]))
				break
			}
			score, err = strconv.ParseFloat(parts[2], 32)
			if err != nil {
				logError(fmt.Errorf("invalid health metric %s\n", args[i]))
				break
			}
			dt.AddMetric(observation, metric, status, float32(score))
		} else {
			logError(fmt.Errorf("invalid health metric %s\n", args[i]))
			break
		}
	}
	if len(observer) == 0 {
		observer = "XFE_0" // default observer
	}
	return &pb.Report{
		Observer:    observer,
		Subject:     subject,
		Observation: observation,
	}
}

func runCmd(args []string) bool {
	cmd := args[0]
	switch cmd {
	case "ping":
		now := time.Now()
		pnow, err := ptypes.TimestampProto(now)
		if err == nil {
			request := &pb.PingRequest{Source: &pb.Peer{string(observer), "localhost"}, Time: pnow}
			reply, err := client.Ping(context.Background(), request)
			if err == nil {
				t, err := ptypes.Timestamp(reply.Time)
				if err == nil {
					fmt.Println("ping reply at time %s", t)
				}
			}
		}
		if err != nil {
			logError(err)
			return false
		}
	case "report":
		r := parseReport(args)
		reply, err := client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: r})
		switch reply.Result {
		case pb.SubmitReportReply_ACCEPTED:
			fmt.Println("Accepted")
		case pb.SubmitReportReply_IGNORED:
			fmt.Println("Ignored")
		case pb.SubmitReportReply_FAILED:
			fmt.Println("Failed")
		}
		if err != nil {
			logError(err)
			return false
		}
	case "get":
		if len(args) == 1 {
			fmt.Println(cmdHelp)
			return false
		}
		switch args[1] {
		case "report":
			if len(args) != 3 {
				fmt.Println(cmdHelp)
				return false
			}
			report, err := client.GetLatestReport(context.Background(), &pb.GetReportRequest{Subject: args[2]})
			if err == nil {
				fmt.Println(report)
				return false
			} else {
				logError(err)
			}
		case "view":
			if len(args) != 4 {
				fmt.Println(cmdHelp)
				return false
			}
			view, err := client.GetView(context.Background(), &pb.GetViewRequest{Observer: args[2], Subject: args[3]})
			if err == nil {
				fmt.Println(view)
				return false
			} else {
				logError(err)
			}
		case "panorama":
			if len(args) != 3 {
				fmt.Println(cmdHelp)
				return false
			}
			pano, err := client.GetPanorama(context.Background(), &pb.GetPanoramaRequest{Subject: args[2]})
			if err == nil {
				fmt.Println(pano)
				return false
			} else {
				logError(err)
			}
		default:
			fmt.Println(cmdHelp)
			return false
		}
	case "me":
		if len(args) == 1 {
			fmt.Println(observer)
		} else {
			observer = args[1]
		}
	case "help":
		fmt.Println(cmdHelp)
	case "exit":
		return true
	default:
		logError(fmt.Errorf("bad command, try \"help\"."))
	}
	return false
}

func runPrompt() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")

	for scanner.Scan() {
		line := scanner.Text()
		args := fields(line)
		if len(args) > 0 {
			if runCmd(args) {
				break
			}
		}
		fmt.Print("> ")
	}

	e := scanner.Err()
	if e != nil {
		panic(e)
	}
}

func fields(s string) []string {
	return strings.Fields(s)
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <server address> [command <args...>]\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println("\nIf no command was specified, the client enters an interactive mode.\n")
		fmt.Println(cmdHelp)
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		flag.Usage()
		os.Exit(1)
	}

	addr := args[0]
	cmdArgs := args[1:]

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Could not connect to %s: %v", addr, err))
	}
	defer conn.Close()
	client = pb.NewHealthServiceClient(conn)
	if len(cmdArgs) == 0 {
		runPrompt()
		fmt.Println()
	} else {
		runCmd(cmdArgs)
	}
}
