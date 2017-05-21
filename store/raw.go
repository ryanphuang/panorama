package store

import (
	"fmt"
	"os"
	"sync"

	pb "deephealth/build/gen"
	dt "deephealth/types"
	du "deephealth/util"

	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	MaxReportPerView = 5 // maximum number of reports to store for a given view
	tag              = "store"
)

const (
	REPORT_IGNORED int = iota
	REPORT_ACCEPTED
	REPORT_FAILED
)

type RawHealthStorage struct {
	Tenants   map[string]*pb.Panorama
	Locks     map[string]*sync.Mutex
	Watchlist map[string]bool

	mu *sync.Mutex
}

func NewRawHealthStorage(subjects ...string) *RawHealthStorage {
	store := &RawHealthStorage{
		Tenants:   make(map[string]*pb.Panorama),
		Locks:     make(map[string]*sync.Mutex),
		Watchlist: make(map[string]bool),

		mu: &sync.Mutex{},
	}
	var panorama *pb.Panorama
	for _, subject := range subjects {
		store.Watchlist[subject] = true
		store.Locks[subject] = new(sync.Mutex)
		panorama = new(pb.Panorama)
		panorama.Subject = subject
		panorama.Views = make(map[string]*pb.View)
		store.Tenants[subject] = panorama
	}
	return store
}

var _ dt.HealthStorage = new(RawHealthStorage)

func (self *RawHealthStorage) AddSubject(subject string) bool {
	self.mu.Lock()
	_, ok := self.Watchlist[subject]
	self.Watchlist[subject] = true
	self.mu.Unlock()
	return !ok
}

func (self *RawHealthStorage) RemoveSubject(subject string, clean bool) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	_, ok := self.Watchlist[subject]
	delete(self.Watchlist, subject)
	if clean {
		delete(self.Tenants, subject)
		delete(self.Locks, subject)
	}
	return ok
}

func (self *RawHealthStorage) AddReport(report *pb.Report, filter bool) (int, error) {
	self.mu.Lock()
	_, ok := self.Watchlist[report.Subject]
	if !ok {
		if filter {
			// subject is not in our watch list, ignore the report
			du.LogI(tag, "%s not in watch list, ignore report...", report.Subject)
			self.mu.Unlock()
			return REPORT_IGNORED, nil
		} else {
			self.Watchlist[report.Subject] = true
		}
	}
	du.LogD(tag, "add report for %s from %s...", report.Subject, report.Observer)
	l, ok := self.Locks[report.Subject]
	if !ok {
		l = new(sync.Mutex)
		self.Locks[report.Subject] = l
	}
	panorama, ok := self.Tenants[report.Subject]
	if !ok {
		panorama = &pb.Panorama{
			Subject: report.Subject,
			Views:   make(map[string]*pb.View),
		}
		self.Tenants[report.Subject] = panorama
	}
	self.mu.Unlock()
	l.Lock()
	defer l.Unlock()
	view, ok := panorama.Views[report.Observer]
	if !ok {
		view = &pb.View{
			Observer:     report.Observer,
			Subject:      report.Subject,
			Observations: make([]*pb.Observation, 0, MaxReportPerView),
		}
		panorama.Views[report.Observer] = view
		du.LogD(tag, "create view for %s->%s...", report.Observer, report.Subject)
	}
	view.Observations = append(view.Observations, report.Observation)
	du.LogD(tag, "add observation to view %s->%s: %s", report.Observer, report.Subject, dt.ObservationString(report.Observation))
	if len(view.Observations) > MaxReportPerView {
		du.LogD(tag, "truncating list")
		view.Observations = view.Observations[1:]
	}
	return REPORT_ACCEPTED, nil
}

func (self *RawHealthStorage) GetPanorama(subject string) (*pb.Panorama, *sync.Mutex) {
	self.mu.Lock()
	defer self.mu.Unlock()
	_, ok := self.Watchlist[subject]
	if ok {
		l, ok := self.Locks[subject]
		if ok {
			panorama, ok := self.Tenants[subject]
			if ok {
				return panorama, l
			}
		}
	}
	return nil, nil
}

func (self *RawHealthStorage) GetView(observer string, subject string) (*pb.View, *sync.Mutex) {
	self.mu.Lock()
	defer self.mu.Unlock()
	_, ok := self.Watchlist[subject]
	if ok {
		l, ok := self.Locks[subject]
		if ok {
			panorama, ok := self.Tenants[subject]
			if ok {
				view, ok := panorama.Views[observer]
				if ok {
					return view, l
				}
			}
		}
	}
	return nil, nil
}

func (self *RawHealthStorage) GetLatestReport(subject string) *pb.Report {
	self.mu.Lock()
	l, ok := self.Locks[subject]
	if !ok {
		return nil
	}
	self.mu.Unlock()
	l.Lock()
	defer l.Unlock()
	panorama, ok := self.Tenants[subject]
	if !ok {
		return nil
	}
	var max_ts *timestamp.Timestamp
	var recent_ob *pb.Observation
	var who string
	first := true
	for observer, view := range panorama.Views {
		for _, val := range view.Observations {
			if first || dt.CompareTimestamp(max_ts, val.Ts) < 0 {
				first = false
				max_ts = val.Ts
				recent_ob = val
				who = observer
			}
		}
	}
	if recent_ob == nil {
		return nil
	}
	return &pb.Report{
		Observer:    who,
		Subject:     subject,
		Observation: recent_ob,
	}
}

func (self *RawHealthStorage) Dump() {
	for subject, panorama := range self.Tenants {
		fmt.Printf("=============%s=============\n", subject)
		dt.DumpPanorama(os.Stdout, panorama)
	}
}
