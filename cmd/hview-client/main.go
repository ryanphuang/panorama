package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	dh "deephealth"
	"deephealth/client"
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

var observer dh.EntityId

func runCmd(c *client.Client, args []string) bool {
	var subject dh.EntityId
	var report dh.Report
	var observation *dh.Observation
	var metric string
	var status dh.Status
	var err error
	var ret int
	var score float64

	cmd := args[0]
	switch cmd {
	case "report":
		subject = dh.EntityId(args[1])
		observation = dh.NewObservation(time.Now())
		for i := 2; i < len(args); i++ {
			parts := strings.Split(args[i], ":")
			if len(parts) == 3 {
				metric = parts[0]
				status = dh.StatusFromStr(parts[1])
				if status == dh.INVALID {
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
			observer = dh.EntityId("XFE_0") // default observer
		}
		report = dh.Report{
			Observer:    observer,
			Subject:     subject,
			Observation: *observation,
		}
		logError(c.AddReport(&report, &ret))
		fmt.Println("Submitted")

	case "get":
		subject = dh.EntityId(args[1])
		logError(c.GetReport(subject, &report))
		fmt.Println(report)
	case "me":
		if len(args) == 1 {
			fmt.Println(observer)
		} else {
			observer = dh.EntityId(args[1])
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

func runPrompt(c *client.Client) {
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
