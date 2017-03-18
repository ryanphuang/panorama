package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"deephealth/client"
	dt "deephealth/types"
)

const (
	help = `Usage:
	hview-client <server address> [command <args...>]

With no command specified to enter interactive mode. 
` + cmdHelp

	cmdHelp = `Command list:
	 me observer
	 report subject [<metric:status:score...>]
	 get subject
	 help
	 exit
`
)

func logError(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}

var observer dt.EntityId

func runCmd(c *client.NClient, args []string) bool {
	var subject dt.EntityId
	var report dt.Report
	var observation *dt.Observation
	var metric string
	var status dt.Status
	var err error
	var ret int
	var score float64

	cmd := args[0]
	switch cmd {
	case "report":
		subject = dt.EntityId(args[1])
		observation = dt.NewObservation(time.Now())
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
		report = dt.Report{
			Observer:    observer,
			Subject:     subject,
			Observation: *observation,
		}
		err := c.AddReport(&report, &ret)
		if err == nil {
			fmt.Println("Submitted")
		} else {
			logError(err)
		}

	case "get":
		subject = dt.EntityId(args[1])
		logError(c.GetReport(subject, &report))
		fmt.Println(report)
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

func runPrompt(c *client.NClient) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")

	for scanner.Scan() {
		line := scanner.Text()
		args := fields(line)
		if len(args) > 0 {
			if runCmd(c, args) {
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
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, help)
		os.Exit(1)
	}

	addr := args[0]
	c := client.NewClient(addr, false)

	cmdArgs := args[1:]
	if len(cmdArgs) == 0 {
		runPrompt(c)
		fmt.Println()
	} else {
		runCmd(c, cmdArgs)
	}
}
