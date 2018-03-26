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
		me:               &pb.Peer{string(config.Id), config.Addr},
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

func (self *ExchangeProtocol) Propagate(report *pb.Report) error {
	var ferr error
	request := &pb.LearnReportRequest{Source: self.me, Report: report}
	self.mu.RLock()
	ignoreset, ok := self.SkipSubjectPeers[report.Subject]
	if !ok {
		ignoreset = nil
	}
	self.mu.RUnlock()
	for peer, addr := range self.Peers {
		if peer == self.Id {
			continue // skip send to self
		}
		if ignoreset != nil {
			if ignoreset.Test(peer) {
				continue
			}
		}
		client, err := self.getOrMakeClient(peer)
		if err != nil {
			du.LogE(etag, "failed to get client for %s", peer)
			ferr = err
			continue
		}
		reply, err := client.LearnReport(context.Background(), request)
		if err != nil {
			du.LogE(etag, "failed to propagate report about %s to %s", report.Subject, peer)
			ferr = err
			continue
		}
		du.LogD(etag, "propagated report about %s to %s at %s", report.Subject, peer, addr)
		if reply.Result == pb.LearnReportReply_IGNORED {
			if ignoreset == nil {
				ignoreset = NewIgnoreSet()
				self.mu.Lock()
				self.SkipSubjectPeers[report.Subject] = ignoreset
				self.mu.Unlock()
			}
			ignoreset.Set(peer)
			du.LogD(etag, "ignore report on subject %s from %s in the future", report.Subject, peer)
		}
	}
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
	ignoreset, ok := self.SkipSubjectPeers[subject]
	if !ok { // no ignoreset yet, great
		return false
	}
	self.mu.Unlock()
	ignoreset.Remove(subject, peer)
	return true
}

func (self *ExchangeProtocol) Uninterested(peer string, subject string) bool {
	self.mu.Lock()
	ignoreset, ok := self.SkipSubjectPeers[subject]
	if !ok {
		ignoreset = NewIgnoreSet()
		self.SkipSubjectPeers[subject] = ignoreset
	}
	self.mu.Unlock()
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
