package service

import (
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "deephealth/build/gen"
	"deephealth/decision"
	"deephealth/store"
	dt "deephealth/types"
)

type HealthGServer struct {
	HealthServerConfig
	storage   dt.HealthStorage
	inference dt.HealthInference
	gossip    dt.HealthGossip
	l         net.Listener
	s         *grpc.Server
}

func NewHealthGServer(config *HealthServerConfig) *HealthGServer {
	gs := new(HealthGServer)
	gs.HealthServerConfig = *config
	storage := store.NewRawHealthStorage(config.Subjects...)
	gs.storage = storage
	var majority decision.SimpleMajorityInference
	gs.inference = store.NewHealthInferenceStorage(storage, majority)
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

func (self *HealthGServer) SubmitReport(ctx context.Context, in *pb.SubmitReportRequest) (*pb.SubmitReportReply, error) {
	report := dt.ReportFromPb(in.Report)
	if report == nil {
		return &pb.SubmitReportReply{Result: pb.SubmitReportReply_FAILED}, fmt.Errorf("Fail to parse report")
	}
	var result pb.SubmitReportReply_Status
	rc, err := self.storage.AddReport(report, self.FilterSubmission)
	switch rc {
	case store.REPORT_IGNORED:
		result = pb.SubmitReportReply_IGNORED
	case store.REPORT_FAILED:
		result = pb.SubmitReportReply_FAILED
	case store.REPORT_ACCEPTED:
		result = pb.SubmitReportReply_ACCEPTED
	}
	return &pb.SubmitReportReply{Result: result}, err
}

func (self *HealthGServer) GetLatestReport(ctx context.Context, in *pb.GetReportRequest) (*pb.GetReportReply, error) {
	report := self.storage.GetLatestReport(dt.EntityId(in.Subject))
	return &pb.GetReportReply{Report: dt.ReportToPb(report)}, nil
}

func (self *HealthGServer) Observe(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	ok := self.storage.AddSubject(dt.EntityId(in.Subject))
	return &pb.ObserveReply{Success: ok}, nil
}

func (self *HealthGServer) StopObserving(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	ok := self.storage.RemoveSubject(dt.EntityId(in.Subject), true)
	return &pb.ObserveReply{Success: ok}, nil
}
