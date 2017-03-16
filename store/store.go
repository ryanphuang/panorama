package store

import (
	"container/list"
	"sync"

	. "deephealth/health"
	"deephealth/util"
)

const (
	MaxReportPerView = 5 // maximum number of reports to store for a given view
	StoreTag         = "store"
)

type HRawViewStore struct {
	Tables    map[EntityId]*HTable
	Locks     map[EntityId]*sync.Mutex
	Watchlist map[EntityId]bool

	mu *sync.Mutex
}

func NewHViewStore(subjects ...EntityId) *HRawViewStore {
	store := &HRawViewStore{
		Tables:    make(map[EntityId]*HTable),
		Locks:     make(map[EntityId]*sync.Mutex),
		Watchlist: make(map[EntityId]bool),

		mu: &sync.Mutex{},
	}
	var table *HTable
	for _, subject := range subjects {
		store.Watchlist[subject] = true
		store.Locks[subject] = new(sync.Mutex)
		table = new(HTable)
		table.Subject = subject
		table.Views = make(map[EntityId]*HView)
		store.Tables[subject] = table
	}
	return store
}

var _ ViewStorage = new(HRawViewStore)

func (self *HRawViewStore) ObserveSubject(subject EntityId, reply *bool) error {
	_, ok := self.Watchlist[subject]
	self.Watchlist[subject] = true
	*reply = !ok
	return nil
}

func (self *HRawViewStore) StopObservingSubject(subject EntityId, reply *bool) error {
	_, ok := self.Watchlist[subject]
	delete(self.Watchlist, subject)
	*reply = ok
	return nil
}

func (self *HRawViewStore) AddReport(report *HReport, reply *int) error {
	_, ok := self.Watchlist[report.Subject]
	if !ok {
		// subject is not in our watch list, ignore the report
		util.LogI(StoreTag, "%s not in watch list, ignore report...", report.Subject)
		*reply = 1
		return nil
	}
	util.LogD(StoreTag, "add report for %s...", report.Subject)
	self.mu.Lock()
	l, ok := self.Locks[report.Subject]
	if !ok {
		l = new(sync.Mutex)
		self.Locks[report.Subject] = l
	}
	self.mu.Unlock()
	l.Lock()
	table, ok := self.Tables[report.Subject]
	if !ok {
		table = &HTable{
			Subject: report.Subject,
			Views:   make(map[EntityId]*HView),
		}
		self.Tables[report.Subject] = table
	}
	view, ok := table.Views[report.Observer]
	if !ok {
		view = &HView{
			Observer:     report.Observer,
			Subject:      report.Subject,
			Observations: list.New(),
		}
		table.Views[report.Observer] = view
		util.LogD(StoreTag, "create view for %s->%s...", report.Observer, report.Subject)
	}
	view.Observations.PushBack(&report.Observation)
	if view.Observations.Len() > MaxReportPerView {
		util.LogD(StoreTag, "truncating list")
		view.Observations.Remove(view.Observations.Front())
	}
	l.Unlock()
	*reply = 0
	return nil
}
