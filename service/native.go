package service

import (
	"net"
	"net/rpc"

	dh "deephealth"
	"deephealth/store"
	dt "deephealth/types"
)

const (
	tag = "native_service"
)

type HealthNServer struct {
	Addr    string
	Owner   dt.EntityId
	Storage dt.HealthStorage
	Done    chan bool

	alive    bool
	listener net.Listener
}

func NewHealthNServer(config *dt.HealthServerConfig) *HealthNServer {
	hs := new(HealthNServer)
	hs.Addr = config.Addr
	hs.Owner = config.Owner

	storage := store.NewRawHealthStorage(config.Subjects...)
	hs.Storage = storage
	hs.Done = make(chan bool)
	hs.alive = true
	return hs
}

func (hs *HealthNServer) ObserveSubject(subject dt.EntityId, reply *bool) error {
	*reply = hs.Storage.ObserveSubject(subject)
	return nil
}

func (hs *HealthNServer) StopObservingSubject(subject dt.EntityId, reply *bool) error {
	*reply = hs.Storage.StopObservingSubject(subject)
	return nil
}

func (hs *HealthNServer) AddReport(report *dt.Report, reply *int) error {
	*reply = hs.Storage.AddReport(report)
	return nil
}

func (hs *HealthNServer) GossipReport(report *dt.Report, reply *int) error {
	return nil
}

func (hs *HealthNServer) GetReport(subject dt.EntityId, report *dt.Report) error {
	return nil
}

func (hs *HealthNServer) Start() error {
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

func (hs *HealthNServer) Stop() {
	hs.alive = false
	hs.listener.Close()
	hs.Done <- true
}
