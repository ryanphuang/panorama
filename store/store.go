package store

import "sync"
import "container/list"

const (
	MaxReportPerView = 5 // maximum number of reports to store for a given view
)

type HRawViewStore struct {
	tables    map[EntityId]*HTable
	mu        *sync.Mutex
	locks     map[EntityId]*sync.Mutex
	watchlist map[EntityId]bool
}

func NewHViewStore(subjects ...EntityId) *HRawViewStore {
	store := &HRawViewStore{
		tables:    make(map[EntityId]*HTable),
		mu:        &sync.Mutex{},
		locks:     make(map[EntityId]*sync.Mutex),
		watchlist: make(map[EntityId]bool),
	}
	var table *HTable
	for _, subject := range subjects {
		store.watchlist[subject] = true
		store.locks[subject] = new(sync.Mutex)
		table = new(HTable)
		table.subject = subject
		table.views = make(map[EntityId]*HView)
		store.tables[subject] = table
	}
}

func (self *HRawViewStore) AddWatchSubject(subject EntityId) bool {
	_, ok := watchlist.set(subject)
	return ok
}

func (self *HRawViewStore) AddReport(report *HReport) (int, error) {
	_, ok := self.watchlist[report.subject]
	if !ok {
		// subject is not in our watch list, ignore the report
		return 1, nil
	}
	self.mu.Lock()
	l, ok := self.locks[report.subject]
	if !ok {
		l = new(sync.Mutex)
		self.locks[report.subject] = l
	}
	self.mu.Unlock()
	l.Lock()
	table, ok := self.tables[report.subject]
	if !ok {
		table = &HTable{
			subject: subject,
			views:   make(map[EntityId]*HView),
		}
		self.tables[report.subject] = table
	}
	view, ok := table.views[report.observer]
	if !view {
		view = &HView{
			observer:     report.observer,
			subject:      report.subject,
			observations: list.New(),
		}
	}
	view.observations.PushBack(&report.observation)
	if view.observations.Len() > MaxReportPerView {
		view.observations.Remove(view.observations.Front())
	}
	l.Unlock()
}
