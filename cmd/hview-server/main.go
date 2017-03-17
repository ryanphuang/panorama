package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	dh "deephealth"
	"deephealth/service"
	"deephealth/store"
)

var (
	addr      = flag.String("addr", "localhost", "server listen address")
	portstart = flag.Int("port_start", 10000, "start of port range for a random port")
	portend   = flag.Int("port_end", 30000, "end of port range for a random port")
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	flag.Parse()
	faddr := *addr
	if !strings.ContainsAny(*addr, ":") {
		port := *portstart + int(r.Intn(*portend-*portstart))
		faddr = fmt.Sprintf("%s:%d", faddr, port)
	}
	fmt.Printf("Starting health service at %s\n", faddr)
	subjects := make([]dh.EntityId, 100)
	for i := 1; i <= 100; i++ {
		subjects[i-1] = dh.EntityId(fmt.Sprintf("TS_%d", i))
	}
	storage := store.NewRawHealthStorage(subjects...)
	hs := service.NewHealthService(faddr, "XFE_1", storage)
	hs.Start()
	<-hs.Done
}
