package store

import (
	"container/list"
	"sync"

	dh "deephealth"
)

const (
	MaxReportPerView = 5 // maximum number of reports to store for a given view
	StoreTag         = "store"
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
		dh.LogI(StoreTag, "%s not in watch list, ignore report...", report.Subject)
		*reply = 1
		return nil
	}
	dh.LogD(StoreTag, "add report for %s...", report.Subject)
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
		dh.LogD(StoreTag, "create view for %s->%s...", report.Observer, report.Subject)
	}
	view.Observations.PushBack(&report.Observation)
	if view.Observations.Len() > MaxReportPerView {
		dh.LogD(StoreTag, "truncating list")
		view.Observations.Remove(view.Observations.Front())
	}
	l.Unlock()
	*reply = 0
	return nil
}
