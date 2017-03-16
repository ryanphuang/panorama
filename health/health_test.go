package health

import "testing"
import "time"

func TestNewHObservation(t *testing.T) {
	var v *HObservation
	v = NewHObservation(time.Now(), "cpu", "disk", "network", "memory")
	if v == nil {
		t.Error("Fail to make health vector")
	}
	t.Log(v)
}
