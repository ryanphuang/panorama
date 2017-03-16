package main

import (
	"deephealth/config"
	"deephealth/util"
	"flag"
	"fmt"
	"math/rand"
	"time"
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
			util.LogF("%s", "Port range must be positive")
		}
		if *portstart > *portend {
			util.LogF("%s", "Port start must not exceed port end")
		}
		p = *portstart + int(r.Intn(*portend-*portstart))
		inc_p = true
	}
	if *localhost {
		s = 0
		inc_s = false
	} else {
		if *sidstart < 0 {
			util.LogF("%s", "Server id must be positive")
		}
		s = *sidstart
		inc_s = true
	}
	rc := new(config.RC)
	rc.Servers = make([]string, *nserver)
	for i := 0; i < *nserver; i++ {
		if *localhost {
			rc.Servers[i] = fmt.Sprintf("localhost:%d", p)
		} else {
			rc.Servers[i] = fmt.Sprintf(*serverp+":%d", s, p)
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
