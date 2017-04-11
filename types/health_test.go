package types

import (
	"testing"
	"time"

	pb "deephealth/build/gen"
)

func TestNewObservation(t *testing.T) {
	var v *pb.Observation
	v = NewObservation(time.Now(), "cpu", "disk", "network")
	if v == nil {
		t.Fatal("Fail to make health vector")
	}
	t.Log(v)
	SetMetric(v, "cpu", pb.Status_UNHEALTHY, 30)
	m := GetMetric(v, "cpu")
	if m.Value.Score != 30 {
		t.Error("Fail to set CPU metric")
	}
	t.Log(m)
}
