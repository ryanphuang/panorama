package service

import (
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "deephealth/build/gen"
	"deephealth/store"
	dt "deephealth/types"
)

type HealthGServer struct {
	Addr    string
	Owner   dt.EntityId
	Storage dt.HealthStorage

	l net.Listener
	s *grpc.Server
}

func NewHealthGServer(config *dt.HealthServerConfig) *HealthGServer {
	gs := new(HealthGServer)
	gs.Addr = config.Addr
	gs.Owner = config.Owner

	storage := store.NewRawHealthStorage(config.Subjects...)
	gs.Storage = storage
	return gs
}

func (self *HealthGServer) Start(errch chan error) error {
	if self.s != nil {
		return fmt.Errorf("HealthGServer is already started\n")
	}
	lis, err := net.Listen("tcp", self.Addr)
	if err != nil {
		return fmt.Errorf("Fail to register RPC server at %s\n", self.Addr)
	}
	self.l = lis
	self.s = grpc.NewServer()
	pb.RegisterHealthServiceServer(self.s, self)
	// Register reflection service on gRPC server.
	reflection.Register(self.s)
	go func() {
		if err := self.s.Serve(self.l); err != nil {
			if errch != nil {
				errch <- err
			}
		}
	}()
	return nil
}

func (self *HealthGServer) Stop(graceful bool) error {
	if self.s == nil {
		return fmt.Errorf("HealthGServer has not started\n")
	}
	if graceful {
		self.s.GracefulStop()
	} else {
		self.s.Stop()
	}
	self.s = nil
	self.l = nil
	return nil
}

func (s *HealthGServer) SubmitReport(ctx context.Context, in *pb.SubmitReportRequest) (*pb.SubmitReportReply, error) {

	return &pb.SubmitReportReply{Result: pb.SubmitReportReply_IGNORED}, nil
}

func (s *HealthGServer) GetReport(ctx context.Context, in *pb.GetReportRequest) (*pb.GetReportReply, error) {
	var report pb.Report
	return &pb.GetReportReply{Report: &report}, nil
}

func (s *HealthGServer) ObserveSubject(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	return &pb.ObserveReply{Success: true}, nil
}

func (s *HealthGServer) StopObservingSubject(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	return &pb.ObserveReply{Success: true}, nil
}
