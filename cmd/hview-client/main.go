package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "deephealth/build/gen"
	"deephealth/client"
	"deephealth/store"
	dt "deephealth/types"
)

const (
	cmdHelp = `Command list:
	 me observer
	 report subject [<metric:status:score...>]
	 get subject
	 help
	 exit
`
)

var (
	g = flag.Bool("grpc", true, "use grpc service client")
)

func logError(e error) {
	fmt.Fprintln(os.Stderr, e)
}

var observer dt.EntityId

type uclient struct {
	nc *client.NClient
	gc pb.HealthServiceClient
}

func parseReport(args []string) *dt.Report {
	var score float64
	var metric string
	var status dt.Status
	var err error

	subject := dt.EntityId(args[1])
	observation := dt.NewObservation(time.Now())
	for i := 2; i < len(args); i++ {
		parts := strings.Split(args[i], ":")
		if len(parts) == 3 {
			metric = parts[0]
			status = dt.StatusFromStr(parts[1])
			if status == dt.INVALID {
				logError(fmt.Errorf("invalid health metric %s\n", args[i]))
				break
			}
			score, err = strconv.ParseFloat(parts[2], 32)
			if err != nil {
				logError(fmt.Errorf("invalid health metric %s\n", args[i]))
				break
			}
			observation.AddMetric(metric, status, float32(score))
		} else {
			logError(fmt.Errorf("invalid health metric %s\n", args[i]))
			break
		}
	}
	if len(observer) == 0 {
		observer = dt.EntityId("XFE_0") // default observer
	}
	return &dt.Report{
		Observer:    observer,
		Subject:     subject,
		Observation: *observation,
	}
}

func runCmd(u *uclient, args []string) bool {
	var subject dt.EntityId
	var report dt.Report
	var err error
	var ret int

	cmd := args[0]
	switch cmd {
	case "report":
		r := parseReport(args)
		if u.nc != nil {
			err = u.nc.SubmitReport(r, &ret)
			switch ret {
			case store.REPORT_ACCEPTED:
				fmt.Println("Accepted")
			case store.REPORT_IGNORED:
				fmt.Println("Ignored")
			case store.REPORT_FAILED:
				fmt.Println("Failed")
			}
			if err != nil {
				logError(err)
			}
		} else {
			pbr := dt.ReportToPb(r)
			reply, err := u.gc.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: pbr})
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
			}
		}
	case "get":
		if u.nc != nil {
			subject = dt.EntityId(args[1])
			err = u.nc.GetReport(subject, &report)
			if err == nil {
				fmt.Println(report)
			} else {
				logError(err)
			}
		} else {
			reply, err := u.gc.GetReport(context.Background(), &pb.GetReportRequest{Subject: args[1]})
			if err == nil {
				rp := dt.ReportFromPb(reply.Report)
				fmt.Println(*rp)
			} else {
				logError(err)
			}
		}
	case "me":
		if len(args) == 1 {
			fmt.Println(observer)
		} else {
			observer = dt.EntityId(args[1])
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

func runPrompt(u *uclient) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")

	for scanner.Scan() {
		line := scanner.Text()
		args := fields(line)
		if len(args) > 0 {
			if runCmd(u, args) {
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

	var u uclient

	if *g {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(fmt.Sprintf("Could not connect to %s: %v", addr, err))
		}
		defer conn.Close()
		u.gc = pb.NewHealthServiceClient(conn)
	} else {
		u.nc = client.NewClient(addr, false)
	}
	if len(cmdArgs) == 0 {
		runPrompt(&u)
		fmt.Println()
	} else {
		runCmd(&u, cmdArgs)
	}
}
