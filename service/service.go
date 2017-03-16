package service

import (
	"fmt"
	"net"
	"net/rpc"

	dh "deephealth"
	"deephealth/store"
)

const (
	tag = "service"
)

type HealthService struct {
	Addr    string
	EId     dh.EntityId
	Storage dh.HealthStorage

	alive    bool
	listener net.Listener
}

type Chat string

func (t *Chat) Len(msg string, length *int) error {
	*length = len(msg)
	return nil
}

func (t *Chat) AddReport(report *dh.Report, reply *int) error {
	fmt.Println("Got report from %s", report.Subject)
	*reply = len(report.Subject)
	return nil
}

type DummyHealthStorage string

func (self *DummyHealthStorage) ObserveSubject(subject dh.EntityId, reply *bool) error {
	return nil
}

func (self *DummyHealthStorage) StopObservingSubject(subject dh.EntityId, reply *bool) error {
	return nil
}

func (self *DummyHealthStorage) AddReport(report *dh.Report, reply *int) error {
	fmt.Printf("Receive report at dummy storage from %s to %s\n", report.Subject, report.Observer)
	*reply = 100
	return nil
}

var _ dh.HealthStorage = new(DummyHealthStorage)
var _ dh.HealthStorage = store.NewRawHealthStorage("XFE_1", "XFE_2", "XFE_3")

func NewHealthService(addr string, eid dh.EntityId, storage dh.HealthStorage) *HealthService {
	hs := new(HealthService)
	hs.Addr = addr
	hs.EId = eid
	hs.Storage = storage
	hs.alive = true
	return hs
}

func (hs *HealthService) Start() {
	server := rpc.NewServer()
	var storage dh.HealthStorage = store.NewRawHealthStorage("XFE_1", "XFE_2", "XFE_3")
	fmt.Println("dummy")
	err := server.Register(storage)
	if err != nil {
		dh.LogF(tag, "Fail to register RPC server")
	}
	listener, err := net.Listen("tcp", hs.Addr)
	if err != nil {
		dh.LogF(tag, "Fail to listen to address %s", hs.Addr)
	}
	hs.listener = listener
	for hs.alive {
		conn, err := hs.listener.Accept()
		if err == nil {
			go func() {
				fmt.Println("Accepted a connection!")
				server.ServeConn(conn)
				fmt.Println("Done with a connection!")
				conn.Close()
			}()
		} else {
			fmt.Println("Failed!!!")
			dh.LogE(tag, "Fail to accept connection %s", err)
		}
	}
}

func (hs *HealthService) Stop() {
	hs.alive = false
	hs.listener.Close()
}
