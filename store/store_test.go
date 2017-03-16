package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	dh "deephealth"
)

func TestAddReport(t *testing.T) {
	store := NewRawHealthStorage("TS_1", "TS_2", "TS_3", "TS_4")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		t.Logf("Making observation %d", i)
		observation := dh.NewObservation(time.Now(), "cpu", "disk", "network", "memory")
		observation.SetMetric("cpu", dh.HEALTHY, 100)
		observation.SetMetric("disk", dh.HEALTHY, 90)
		observation.SetMetric("network", dh.UNHEALTHY, 10)
		observation.SetMetric("memory", dh.MAYBE_UNHEALTHY, 30)
		observer := dh.EntityId(fmt.Sprintf("FE_%d", i))
		subject := dh.EntityId(fmt.Sprintf("TS_%d", i%3))
		report := &dh.Report{Observer: observer, Subject: subject, Observation: *observation}
		wg.Add(1)
		go func() {
			var reply int
			store.AddReport(report, &reply)
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
				val := e.Value.(*dh.Observation)
				t.Logf("|%s| %s %s", observer, val.Ts.Format(time.UnixDate), val.Metrics)
			}
		}
	}
}
