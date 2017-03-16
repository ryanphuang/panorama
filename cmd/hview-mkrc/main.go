package main

import (
	"deephealth/util"
	"flag"
	"fmt"
	"math/rand"
	"time"
)

var (
	nserver   = flag.Int("nserver", 3, "number of hview server")
	portstart = flag.Int("port_start", 10000, "start of port range for a random port")
	portend   = flag.Int("port_end", 30000, "end of port range for a random port")
	fixport   = flag.Int("fix_port", -1, "fix port instead of random port number")
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	flag.Parse()
	var p int
	var inc_p bool
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
	for i := 0; i < *nserver; i++ {
		fmt.Printf("localhost:%d\n", p)
		if inc_p {
			p++
		}
	}
}
