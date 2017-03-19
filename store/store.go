package store

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	dh "deephealth"
	dt "deephealth/types"
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
	Tenants   map[dt.EntityId]*dt.Panorama
	Locks     map[dt.EntityId]*sync.Mutex
	Watchlist map[dt.EntityId]bool

	mu *sync.Mutex
}

func NewRawHealthStorage(subjects ...dt.EntityId) *RawHealthStorage {
	store := &RawHealthStorage{
		Tenants:   make(map[dt.EntityId]*dt.Panorama),
		Locks:     make(map[dt.EntityId]*sync.Mutex),
		Watchlist: make(map[dt.EntityId]bool),

		mu: &sync.Mutex{},
	}
	var panorama *dt.Panorama
	for _, subject := range subjects {
		store.Watchlist[subject] = true
		store.Locks[subject] = new(sync.Mutex)
		panorama = new(dt.Panorama)
		panorama.Subject = subject
		panorama.Views = make(map[dt.EntityId]*dt.View)
		store.Tenants[subject] = panorama
	}
	return store
}

var _ dt.HealthStorage = new(RawHealthStorage)

func (self *RawHealthStorage) AddSubject(subject dt.EntityId) bool {
	_, ok := self.Watchlist[subject]
	self.Watchlist[subject] = true
	return !ok
}

func (self *RawHealthStorage) RemoveSubject(subject dt.EntityId, clean bool) bool {
	_, ok := self.Watchlist[subject]
	delete(self.Watchlist, subject)
	if clean {
		delete(self.Tenants, subject)
		delete(self.Locks, subject)
	}
	return ok
}

func (self *RawHealthStorage) AddReport(report *dt.Report) (int, error) {
	_, ok := self.Watchlist[report.Subject]
	if !ok {
		// subject is not in our watch list, ignore the report
		dh.LogI(tag, "%s not in watch list, ignore report...", report.Subject)
		return REPORT_IGNORED, nil
	}
	dh.LogD(tag, "add report for %s from %s...", report.Subject, report.Observer)
	self.mu.Lock()
	l, ok := self.Locks[report.Subject]
	if !ok {
		l = new(sync.Mutex)
		self.Locks[report.Subject] = l
	}
	self.mu.Unlock()
	l.Lock()
	defer l.Unlock()
	panorama, ok := self.Tenants[report.Subject]
	if !ok {
		panorama = &dt.Panorama{
			Subject: report.Subject,
			Views:   make(map[dt.EntityId]*dt.View),
		}
		self.Tenants[report.Subject] = panorama
	}
	view, ok := panorama.Views[report.Observer]
	if !ok {
		view = &dt.View{
			Observer:     report.Observer,
			Subject:      report.Subject,
			Observations: list.New(),
		}
		panorama.Views[report.Observer] = view
		dh.LogD(tag, "create view for %s->%s...", report.Observer, report.Subject)
	}
	view.Observations.PushBack(&report.Observation)
	dh.LogD(tag, "add observation to view %s->%s: %s", report.Observer, report.Subject, report.Observation)
	if view.Observations.Len() > MaxReportPerView {
		dh.LogD(tag, "truncating list")
		view.Observations.Remove(view.Observations.Front())
	}
	return REPORT_ACCEPTED, nil
}

func (self *RawHealthStorage) GetLatestReport(subject dt.EntityId) *dt.Report {
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
	var max_ts time.Time
	var recent_ob *dt.Observation
	var who dt.EntityId
	first := true
	for observer, view := range panorama.Views {
		e := view.Observations.Back()
		val := e.Value.(*dt.Observation)
		if first || max_ts.Before(val.Ts) {
			first = false
			max_ts = val.Ts
			recent_ob = val
			who = observer
		}
	}
	return &dt.Report{
		Observer:    who,
		Subject:     subject,
		Observation: *recent_ob,
	}
}

func (self *RawHealthStorage) Dump() {
	for subject, panorama := range self.Tenants {
		fmt.Printf("=============%s=============\n", subject)
		for observer, view := range panorama.Views {
			fmt.Printf("%d observations for %s->%s\n", view.Observations.Len(), observer, subject)
			for e := view.Observations.Front(); e != nil; e = e.Next() {
				val := e.Value.(*dt.Observation)
				fmt.Printf("|%s| %s %s\n", observer, val.Ts.Format(time.UnixDate), val.Metrics)
			}
		}
	}
}
