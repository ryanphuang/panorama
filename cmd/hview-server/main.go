package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"deephealth/service"
	dt "deephealth/types"
)

var (
	rc        = flag.String("config", "", "use config file to initialize service")
	addr      = flag.String("addr", "localhost", "server listen address")
	grpc      = flag.Bool("grpc", true, "use grpc service implementation")
	portstart = flag.Int("port_start", 10000, "start of port range for a random port")
	portend   = flag.Int("port_end", 30000, "end of port range for a random port")
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] ID\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	var config *dt.HealthServerConfig
	var err error
	if len(*rc) > 0 {
		config, err = dt.LoadConfig(*rc)
		if err != nil {
			panic(err)
		}
		myaddr, ok := config.Peers[config.Id]
		if !ok {
			panic("Id is not present in peers")
		}
		if len(config.Addr) == 0 {
			config.Addr = myaddr
		} else if config.Addr != myaddr {
			panic("Addr is not the same as the one in peers")
		}
	} else {
		faddr := *addr
		if !strings.ContainsAny(*addr, ":") {
			port := *portstart + int(r.Intn(*portend-*portstart))
			faddr = fmt.Sprintf("%s:%d", faddr, port)
		}
		args := flag.Args()
		if len(args) != 1 {
			flag.Usage()
			os.Exit(1)
		}
		subjects := make([]dt.EntityId, 100)
		for i := 1; i <= 100; i++ {
			subjects[i-1] = dt.EntityId(fmt.Sprintf("TS_%d", i))
		}
		config = &dt.HealthServerConfig{
			Addr:     faddr,
			Id:       dt.EntityId(args[0]),
			Subjects: subjects,
		}
	}
	fmt.Printf("Starting health service at %s with config %s\n", config.Addr, config)
	if *grpc {
		gs := service.NewHealthGServer(config)
		errch := make(chan error)
		gs.Start(errch)
		<-errch
	} else {
		ns := service.NewHealthNServer(config)
		ns.Start()
		ns.WaitForDone()
	}
}
