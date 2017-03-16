package deephealth

import "testing"
import "time"

func TestNewObservation(t *testing.T) {
	var v *Observation
	v = NewObservation(time.Now(), "cpu", "disk", "network")
	if v == nil {
		t.Error("Fail to make health vector")
	}
	t.Log(v)
	v.SetMetric("cpu", UNHEALTHY, 30)
	m := v.GetMetric("cpu")
	if m.Score != 30 {
		t.Error("Fail to set CPU metric")
	}
	t.Log(m)
}
