package exchange

import (
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "deephealth/build/gen"
	dt "deephealth/types"
	du "deephealth/util"
)

const (
	etag = "exchange"
)

type IgnoreSet struct {
	mu      sync.RWMutex
	entries map[string]bool
}

type ExchangeProtocol struct {
	Id   string // my id
	Addr string // my addr

	Peers            map[string]string     // all peers' id and address
	SkipSubjectPeers map[string]*IgnoreSet // skip sending reports about a subject to certain peers

	Clients map[string]pb.HealthServiceClient // clients to all peers

	me *pb.Peer
	mu sync.RWMutex
}

var _ dt.HealthExchange = new(ExchangeProtocol)

func NewIgnoreSet() *IgnoreSet {
	return &IgnoreSet{
		entries: make(map[string]bool),
	}
}

func NewExchangeProtocol(config *dt.HealthServerConfig) *ExchangeProtocol {
	return &ExchangeProtocol{
		Id:               config.Id,
		Addr:             config.Addr,
		Peers:            config.Peers,
		SkipSubjectPeers: make(map[string]*IgnoreSet),
		Clients:          make(map[string]pb.HealthServiceClient),
		me:               &pb.Peer{Id: string(config.Id), Addr: config.Addr},
	}
}

func (self *IgnoreSet) Test(peer string) bool {
	self.mu.RLock()
	_, ok := self.entries[peer]
	self.mu.RUnlock()
	return ok
}

func (self *IgnoreSet) Set(peer string) {
	self.mu.Lock()
	self.entries[peer] = true
	self.mu.Unlock()
}

func (self *IgnoreSet) Remove(subject string, peer string) {
	self.mu.Lock()
	_, ok := self.entries[peer]
	delete(self.entries, peer) // remove peer from the ignoreset
	if ok {
		du.LogD(etag, "removing %s from the ignoreset of peer %s", subject, peer)
	}
	self.mu.Unlock()
}

func (self *ExchangeProtocol) Subscribe(subject string) error {
	report := &pb.Report{Observer: self.me.Id, Subject: subject}
	request := &pb.LearnReportRequest{Kind: pb.LearnReportRequest_SUBSCRIPTION, Source: self.me, Report: report}
	du.LogI(etag, "subscribe to reports about for %s", report.Subject)
	return self.PropagateAll(request)
}

func (self *ExchangeProtocol) Unsubscribe(subject string) error {
	report := &pb.Report{Observer: self.me.Id, Subject: subject}
	request := &pb.LearnReportRequest{Kind: pb.LearnReportRequest_UNSUBSCRIPTION, Source: self.me, Report: report}
	du.LogI(etag, "unsubscribe to reports about for %s", report.Subject)
	return self.PropagateAll(request)
}

func (self *ExchangeProtocol) Propagate(report *pb.Report) error {
	request := &pb.LearnReportRequest{Kind: pb.LearnReportRequest_NORMAL, Source: self.me, Report: report}
	du.LogI(etag, "about to propagate report about %s", report.Subject)
	return self.PropagateAll(request)
}

// Propagate something to a peer. This something could be a normal report or a
// subscription/unsubscription request. In the former case, we should check
// if the receiver peer is interested in knowing the report. If not, we
// should stop propagating the report to the receiver in the future until
// it becomes interested in the report again.
func (self *ExchangeProtocol) PropagatePeer(peer string, addr string, ignoreset *IgnoreSet, request *pb.LearnReportRequest) (bool, error, time.Duration) {
	if peer == self.Id {
		du.LogI(etag, "skip propagating to self")
		return true, nil, 0 // skip send to self
	}
	report := request.Report
	if ignoreset != nil {
		if ignoreset.Test(peer) {
			du.LogI(etag, "skip propagating report about %s to %s", report.Subject, peer)
			return true, nil, 0
		}
	}
	du.LogI(etag, "propagating report about %s to %s", report.Subject, peer)
	client, err := self.getOrMakeClient(peer)
	if err != nil {
		du.LogE(etag, "failed to get client for %s", peer)
		return false, err, 0
	}
	t1 := time.Now()
	reply, err := client.LearnReport(context.Background(), request)
	duration := time.Since(t1)
	if err != nil {
		du.LogE(etag, "failed to propagate report about %s to %s", report.Subject, peer)
		return false, err, duration
	}
	if request.Kind == pb.LearnReportRequest_NORMAL && reply.Result == pb.LearnReportReply_IGNORED {
		if ignoreset == nil {
			ignoreset = NewIgnoreSet()
			self.mu.Lock()
			self.SkipSubjectPeers[report.Subject] = ignoreset
			self.mu.Unlock()
		}
		ignoreset.Set(peer)
		du.LogI(etag, "stop propgating report on subject %s to %s in the future", report.Subject, peer)
		return true, nil, duration
	} else {
		du.LogI(etag, "propagated report about %s to %s at %s", report.Subject, peer, addr)
		return false, nil, duration
	}
}

func (self *ExchangeProtocol) PropagateAll(request *pb.LearnReportRequest) error {
	var ferr error
	var serial_prop_latency time.Duration
	var prop_subjects int
	var ignore_subjects int
	var wg sync.WaitGroup
	var mu sync.Mutex // local mutex for updating stats in the parallelized loop

	self.mu.RLock()
	report := request.Report
	ignoreset, ok := self.SkipSubjectPeers[report.Subject]
	if !ok {
		ignoreset = nil
	}
	self.mu.RUnlock()
	du.LogD(etag, "ignoreset about %s: %v", report.Subject, ignoreset)

	wg.Add(len(self.Peers))
	t1 := time.Now()
	for peer, addr := range self.Peers {
		// The propagation is now parallelized, blazing fast...
		go func(peer string, addr string) {
			defer wg.Done()
			ignored, err, duration := self.PropagatePeer(peer, addr, ignoreset, request)
			if err != nil {
				ferr = err
			} else {
				mu.Lock()
				if ignored {
					ignore_subjects++
				} else {
					prop_subjects++
				}
				// Count the duration into propagation latency even if the reply
				// is ignored. Presumably, the next time, it won't even be tried (duration=0)
				// This gives us a sense on how fast the propagation can converge
				serial_prop_latency += duration
				mu.Unlock()
			}
		}(peer, addr)
	}
	wg.Wait()
	parallel_prop_latency := time.Since(t1)
	du.LogI(etag, "propagated report to %d subjects in %s (serial latency %s), ignored %d subjects",
		prop_subjects, parallel_prop_latency, serial_prop_latency, ignore_subjects)
	return ferr
}

func (self *ExchangeProtocol) Ping(peer string) (*pb.PingReply, error) {
	client, err := self.getOrMakeClient(peer)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	pnow, err := ptypes.TimestampProto(now)
	if err != nil {
		return nil, err
	}
	request := &pb.PingRequest{Source: self.me, Time: pnow}
	du.LogD(etag, "ping %s at %s", peer, now)
	reply, err := client.Ping(context.Background(), request)
	if err != nil {
		return nil, err
	}
	du.LogD(etag, "got ping reply from %s at %s", peer, ptypes.TimestampString(reply.Time))
	return reply, nil
}

func (self *ExchangeProtocol) PingAll() (map[string]*pb.PingReply, error) {
	var ferr error
	result := make(map[string]*pb.PingReply)
	for peer, _ := range self.Peers {
		if peer == self.Id {
			continue
		}
		reply, err := self.Ping(peer)
		if err != nil {
			ferr = err
			continue
		}
		result[peer] = reply
	}
	return result, ferr
}

func (self *ExchangeProtocol) Interested(peer string, subject string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	ignoreset, ok := self.SkipSubjectPeers[subject]
	if !ok { // no ignoreset yet, great
		return false
	}
	ignoreset.Remove(subject, peer)
	return true
}

func (self *ExchangeProtocol) Uninterested(peer string, subject string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	ignoreset, ok := self.SkipSubjectPeers[subject]
	if !ok {
		ignoreset = NewIgnoreSet()
		self.SkipSubjectPeers[subject] = ignoreset
	}
	ignoreset.Set(peer)
	du.LogD(etag, "stop notifying %s about health of %s in the future", peer, subject)
	return true
}

func (self *ExchangeProtocol) getOrMakeClient(peer string) (pb.HealthServiceClient, error) {
	client, ok := self.Clients[peer]
	if ok {
		return client, nil
	}
	addr := self.Peers[peer]
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client = pb.NewHealthServiceClient(conn)
	self.Clients[peer] = client
	return client, nil
}
