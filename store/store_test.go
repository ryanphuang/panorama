package store

import "fmt"
import "testing"
import "time"
import "sync"

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
		report := &HReport{observer: observer, subject: subject, observation: *observation}
		wg.Add(1)
		go func() {
			store.AddReport(report)
			wg.Done()
		}()
	}
	wg.Wait()

	if len(store.tables) == 0 {
		t.Error("Health table should not be empty")
	}
	for subject, table := range store.tables {
		t.Logf("=============%s=============", subject)
		for observer, view := range table.views {
			t.Logf("%d observations for %s->%s", view.observations.Len(), observer, subject)
			for e := view.observations.Front(); e != nil; e = e.Next() {
				val := e.Value.(*HObservation)
				t.Logf("|%s| %s %s...%s", observer, val.ts.Format(time.UnixDate), *val.vector[0], *val.vector[len(val.vector)-1])
			}
		}
	}
}
