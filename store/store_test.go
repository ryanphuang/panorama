package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	dh "deephealth"
	dt "deephealth/types"
)

func TestAddSubject(t *testing.T) {
	dh.SetLogLevel(dh.InfoLevel)
	subjects := []dt.EntityId{"TS_1", "TS_2"}
	store := NewRawHealthStorage(subjects...)
	metrics := map[string]dt.Value{"cpu": dt.Value{dt.HEALTHY, 100}}
	report := dt.NewReport("FE_2", "TS_3", metrics)
	result, err := store.AddReport(report)
	if err != nil {
		t.Errorf("Fail to add report %s", report)
	}
	if result != REPORT_IGNORED {
		t.Errorf("Report %s should get ignored", report)
	}
	store.AddSubject("TS_3")
	result, err = store.AddReport(report)
	if err != nil {
		t.Errorf("Fail to add report %s", report)
	}
	if result != REPORT_ACCEPTED {
		t.Errorf("Report %s should get accepted", report)
	}
}

func TestAddReport(t *testing.T) {
	dh.SetLogLevel(dh.InfoLevel)
	subjects := []dt.EntityId{"TS_1", "TS_2", "TS_3", "TS_4"}
	smap := make(map[dt.EntityId]bool)
	for _, s := range subjects {
		smap[s] = true
	}

	store := NewRawHealthStorage(subjects...)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		t.Logf("Making observation %d", i)
		metrics := map[string]dt.Value{
			"cpu":     dt.Value{dt.HEALTHY, 100},
			"disk":    dt.Value{dt.HEALTHY, 90},
			"network": dt.Value{dt.UNHEALTHY, 10},
			"memory":  dt.Value{dt.MAYBE_UNHEALTHY, 30},
		}
		observer := dt.EntityId(fmt.Sprintf("FE_%d", i))
		subject := dt.EntityId(fmt.Sprintf("TS_%d", i%3))
		report := dt.NewReport(observer, subject, metrics)
		wg.Add(1)
		go func() {
			result, err := store.AddReport(report)
			if err != nil {
				t.Errorf("Fail to add report %s", report)
			}
			_, watched := smap[subject]
			if watched && result == REPORT_IGNORED {
				t.Errorf("Report %s should not get ignored", report)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if len(store.Tenants) == 0 {
		t.Error("Health table should not be empty")
	}
	for subject, stereo := range store.Tenants {
		t.Logf("=============%s=============", subject)
		for observer, view := range stereo.Views {
			t.Logf("%d observations for %s->%s", view.Observations.Len(), observer, subject)
			for e := view.Observations.Front(); e != nil; e = e.Next() {
				val := e.Value.(*dt.Observation)
				t.Logf("|%s| %s %s", observer, val.Ts.Format(time.UnixDate), val.Metrics)
			}
		}
	}
}
