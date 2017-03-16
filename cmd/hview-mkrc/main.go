package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	dh "deephealth"
)

var (
	nserver   = flag.Int("nserver", 3, "number of hview server")
	localhost = flag.Bool("localhost", true, "whether all servers are localhost")
	serverp   = flag.String("serverp", "10.10.2.%d", "pattern of server address")
	sidstart  = flag.Int("sidstart", 1, "start of the server id in server pattern")
	portstart = flag.Int("port_start", 10000, "start of port range for a random port")
	portend   = flag.Int("port_end", 30000, "end of port range for a random port")
	fixport   = flag.Int("fix_port", -1, "fix port instead of random port number")
	output    = flag.String("output", "", "file path to output the generated RC")
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	flag.Parse()
	var p, s int
	var inc_p, inc_s bool
	if *fixport > 0 {
		p = *fixport
		inc_p = false
	} else {
		if *portstart <= 0 || *portend <= 0 {
			dh.LogF("%s", "Port range must be positive")
		}
		if *portstart > *portend {
			dh.LogF("%s", "Port start must not exceed port end")
		}
		p = *portstart + int(r.Intn(*portend-*portstart))
		inc_p = true
	}
	if *localhost {
		s = 0
		inc_s = false
	} else {
		if *sidstart < 0 {
			dh.LogF("%s", "Server id must be positive")
		}
		s = *sidstart
		inc_s = true
	}
	rc := new(dh.RC)
	rc.HealthServers = make(map[dh.EntityId]string)
	for i := 0; i < *nserver; i++ {
		eid := dh.EntityId(fmt.Sprintf("HS_%d", i+1))
		if *localhost {
			rc.HealthServers[eid] = fmt.Sprintf("localhost:%d", p)
		} else {
			rc.HealthServers[eid] = fmt.Sprintf(*serverp+":%d", s, p)
		}
		if inc_p {
			p++
		}
		if inc_s {
			s++
		}
	}
	if len(*output) > 0 {
		rc.Save(*output)
	}
	fmt.Printf("%s\n", rc.String())
}
