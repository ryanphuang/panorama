package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "panorama/build/gen"
	dt "panorama/types"
)

var (
	server = flag.String("server", "", "Address of health server to report events to (Required)")
)

const (
	cmdHelp = `Command list:
	 me observer
	 report subject [<metric:status:score...>]
	 list [subject]
	 get [report|view|inference|panorama] [observer] subject 
	 dump [inference|panorama]
	 tail freq [get|dump]...
	 ping
	 help
	 exit
`
)

func logError(e error) {
	fmt.Fprintln(os.Stderr, e)
}

type subjectSorter struct {
	subjects []string
	cmp      func(str1, str2 string) bool
}

type tok struct {
	s string
	n int
}

var (
	dx = regexp.MustCompile(`\d+|\D+`)
)

func tokString(str string) tok {
	var t tok
	x := dx.FindAllString(str, 2)
	for _, s := range x {
		if n, err := strconv.Atoi(s); err == nil {
			t.n = n
		} else {
			t.s = s
		}
	}
	return t
}

func cmpSubject(str1, str2 string) bool {
	t1 := tokString(str1)
	t2 := tokString(str2)
	if t1.s == "" && (t2.s > "" || t1.n < t2.n) {
		return true
	}
	if t1.s < t2.s {
		return true
	}
	return t1.n < t2.n
}

func sortSubjects(subjects []string) {
	strSort := &subjectSorter{
		subjects: subjects,
		cmp:      cmpSubject,
	}
	sort.Sort(strSort)
}

func (s *subjectSorter) Len() int           { return len(s.subjects) }
func (s *subjectSorter) Swap(i, j int)      { s.subjects[i], s.subjects[j] = s.subjects[j], s.subjects[i] }
func (s *subjectSorter) Less(i, j int) bool { return s.cmp(s.subjects[i], s.subjects[j]) }

var observer string
var client pb.HealthServiceClient
var handle uint64
var empty pb.Empty

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

func register() error {
	if len(observer) == 0 {
		observer = "client"
	}
	reply, err := client.Register(context.Background(), &pb.RegisterRequest{Module: "default", Observer: observer})
	if err != nil {
		return err
	}
	handle = reply.Handle
	return nil
}

func exeGet(args []string) {
	if len(args) == 1 {
		fmt.Println(cmdHelp)
		return
	}
	switch args[1] {
	case "report":
		{
			if len(args) != 3 {
				fmt.Println(cmdHelp)
				return
			}
			report, err := client.GetLatestReport(context.Background(), &pb.GetReportRequest{Subject: args[2]})
			if err == nil {
				fmt.Println(report)
			} else {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
			}
		}
	case "view":
		{
			if len(args) != 4 {
				fmt.Println(cmdHelp)
				return
			}
			view, err := client.GetView(context.Background(), &pb.GetViewRequest{Observer: args[2], Subject: args[3]})
			if err == nil {
				dt.DumpView(os.Stdout, view)
			} else {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
			}
		}
	case "panorama":
		{
			if len(args) != 3 {
				fmt.Println(cmdHelp)
				return
			}
			pano, err := client.GetPanorama(context.Background(), &pb.GetPanoramaRequest{Subject: args[2]})
			if err == nil {
				dt.DumpPanorama(os.Stdout, pano)
			} else {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
			}
		}
	case "inference":
		{
			if len(args) != 3 {
				fmt.Println(cmdHelp)
			}
			inference, err := client.GetInference(context.Background(), &pb.GetInferenceRequest{Subject: args[2]})
			if err == nil {
				fmt.Println(dt.InferenceString(inference))
			} else {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
			}
		}
	default:
		fmt.Println(cmdHelp)
	}
}

func exeDump(args []string) {
	if len(args) != 2 {
		fmt.Println(cmdHelp)
		return
	}
	switch args[1] {
	case "panorama":
		{
			tenants, err := client.DumpPanorama(context.Background(), &empty)
			if err == nil {
				keys := make([]string, 0, len(tenants.Panoramas))
				for key := range tenants.Panoramas {
					keys = append(keys, key)
				}
				sortSubjects(keys)
				for _, key := range keys {
					fmt.Printf("=============%s=============\n", key)
					dt.DumpPanorama(os.Stdout, tenants.Panoramas[key])
				}
			} else {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
			}
		}
	case "inference":
		{
			tenants, err := client.DumpInference(context.Background(), &empty)
			if err == nil {
				keys := make([]string, 0, len(tenants.Inferences))
				for key := range tenants.Inferences {
					keys = append(keys, key)
				}
				sortSubjects(keys)
				for _, key := range keys {
					fmt.Printf("=============%s=============\n", key)
					fmt.Println(dt.InferenceString(tenants.Inferences[key]))
				}
			} else {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
			}
		}
	default:
		fmt.Println(cmdHelp)
	}
}

func runCmd(args []string) bool {
	cmd := args[0]
	switch cmd {
	case "ping":
		{
			now := time.Now()
			pnow, err := ptypes.TimestampProto(now)
			if err == nil {
				request := &pb.PingRequest{
					Source: &pb.Peer{Id: string(observer), Addr: "localhost"},
					Time:   pnow,
				}
				reply, err := client.Ping(context.Background(), request)
				if err == nil {
					t, err := ptypes.Timestamp(reply.Time)
					if err == nil {
						fmt.Println("ping reply at time %s", t)
					}
				}
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
				return false
			}
		}
	case "report":
		{
			r := parseReport(args)
			reply, err := client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Handle: handle, Report: r})
			switch reply.Result {
			case pb.SubmitReportReply_ACCEPTED:
				fmt.Println("Accepted")
			case pb.SubmitReportReply_IGNORED:
				fmt.Println("Ignored")
			case pb.SubmitReportReply_FAILED:
				fmt.Println("Failed")
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
				return false
			}
		}
	case "get":
		exeGet(args)
		return false
	case "dump":
		exeDump(args)
		return false
	case "tail":
		{
			if len(args) < 3 {
				fmt.Println(cmdHelp)
				return false
			}
			freq, err := strconv.Atoi(args[1])
			if err != nil || freq <= 0 {
				fmt.Println("Error, frequency must be a positive integer")
				fmt.Println(cmdHelp)
				return false
			}
			switch args[2] {
			case "get":
				for {
					exeGet(args[2:])
					time.Sleep(time.Duration(freq) * time.Second)
				}
			case "dump":
				for {
					exeDump(args[2:])
					time.Sleep(time.Duration(freq) * time.Second)
				}
			default:
				fmt.Println(cmdHelp)
			}
			return false
		}
	case "list":
		{
			if len(args) != 2 {
				fmt.Println(cmdHelp)
				return false
			}
			switch args[1] {
			case "subject":
				{
					reply, err := client.GetObservedSubjects(context.Background(), &empty)
					if err == nil {
						for subject, ts := range reply.Subjects {
							t, _ := ptypes.Timestamp(ts)
							fmt.Printf("%s\t%s\n", subject, t)
						}
					} else {
						fmt.Fprintln(os.Stderr, grpc.ErrorDesc(err))
					}
				}
			default:
				fmt.Println(cmdHelp)
				return false
			}
		}
	case "me":
		if len(args) == 1 {
			fmt.Println(observer)
		} else {
			observer = args[1]
			err := register()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fail to register new observer with the health service: %v\n", err)
				return false
			}
		}
	case "help":
		fmt.Println(cmdHelp)
	case "exit":
		return true
	case "quit":
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
		fmt.Printf("Usage: %s [options] [command <args...>]\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println("\nIf no command was specified, the client enters an interactive mode.\n")
		fmt.Println(cmdHelp)
	}
	flag.Parse()
	args := flag.Args()
	if len(args) > 1 && (args[0] == "-h" || args[0] == "--help") {
		flag.Usage()
		os.Exit(1)
	}
	addr := *server
	if len(*server) == 0 {
		host, err := os.Hostname()
		if err != nil {
			fmt.Printf("Fail to get host name. Use localhost instead")
			host = "localhost"
		} else {
			host = strings.Split(host, ".")[0]
		}
		addr = host + ":6688"
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Could not connect to %s: %v", *server, err))
	}
	defer conn.Close()
	client = pb.NewHealthServiceClient(conn)
	err = register()
	if err != nil {
		panic(fmt.Sprintf("Fail to register with the health service: %v", err))
	}
	if len(args) == 0 {
		runPrompt()
		fmt.Println()
	} else {
		runCmd(args)
	}
}
