package store

import "testing"

func TestMakeHVector(t *testing.T) {
	var v *HVector
	v = NewHVector("cpu", "disk", "network", "memory")
	if v == nil {
		t.Error("Fail to make health vector")
	}
	t.Log(v)
}
