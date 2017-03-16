package deephealth

import (
	"net"
	"net/rpc"
)

const (
	tag = "service"
)

type HealthService struct {
	Addr    string
	EId     EntityId
	Storage HealthStorage

	alive    bool
	listener net.Listener
}

func NewHealthService(addr string, eid EntityId, storage HealthStorage) *HealthService {
	hs := new(HealthService)
	hs.Addr = addr
	hs.EId = eid
	hs.alive = true
	return hs
}

func (hs *HealthService) Start() {
	server := rpc.NewServer()
	err := server.Register(hs.Storage)
	if err != nil {
		LogF(tag, "Fail to register RPC server")
	}
	listener, err := net.Listen("tcp", hs.Addr)
	if err != nil {
		LogF(tag, "Fail to listen to address %s", hs.Addr)
	}
	hs.listener = listener
	go func() {
		for hs.alive {
			conn, err := hs.listener.Accept()
			if err == nil {
				go func() {
					server.ServeConn(conn)
					conn.Close()
				}()
			} else {
				LogE(tag, "Fail to accept connection %s", err)
			}
		}
	}()
}

func (hs *HealthService) Stop() {
	hs.alive = false
	hs.listener.Close()
}
