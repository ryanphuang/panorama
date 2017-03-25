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
	HealthServerConfig

	storage  dt.HealthStorage
	done     chan bool
	alive    bool
	listener net.Listener
}

func NewHealthNServer(config *HealthServerConfig) *HealthNServer {
	hs := new(HealthNServer)
	hs.HealthServerConfig = *config

	storage := store.NewRawHealthStorage(config.Subjects...)
	hs.storage = storage
	hs.done = make(chan bool)
	hs.alive = true
	return hs
}

func (hs *HealthNServer) Observe(subject dt.EntityId, reply *bool) error {
	*reply = hs.storage.AddSubject(subject)
	return nil
}

func (hs *HealthNServer) StopObserving(subject dt.EntityId, reply *bool) error {
	*reply = hs.storage.RemoveSubject(subject, true)
	return nil
}

func (hs *HealthNServer) SubmitReport(report *dt.Report, reply *int) error {
	var err error
	*reply, err = hs.storage.AddReport(report, hs.FilterSubmission)
	return err
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
	hs.done <- true
}

func (hs *HealthNServer) WaitForDone() {
	<-hs.done
}
