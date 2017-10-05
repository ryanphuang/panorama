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
	du "deephealth/util"
)

var (
	rc        = flag.String("config", "", "use config file to initialize service")
	addr      = flag.String("addr", "localhost", "server listen address")
	portstart = flag.Int("port_start", 10000, "start of port range for a random port")
	portend   = flag.Int("port_end", 30000, "end of port range for a random port")
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] [ID]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	config := new(dt.HealthServerConfig)
	var err error
	if len(*rc) > 0 {
		err = dt.LoadConfig(*rc, config)
		if err != nil {
			panic(err)
		}
		du.SetLogLevelString(config.LogLevel)
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
		config = &dt.HealthServerConfig{
			Addr: faddr,
			Id:   args[0],
		}
	}
	fmt.Printf("Starting health service at %s with config %s\n", config.Addr, config)
	gs := service.NewHealthGServer(config)
	errch := make(chan error)
	gs.Start(errch)
	<-errch
	fmt.Println("Encountered error, exit.")
	os.Exit(1)
}
