package store

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	dh "deephealth"
)

const (
	MaxReportPerView = 5 // maximum number of reports to store for a given view
	tag              = "store"
)

type RawHealthStorage struct {
	Tenants   map[dh.EntityId]*dh.Panorama
	Locks     map[dh.EntityId]*sync.Mutex
	Watchlist map[dh.EntityId]bool

	mu *sync.Mutex
}

func NewRawHealthStorage(subjects ...dh.EntityId) *RawHealthStorage {
	store := &RawHealthStorage{
		Tenants:   make(map[dh.EntityId]*dh.Panorama),
		Locks:     make(map[dh.EntityId]*sync.Mutex),
		Watchlist: make(map[dh.EntityId]bool),

		mu: &sync.Mutex{},
	}
	var stereo *dh.Panorama
	for _, subject := range subjects {
		store.Watchlist[subject] = true
		store.Locks[subject] = new(sync.Mutex)
		stereo = new(dh.Panorama)
		stereo.Subject = subject
		stereo.Views = make(map[dh.EntityId]*dh.View)
		store.Tenants[subject] = stereo
	}
	return store
}

var _ dh.HealthStorage = new(RawHealthStorage)

func (self *RawHealthStorage) ObserveSubject(subject dh.EntityId, reply *bool) error {
	_, ok := self.Watchlist[subject]
	self.Watchlist[subject] = true
	*reply = !ok
	return nil
}

func (self *RawHealthStorage) StopObservingSubject(subject dh.EntityId, reply *bool) error {
	_, ok := self.Watchlist[subject]
	delete(self.Watchlist, subject)
	*reply = ok
	return nil
}

func (self *RawHealthStorage) AddReport(report *dh.Report, reply *int) error {
	_, ok := self.Watchlist[report.Subject]
	if !ok {
		// subject is not in our watch list, ignore the report
		dh.LogI(tag, "%s not in watch list, ignore report...", report.Subject)
		*reply = 1
		return nil
	}
	dh.LogD(tag, "add report for %s from...", report.Subject, report.Observer)
	self.mu.Lock()
	l, ok := self.Locks[report.Subject]
	if !ok {
		l = new(sync.Mutex)
		self.Locks[report.Subject] = l
	}
	self.mu.Unlock()
	l.Lock()
	stereo, ok := self.Tenants[report.Subject]
	if !ok {
		stereo = &dh.Panorama{
			Subject: report.Subject,
			Views:   make(map[dh.EntityId]*dh.View),
		}
		self.Tenants[report.Subject] = stereo
	}
	view, ok := stereo.Views[report.Observer]
	if !ok {
		view = &dh.View{
			Observer:     report.Observer,
			Subject:      report.Subject,
			Observations: list.New(),
		}
		stereo.Views[report.Observer] = view
		dh.LogD(tag, "create view for %s->%s...", report.Observer, report.Subject)
	}
	view.Observations.PushBack(&report.Observation)
	dh.LogD(tag, "add observation to view %s->%s: %s", report.Observer, report.Subject, report.Observation)
	if view.Observations.Len() > MaxReportPerView {
		dh.LogD(tag, "truncating list")
		view.Observations.Remove(view.Observations.Front())
	}
	l.Unlock()
	*reply = 0
	return nil
}

func (self *RawHealthStorage) Dump() {
	for subject, panorama := range self.Tenants {
		fmt.Printf("=============%s=============\n", subject)
		for observer, view := range panorama.Views {
			fmt.Printf("%d observations for %s->%s\n", view.Observations.Len(), observer, subject)
			for e := view.Observations.Front(); e != nil; e = e.Next() {
				val := e.Value.(*dh.Observation)
				fmt.Printf("|%s| %s %s\n", observer, val.Ts.Format(time.UnixDate), val.Metrics)
			}
		}
	}
}
