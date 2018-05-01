package service

import (
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "deephealth/build/gen"
	"deephealth/decision"
	"deephealth/exchange"
	"deephealth/store"
	dt "deephealth/types"
	du "deephealth/util"
)

const (
	stag          = "service"
	HANDLE_START  = 10000
	GC_FREQUENCY  = 3 * time.Minute // frequency to invoke garbage collection
	GC_THRESHOLD  = 5 * time.Minute // TTL threshold
	GC_RELATIVE   = true            // garbage collect based on relative timestamp
	HOLD_TIME     = 3 * time.Minute // time to hold ignored reports
	HOLD_LIST_LEN = 60              // number of items to hold at most for each subject
)

var (
	gc_frequency time.Duration = 0
	gc_threshold time.Duration = 0
	gc_relative                = GC_RELATIVE
)

type HealthGServer struct {
	dt.HealthServerConfig
	storage     dt.HealthStorage
	db          dt.HealthDB
	inference   dt.HealthInference
	exchange    dt.HealthExchange
	hold_buffer *store.CacheList

	handles map[uint64]*dt.ObserverModule
	l       net.Listener
	s       *grpc.Server
}

func NewHealthGServer(config *dt.HealthServerConfig) *HealthGServer {
	gs := new(HealthGServer)
	gs.HealthServerConfig = *config
	storage := store.NewRawHealthStorage(config.Subjects...)
	gs.storage = storage
	gs.handles = make(map[uint64]*dt.ObserverModule)
	// hold ignored entries for 3 minutes
	if config.BufConfig.HoldTime > 0 {
		gs.hold_buffer = store.NewCacheList(time.Duration(config.BufConfig.HoldTime)*time.Second,
			config.BufConfig.HoldListLen)
	} else {
		gs.hold_buffer = store.NewCacheList(HOLD_TIME, HOLD_LIST_LEN)
	}
	if config.GCConfig.Enable {
		if config.GCConfig.Frequency > 0 {
			gc_frequency = time.Duration(config.GCConfig.Frequency) * time.Second
			gc_threshold = time.Duration(config.GCConfig.Threshold) * time.Second
		} else {
			gc_frequency = GC_FREQUENCY
			gc_threshold = GC_THRESHOLD
		}
		gc_relative = config.GCConfig.Relative
	}
	var majority decision.SimpleMajorityInference
	infs := store.NewHealthInferenceStorage(storage, majority)
	gs.inference = infs
	gs.exchange = exchange.NewExchangeProtocol(config)
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
	if len(self.DBFile) > 0 {
		self.db = store.NewHealthDBStorage(self.DBFile)
		_, err = self.db.Open()
		if err == nil {
			self.storage.SetDB(self.db)
			self.inference.SetDB(self.db)
		}
	}
	self.inference.Start()
	self.exchange.PingAll()
	if gc_frequency > 0 {
		// set GC frequency to negative to disable GC
		go self.GC()
	}
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
	if self.db != nil {
		self.db.Close()
	}
	return nil
}

func (self *HealthGServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterReply, error) {
	var max_handle uint64 = HANDLE_START
	for handle, module := range self.handles {
		if module.Module == in.Module && module.Observer == in.Observer {
			return &pb.RegisterReply{handle}, nil
		}
		if handle > max_handle {
			max_handle = handle
		}
	}
	max_handle = max_handle + 1
	self.storage.AddSubject(in.Observer) // should include this local observer into watch list
	self.handles[max_handle] = &dt.ObserverModule{Module: in.Module, Observer: in.Observer}
	du.LogD(stag, "received register request from (%s,%s), assigned handle %d", in.Module, in.Observer, max_handle)
	return &pb.RegisterReply{max_handle}, nil
}

func (self *HealthGServer) SubmitReport(ctx context.Context, in *pb.SubmitReportRequest) (*pb.SubmitReportReply, error) {
	// TODO: validate submission handles here
	_, ok := self.handles[in.Handle]
	if !ok {
		return nil, fmt.Errorf("Invalid submission handle")
	}

	report := in.Report
	var result pb.SubmitReportReply_Status
	du.LogD(stag, "submitting report about %s", report.Subject)
	rc, err := self.storage.AddReport(report, false) // never ignore local reports
	switch rc {
	case store.REPORT_IGNORED:
		return nil, fmt.Errorf("Should not ignore local report. Probably due to a bug")
	case store.REPORT_FAILED:
		result = pb.SubmitReportReply_FAILED
	case store.REPORT_ACCEPTED:
		result = pb.SubmitReportReply_ACCEPTED
		du.LogD(stag, "accepted report about %s, analyzing...", report.Subject)
		go self.AnalyzeReport(report, true)
		du.LogD(stag, "propagating report about %s", report.Subject)
		go self.exchange.Propagate(report)
	}
	return &pb.SubmitReportReply{Result: result}, err
}

func (self *HealthGServer) LearnReport(ctx context.Context, in *pb.LearnReportRequest) (*pb.LearnReportReply, error) {
	report := in.Report
	switch in.Kind {
	case pb.LearnReportRequest_NORMAL:
		{
			du.LogD(stag, "learning report about %s from %s at %s", report.Subject, report.Observer, in.Source.Id)
			var result pb.LearnReportReply_Status
			rc, err := self.storage.AddReport(report, self.FilterSubmission)
			switch rc {
			case store.REPORT_IGNORED:
				result = pb.LearnReportReply_IGNORED
				du.LogD(stag, "ignored about report %s from %s at %s", report.Subject, report.Observer, in.Source.Id)
				self.hold_buffer.Set(report.Subject, report) // put this report on hold for a while
			case store.REPORT_FAILED:
				result = pb.LearnReportReply_FAILED
			case store.REPORT_ACCEPTED:
				result = pb.LearnReportReply_ACCEPTED
				du.LogD(stag, "accepted report %s from %s at %s", report.Subject, report.Observer, in.Source.Id)
				self.exchange.Interested(in.Source.Id, report.Subject)
				go self.AnalyzeReport(report, false)
			}
			return &pb.LearnReportReply{Result: result}, err
		}
	case pb.LearnReportRequest_SUBSCRIPTION:
		{
			du.LogI(stag, "got a subscription request about %s from %s at %s", report.Subject, report.Observer, in.Source.Id)
			self.exchange.Interested(in.Source.Id, report.Subject)
			return &pb.LearnReportReply{Result: pb.LearnReportReply_ACCEPTED}, nil
		}
	case pb.LearnReportRequest_UNSUBSCRIPTION:
		{
			du.LogI(stag, "got a unsubscription request about %s from %s at %s", report.Subject, report.Observer, in.Source.Id)
			self.exchange.Uninterested(in.Source.Id, report.Subject)
			return &pb.LearnReportReply{Result: pb.LearnReportReply_ACCEPTED}, nil
		}
	}
	return &pb.LearnReportReply{Result: pb.LearnReportReply_FAILED}, nil
}

func (self *HealthGServer) GetLatestReport(ctx context.Context, in *pb.GetReportRequest) (*pb.Report, error) {
	report := self.storage.GetLatestReport(in.Subject)
	if report == nil {
		return nil, fmt.Errorf("No report for %s", in.Subject)
	}
	return report, nil
}

func (self *HealthGServer) GetPanorama(ctx context.Context, in *pb.GetPanoramaRequest) (*pb.Panorama, error) {
	pano := self.storage.GetPanorama(in.Subject)
	if pano == nil {
		return nil, fmt.Errorf("No panorama for %s", in.Subject)
	}
	return pano.Value, nil
}

func (self *HealthGServer) GetView(ctx context.Context, in *pb.GetViewRequest) (*pb.View, error) {
	view := self.storage.GetView(in.Subject, in.Observer)
	if view == nil {
		return nil, fmt.Errorf("No view for %s", in.Subject)
	}
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
	go self.exchange.Subscribe(in.Subject) // tell others I'd like to subscribe to subject
	return &pb.ObserveReply{Success: ok}, nil
}

func (self *HealthGServer) StopObserving(ctx context.Context, in *pb.ObserveRequest) (*pb.ObserveReply, error) {
	ok := self.storage.RemoveSubject(in.Subject, true)
	go self.exchange.Unsubscribe(in.Subject) // tell others I'd like to subscribe to subject
	return &pb.ObserveReply{Success: ok}, nil
}

func (self *HealthGServer) GetObservedSubjects(ctx context.Context, in *pb.Empty) (*pb.GetObservedSubjectsReply, error) {
	watchList := self.storage.GetSubjects()
	result := make(map[string]*tspb.Timestamp)
	for subject, ts := range watchList {
		pts, err := ptypes.TimestampProto(ts)
		if err != nil {
			return nil, err
		}
		result[subject] = pts
	}
	return &pb.GetObservedSubjectsReply{result}, nil
}

func (self *HealthGServer) DumpPanorama(ctx context.Context, in *pb.Empty) (*pb.DumpPanoramaReply, error) {
	return &pb.DumpPanoramaReply{self.storage.DumpPanorama()}, nil
}

func (self *HealthGServer) DumpInference(ctx context.Context, in *pb.Empty) (*pb.DumpInferenceReply, error) {
	return &pb.DumpInferenceReply{self.inference.DumpInference()}, nil
}

func (self *HealthGServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingReply, error) {
	ts, err := ptypes.Timestamp(in.Time)
	if err != nil {
		return nil, err
	}
	du.LogD(stag, "got ping request from %s at time %s", in.Source.Id, ts)
	now := time.Now()
	pnow, err := ptypes.TimestampProto(now)
	if err != nil {
		return nil, err
	}
	return &pb.PingReply{Result: pb.PingReply_GOOD, Time: pnow}, nil
}

func (self *HealthGServer) GC() {
	for self.s != nil {
		time.Sleep(gc_frequency)
		retired := self.storage.GC(gc_threshold, gc_relative) // retired reports older then GC_THREASHOLD
		if retired != nil && len(retired) != 0 {
			for subject, r := range retired {
				du.LogD(stag, "Retired %d observations for %s", r, subject)
				// TODO: update inference result here
				self.inference.InferSubjectAsync(subject)
			}
		} else {
			du.LogD(stag, "No observations retired at this GC round")
		}
	}
}

func (self *HealthGServer) AnalyzeReport(report *pb.Report, check_hold bool) {
	if check_hold {
		items := self.hold_buffer.Get(report.Subject)
		if items != nil && len(items) > 0 {
			du.LogI(stag, "found %d recent reports about %s in hold buffer", len(items), report.Subject)
			for _, item := range items {
				r := item.Value.(*pb.Report)
				_, err := self.storage.AddReport(r, false)
				if err != nil {
					du.LogE(stag, "fail to add hold buffer report %s->%s", r.Observer, r.Subject)
				} else {
					du.LogD(stag, "hold buffer report %s->%s successfully added back to storage", r.Observer, r.Subject)
				}
			}
			self.hold_buffer.Empty(report.Subject)     // clear the report from hold buffer
			go self.exchange.Subscribe(report.Subject) // tell others I'd like to subscribe to subject
		}
	}
	du.LogD(stag, "sent report for %s for inference", report.Subject)
	self.inference.InferReportAsync(report)
}

func (self *HealthGServer) GetPeers(ctx context.Context, in *pb.Empty) (*pb.GetPeerReply, error) {
	peers := make([]*pb.Peer, 0, len(self.Peers))
	for id, addr := range self.Peers {
		peers = append(peers, &pb.Peer{Id: id, Addr: addr})
	}
	return &pb.GetPeerReply{Peers: peers}, nil
}

func (self *HealthGServer) GetId(ctx context.Context, in *pb.Empty) (*pb.Peer, error) {
	return &pb.Peer{Id: self.Id, Addr: self.Addr}, nil
}
