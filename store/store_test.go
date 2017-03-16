package store

import (
	. "deephealth/health"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAddReport(t *testing.T) {
	store := NewHViewStore("TS_1", "TS_2", "TS_3", "TS_4")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		t.Logf("Making observation %d", i)
		observation := NewHObservation(time.Now(), "cpu", "disk", "network", "memory")
		observation.SetMetric("cpu", HEALTHY, 100)
		observation.SetMetric("disk", HEALTHY, 90)
		observation.SetMetric("network", UNHEALTHY, 10)
		observation.SetMetric("memory", MAYBE_UNHEALTHY, 30)
		observer := EntityId(fmt.Sprintf("FE_%d", i))
		subject := EntityId(fmt.Sprintf("TS_%d", i%3))
		report := &HReport{Observer: observer, Subject: subject, Observation: *observation}
		wg.Add(1)
		go func() {
			var reply int
			store.AddReport(report, &reply)
			wg.Done()
		}()
	}
	wg.Wait()

	if len(store.Tables) == 0 {
		t.Error("Health table should not be empty")
	}
	for subject, table := range store.Tables {
		t.Logf("=============%s=============", subject)
		for observer, view := range table.Views {
			t.Logf("%d observations for %s->%s", view.Observations.Len(), observer, subject)
			for e := view.Observations.Front(); e != nil; e = e.Next() {
				val := e.Value.(*HObservation)
				t.Logf("|%s| %s %s...%s", observer, val.Ts.Format(time.UnixDate), *val.Vector[0], *val.Vector[len(val.Vector)-1])
			}
		}
	}
}
