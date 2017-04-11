package service

import (
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	dh "deephealth"
	pb "deephealth/build/gen"
	"deephealth/decision"
	"deephealth/exchange"
	"deephealth/store"
	dt "deephealth/types"
)

const (
	stag = "service"
)

type HealthGServer struct {
	dt.HealthServerConfig
	storage   dt.HealthStorage
	inference dt.HealthInference
	exchange  dt.HealthExchange

	rch chan *pb.Report
	l   net.Listener
	s   *grpc.Server
}

func NewHealthGServer(config *dt.HealthServerConfig) *HealthGServer {
	gs := new(HealthGServer)
	gs.HealthServerConfig = *config
	storage := store.NewRawHealthStorage(config.Subjects...)
	gs.storage = storage
	var majority decision.SimpleMajorityInference
	infs := store.NewHealthInferenceStorage(storage, majority)
	gs.inference = infs
	gs.exchange = exchange.NewExchangeProtocol(config)
	gs.rch = infs.ReportCh
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
	self.inference.Start()
	self.exchange.PingAll()
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
	self.inference.Stop()
	return nil
}

func (self *HealthGServer) SubmitReport(ctx context.Context, in *pb.SubmitReportRequest) (*pb.SubmitReportReply, error) {
	report := in.Report
	var result pb.SubmitReportReply_Status
	rc, err := self.storage.AddReport(report, false) // never ignore local reports
	switch rc {
	case store.REPORT_IGNORED:
		result = pb.SubmitReportReply_IGNORED
	case store.REPORT_FAILED:
		result = pb.SubmitReportReply_FAILED
	case store.REPORT_ACCEPTED:
		result = pb.SubmitReportReply_ACCEPTED
		go self.AnalyzeReport(report)
		go self.exchange.Propagate(report)
	}
	return &pb.SubmitReportReply{Result: result}, err
}

func (self *HealthGServer) LearnReport(ctx context.Context, in *pb.LearnReportRequest) (*pb.LearnReportReply, error) {
	report := in.Report
	dh.LogD(stag, "learn report about %s from %s at %s", report.Subject, in.Source.Id, in.Source.Addr)
	var result pb.LearnReportReply_Status
	rc, err := self.storage.AddReport(report, self.FilterSubmission)
	switch rc {
	case store.REPORT_IGNORED:
		result = pb.LearnReportReply_IGNORED
	case store.REPORT_FAILED:
		result = pb.LearnReportReply_FAILED
	case store.REPORT_ACCEPTED:
		result = pb.LearnReportReply_ACCEPTED
		go self.AnalyzeReport(report)
		go self.exchange.Interested(in.Source.Id, report.Subject)
	}
	return &pb.LearnReportReply{Result: result}, err
}

func (self *HealthGServer) GetLatestReport(ctx context.Context, in *pb.GetReportRequest) (*pb.Report, error) {
	report := self.storage.GetLatestReport(in.Subject)
	return report, nil
}

func (self *HealthGServer) GetPanorama(ctx context.Context, in *pb.GetPanoramaRequest) (*pb.Panorama, error) {
	panorama, _ := self.storage.GetPanorama(in.Subject)
	return panorama, nil
}

func (self *HealthGServer) GetView(ctx context.Context, in *pb.GetViewRequest) (*pb.View, error) {
	view, _ := self.storage.GetView(in.Subject, in.Observer)
	return view, nil
}

func (self *HealthGServer) GetInference(ctx context.Context, in *pb.GetInferenceRequest) (*pb.Inference, error) {
	inference := self.inference.GetInference(in.Subject)
	if inference == nil {
		return nil, fmt.Errorf("inference does not exist for view")
	}
	return inference, nil
}

func (self *HealthGServer) Observe(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	ok := self.storage.AddSubject(in.Subject)
	return &pb.ObserveReply{Success: ok}, nil
}

func (self *HealthGServer) StopObserving(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	ok := self.storage.RemoveSubject(in.Subject, true)
	return &pb.ObserveReply{Success: ok}, nil
}

func (self *HealthGServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingReply, error) {
	ts, err := ptypes.Timestamp(in.Time)
	if err != nil {
		return nil, err
	}
	dh.LogD(stag, "got ping request from %s at time %s", in.Source.Id, ts)
	now := time.Now()
	pnow, err := ptypes.TimestampProto(now)
	if err != nil {
		return nil, err
	}
	return &pb.PingReply{Result: pb.PingReply_GOOD, Time: pnow}, nil
}

func (self *HealthGServer) AnalyzeReport(report *pb.Report) {
	self.rch <- report
	dh.LogD(stag, "sent report for %s for inference", report.Subject)
	/*
		select {
		case self.rch <- report:
			dh.LogD(stag, "send report for %s for inference", report.Subject)
		default:
			dh.LogD(stag, "fail to send report for %s for inference", report.Subject)
		}
	*/
}
