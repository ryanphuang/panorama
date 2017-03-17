package service

import (
	"net"
	"net/rpc"

	dh "deephealth"
)

const (
	tag = "service"
)

type HealthService struct {
	Addr    string
	Owner   dh.EntityId
	Storage dh.HealthStorage
	Done    chan bool

	alive    bool
	listener net.Listener
}

func NewHealthService(addr string, eid dh.EntityId, storage dh.HealthStorage) *HealthService {
	hs := new(HealthService)
	hs.Addr = addr
	hs.Owner = eid
	hs.Storage = storage
	hs.Done = make(chan bool)
	hs.alive = true
	return hs
}

func (hs *HealthService) ObserveSubject(subject dh.EntityId, reply *bool) error {
	return hs.Storage.ObserveSubject(subject, reply)
}

func (hs *HealthService) StopObservingSubject(subject dh.EntityId, reply *bool) error {
	return hs.Storage.StopObservingSubject(subject, reply)
}

func (hs *HealthService) AddReport(report *dh.Report, reply *int) error {
	return hs.Storage.AddReport(report, reply)
}

func (hs *HealthService) GossipReport(report *dh.Report, reply *int) error {
	return nil
}

func (hs *HealthService) GetReport(subject dh.EntityId, report *dh.Report) error {
	return nil
}

var _ dh.HealthService = new(HealthService)

func (hs *HealthService) Start() error {
	server := rpc.NewServer()
	err := server.Register(hs)
	if err != nil {
		dh.LogF(tag, "Fail to register RPC server")
	}
	listener, err := net.Listen("tcp", hs.Addr)
	if err != nil {
		dh.LogF(tag, "Fail to listen to address %s", hs.Addr)
	}
	hs.listener = listener
	go func() {
		for hs.alive {
			conn, err := hs.listener.Accept()
			if err == nil {
				go func() {
					dh.LogD(tag, "Accepted a connection!")
					server.ServeConn(conn)
					conn.Close()
				}()
			} else {
				dh.LogE(tag, "Fail to accept connection %s", err)
			}
		}
	}()
	return nil
}

func (hs *HealthService) Stop() {
	hs.alive = false
	hs.listener.Close()
	hs.Done <- true
}
