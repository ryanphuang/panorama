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

type IgnoreSet map[string]bool

type ExchangeProtocol struct {
	Id   string // my id
	Addr string // my addr

	Peers            map[string]string    // all peers' id and address
	SkipSubjectPeers map[string]IgnoreSet // skip sending reports about a subject to certain peers

	Clients map[string]pb.HealthServiceClient // clients to all peers

	me *pb.Peer
	mu *sync.Mutex
}

var _ dt.HealthExchange = new(ExchangeProtocol)

func NewExchangeProtocol(config *dt.HealthServerConfig) *ExchangeProtocol {
	return &ExchangeProtocol{
		Id:               config.Id,
		Addr:             config.Addr,
		Peers:            config.Peers,
		SkipSubjectPeers: make(map[string]IgnoreSet),
		Clients:          make(map[string]pb.HealthServiceClient),
		me:               &pb.Peer{string(config.Id), config.Addr},
		mu:               &sync.Mutex{},
	}
}

func (self *ExchangeProtocol) Propagate(report *pb.Report) error {
	var ferr error
	request := &pb.LearnReportRequest{Source: self.me, Report: report}
	self.mu.Lock()
	ignoreset, ok := self.SkipSubjectPeers[report.Subject]
	if !ok {
		ignoreset = nil
	}
	self.mu.Unlock()
	for peer, addr := range self.Peers {
		if peer == self.Id {
			continue // skip send to self
		}
		if ignoreset != nil {
			_, ok := ignoreset[peer]
			if ok {
				continue
			}
		}
		client, err := self.getOrMakeClient(peer)
		if err != nil {
			ferr = err
			continue
		}
		reply, err := client.LearnReport(context.Background(), request)
		if err != nil {
			ferr = err
			continue
		}
		du.LogD(etag, "propagated report about %s to %s at %s", report.Subject, peer, addr)
		if reply.Result == pb.LearnReportReply_IGNORED {
			self.mu.Lock()
			if ignoreset == nil {
				ignoreset = make(IgnoreSet)
				self.SkipSubjectPeers[report.Subject] = ignoreset
			}
			ignoreset[peer] = true
			self.mu.Unlock()
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
	defer self.mu.Unlock()
	ignoreset, ok := self.SkipSubjectPeers[subject]
	if !ok { // no ignoreset yet, great
		return false
	}
	_, ok = ignoreset[peer]
	delete(ignoreset, peer) // remove peer from the ignoreset
	if ok {
		du.LogD(etag, "removing %s from the ignoreset of peer %s", subject, peer)
	}
	return true
}

func (self *ExchangeProtocol) Uninterested(peer string, subject string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	ignoreset, ok := self.SkipSubjectPeers[subject]
	if !ok {
		ignoreset = make(IgnoreSet)
		self.SkipSubjectPeers[subject] = ignoreset
	}
	ignoreset[peer] = true
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
