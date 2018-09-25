package store

import (
	"fmt"
	"os"
	"sync"
	"time"

	pb "panorama/build/gen"
	dt "panorama/types"
	du "panorama/util"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	MaxReportPerView = 10 // maximum number of reports to store for a given view
	stag             = "store"
)

const (
	REPORT_IGNORED int = iota
	REPORT_ACCEPTED
	REPORT_FAILED
)

type RawHealthStorage struct {
	Tenants   map[string]*dt.ConcurrentPanorama
	Watchlist map[string]time.Time

	db dt.HealthDB
	mu *sync.RWMutex
}

func NewRawHealthStorage(subjects ...string) *RawHealthStorage {
	store := &RawHealthStorage{
		Tenants:   make(map[string]*dt.ConcurrentPanorama),
		Watchlist: make(map[string]time.Time),

		mu: &sync.RWMutex{},
	}
	now := time.Now()
	for _, subject := range subjects {
		store.Watchlist[subject] = now
	}
	return store
}

var _ dt.HealthStorage = new(RawHealthStorage)

func (self *RawHealthStorage) SetDB(db dt.HealthDB) {
	self.db = db
}

func (self *RawHealthStorage) AddSubject(subject string) bool {
	self.mu.Lock()
	_, ok := self.Watchlist[subject]
	if !ok {
		self.Watchlist[subject] = time.Now()
	}
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
	}
	return ok
}

func (self *RawHealthStorage) GetSubjects() map[string]time.Time {
	return self.Watchlist
}

func (self *RawHealthStorage) AddReport(report *pb.Report, filter bool) (int, error) {
	self.mu.Lock()
	_, ok := self.Watchlist[report.Subject]
	if !ok {
		if filter {
			// subject is not in our watch list, ignore the report
			du.LogI(stag, "%s not in watch list, ignore report...", report.Subject)
			self.mu.Unlock()
			return REPORT_IGNORED, nil
		} else {
			// no filtering, add subject to watch list
			self.Watchlist[report.Subject] = time.Now()
		}
	}
	du.LogD(stag, "add report for %s from %s...", report.Subject, report.Observer)
	pano, ok := self.Tenants[report.Subject]
	if !ok {
		pano = &dt.ConcurrentPanorama{
			Value: &pb.Panorama{
				Subject: report.Subject,
				Views:   make(map[string]*pb.View),
			},
		}
		self.Tenants[report.Subject] = pano
	}
	self.mu.Unlock()
	pano.Lock()
	defer pano.Unlock()
	view, ok := pano.Value.Views[report.Observer]
	if !ok {
		view = &pb.View{
			Observer:     report.Observer,
			Subject:      report.Subject,
			Observations: make([]*pb.Observation, 0, MaxReportPerView+1),
		}
		pano.Value.Views[report.Observer] = view
		du.LogD(stag, "create view for %s->%s...", report.Observer, report.Subject)
	}
	view.Observations = append(view.Observations, report.Observation)
	// du.LogD(stag, "add observation to view %s->%s: %s", report.Observer, report.Subject, dt.ObservationString(report.Observation))
	// du.PrintMemUsage(os.Stdout)
	if len(view.Observations) > MaxReportPerView {
		du.LogD(stag, "truncating list")
		view.Observations = view.Observations[1:]
	}
	if self.db != nil {
		go self.db.InsertReport(report)
	}
	return REPORT_ACCEPTED, nil
}

func (self *RawHealthStorage) GetPanorama(subject string) *dt.ConcurrentPanorama {
	self.mu.RLock()
	pano, _ := self.Tenants[subject]
	self.mu.RUnlock()
	return pano
}

func (self *RawHealthStorage) GetView(observer string, subject string) *pb.View {
	self.mu.RLock()
	pano, ok := self.Tenants[subject]
	self.mu.RUnlock()
	if ok {
		pano.RLock()
		view, _ := pano.Value.Views[observer]
		pano.RUnlock()
		return view
	}
	return nil
}

func (self *RawHealthStorage) GetLatestReport(subject string) *pb.Report {
	self.mu.RLock()
	pano, ok := self.Tenants[subject]
	self.mu.RUnlock()
	if !ok {
		return nil
	}
	pano.RLock()
	defer pano.RUnlock()
	var max_ts *timestamp.Timestamp
	var recent_ob *pb.Observation
	var who string
	first := true
	for observer, view := range pano.Value.Views {
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

func (self *RawHealthStorage) GC(ttl time.Duration, relative bool) map[string]uint32 {
	var expire *timestamp.Timestamp
	var err error
	if !relative {
		expire, err = ptypes.TimestampProto(time.Now().Add(-ttl))
		if err != nil {
			du.LogE(stag, "Fail to convert expire timestamp: %s", err)
			return nil
		}
	}
	self.mu.RLock()
	defer self.mu.RUnlock()
	retired := make(map[string]uint32)

	remains := make([]int, MaxReportPerView+1)
	ttl_nanos := ttl.Nanoseconds()
	for subject, pano := range self.Tenants {
		self.mu.RUnlock()
		pano.Lock()
		r1 := 0
		for observer, view := range pano.Value.Views {
			lo := len(view.Observations)
			if lo == 0 {
				continue
			}
			ri := 0
			if relative {
				max_ts := view.Observations[lo-1].Ts
				du.LogD(stag, "most recent report %s=>%s is at: %s", observer, subject, ptypes.TimestampString(max_ts))
				for i := 0; i < lo-1; i++ {
					val := view.Observations[i]
					elapsed := dt.SubtractTimestamp(max_ts, val.Ts)
					du.LogD(stag, "[%d] at %s, elapsed %.1f seconds", i, ptypes.TimestampString(val.Ts), float64(elapsed)/1000000000.0)
					if elapsed < ttl_nanos {
						remains[ri] = i
						ri++
					}
				}
				remains[ri] = lo - 1
				ri++
			} else {
				for i, val := range view.Observations {
					if dt.CompareTimestamp(val.Ts, expire) > 0 {
						remains[ri] = i
						ri++
					}
				}
			}
			if ri < lo {
				obs := make([]*pb.Observation, ri, MaxReportPerView+1)
				for i := 0; i < ri; i++ {
					obs[i] = view.Observations[remains[i]]
				}
				r1 += lo - ri
				view.Observations = obs
			}
		}
		if r1 > 0 {
			retired[subject] = uint32(r1)
		}
		pano.Unlock()
		self.mu.RLock()
	}
	return retired
}

func (self *RawHealthStorage) DumpPanorama() map[string]*pb.Panorama {
	snapshot := make(map[string]*pb.Panorama)
	self.mu.RLock()
	defer self.mu.RUnlock()
	for subject, pano := range self.Tenants {
		snapshot[subject] = pano.Value
	}
	return snapshot
}

func (self *RawHealthStorage) Dump() {
	self.mu.RLock()
	for subject, pano := range self.Tenants {
		fmt.Printf("=============%s=============\n", subject)
		dt.DumpPanorama(os.Stdout, pano.Value)
	}
	defer self.mu.RUnlock()
}
