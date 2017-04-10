package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "deephealth/build/gen"
	dt "deephealth/types"
)

type mRegexp struct {
	*regexp.Regexp
}

func (r *mRegexp) FindStringSubmatchMap(s string) map[string]string {
	groups := make(map[string]string)
	result := r.FindStringSubmatch(s)
	if result == nil {
		return groups
	}
	for i, name := range r.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		groups[name] = result[i]
	}
	return groups
}

type Event struct {
	ts      time.Time
	id      string
	subject string
	context string
	extra   string
}

var (
	config   = flag.String("conf", "logtail.conf", "configuration to the logtail service")
	reg      = mRegexp{regexp.MustCompile(`^(?P<time>[0-9,-: ]+) \[myid:(?P<id>\d+)\] - (?P<level>[A-Z]+) +\[(?P<tag>.+):(?P<class>[a-zA-Z_\$]+)@(?P<line>[0-9]+)\] - (?P<content>.+)`)}
	commTags = map[string]*regexp.Regexp{
		"RecvWorker": nil,
		"SendWorker": nil,
		"SyncThread": regexp.MustCompile("^Too busy to snap, skipping.*$"),
	}
	selfTags = map[string]*regexp.Regexp{
		"Snapshot Thread": regexp.MustCompile("^Slow serializing node .*$"),
	}
	ipTags = map[string]*regexp.Regexp{
		"LearnerHandler-/": regexp.MustCompile("^Slow serializing node .*$"),
	}
)

const (
	staleSeconds = 5 * 60 // 5 minutes
	mergeSeconds = 5      // merge within 5 seconds
)

var lastReportTime = make(map[string]time.Time)

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] <log file> <server address>...\n\n", os.Args[0])
	flag.PrintDefaults()
}

func parseEvent(line string) *Event {
	result := reg.FindStringSubmatchMap(line)
	if len(result) == 0 {
		return nil
	}
	if result["level"] == "INFO" || result["level"] == "DEBUG" {
		return nil
	}
	fields := strings.Split(result["tag"], ":")
	l := len(fields)
	myid := "peer@" + result["id"]
	subject := myid
	if l == 1 {
		re, ok := selfTags[fields[0]]
		if !ok || !re.MatchString(result["content"]) {
			return nil
		}
	} else if l == 2 {
		re, ok := commTags[fields[0]]
		found := false
		if !ok {
			for pref, cre := range ipTags {
				if strings.HasPrefix(fields[0], pref) {
					if cre.MatchString(result["content"]) {
						found = true
						subject = fields[0][len(pref):]
					}
					break
				}
			}
			if !found {
				return nil
			}
		} else {
			if re != nil && !re.MatchString(result["content"]) {
				return nil
			}
			_, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil
			}
			subject = "peer@" + fields[1]
		}
	} else {
		return nil
	}
	ts, err := time.Parse("2006-01-02 15:04:05", result["time"][:19])
	if err == nil {
		return &Event{ts: ts, id: myid, subject: subject, context: fields[0], extra: result["content"]}
	}
	return nil
}

func reportEvent(client pb.HealthServiceClient, event *Event) error {
	ts, ok := lastReportTime[event.subject]
	if ok && time.Now().Sub(ts).Seconds() < mergeSeconds {
		fmt.Printf("report for %s is too frequent, skip\n", event.subject)
		return nil
	}
	observation := dt.NewPbObservationSingleMetric(event.ts, event.context, pb.Status_UNHEALTHY, 20)
	report := &pb.Report{
		Observer:    event.id,
		Subject:     event.subject,
		Observation: observation,
	}
	reply, err := client.SubmitReport(context.Background(), &pb.SubmitReportRequest{Report: report})
	lastReportTime[event.subject] = time.Now()
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
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	addr := flag.Arg(1)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Could not connect to %s: %v", addr, err))
	}
	defer conn.Close()
	client := pb.NewHealthServiceClient(conn)

	t, _ := tail.TailFile(flag.Arg(0), tail.Config{Follow: true})
	for line := range t.Lines {
		event := parseEvent(line.Text)
		if event != nil {
			/*
				if time.Since(event.ts).Seconds() > staleSeconds {
					fmt.Printf("skip stale event: %s\n", event)
					continue
				}
			*/
			reportEvent(client, event)
		}
	}
}
