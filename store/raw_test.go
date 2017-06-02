package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	pb "deephealth/build/gen"

	dt "deephealth/types"
	du "deephealth/util"
)

func TestAddSubject(t *testing.T) {
	du.SetLogLevel(du.InfoLevel)
	store := NewRawHealthStorage("TS_1", "TS_2")
	metrics := map[string]*pb.Value{"cpu": &pb.Value{pb.Status_HEALTHY, 100}}
	report := dt.NewReport("FE_2", "TS_3", metrics)
	result, err := store.AddReport(report, true)
	if err != nil {
		t.Errorf("Fail to add report %s", report)
	}
	if result != REPORT_IGNORED {
		t.Errorf("Report %s should get ignored", report)
	}
	store.AddSubject("TS_3")
	result, err = store.AddReport(report, true)
	if err != nil {
		t.Errorf("Fail to add report %s", report)
	}
	if result != REPORT_ACCEPTED {
		t.Errorf("Report %s should get accepted", report)
	}
}

func TestAddReport(t *testing.T) {
	du.SetLogLevel(du.InfoLevel)
	subjects := []string{"TS_1", "TS_2", "TS_3", "TS_4"}
	smap := make(map[string]bool)
	for _, s := range subjects {
		smap[s] = true
	}

	store := NewRawHealthStorage(subjects...)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		t.Logf("Making observation %d", i)
		metrics := map[string]*pb.Value{
			"cpu":     &pb.Value{pb.Status_HEALTHY, 100},
			"disk":    &pb.Value{pb.Status_HEALTHY, 90},
			"network": &pb.Value{pb.Status_UNHEALTHY, 10},
			"memory":  &pb.Value{pb.Status_MAYBE_UNHEALTHY, 30},
		}
		observer := fmt.Sprintf("FE_%d", i)
		subject := fmt.Sprintf("TS_%d", i%3)
		report := dt.NewReport(observer, subject, metrics)
		wg.Add(1)
		go func() {
			result, err := store.AddReport(report, true)
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
		for observer, view := range stereo.Value.Views {
			t.Logf("%d observations for %s->%s", len(view.Observations), observer, subject)
			for _, ob := range view.Observations {
				t.Logf("|%s| %s\n", observer, dt.ObservationString(ob))
			}
		}
	}
}

func TestRecentReport(t *testing.T) {
	du.SetLogLevel(du.InfoLevel)
	store := NewRawHealthStorage("TS_1", "TS_2")

	metrics := map[string]*pb.Value{"cpu": &pb.Value{pb.Status_HEALTHY, 100}}
	report := dt.NewReport("FE_2", "TS_1", metrics)
	store.AddReport(report, true)
	metrics = map[string]*pb.Value{"cpu": &pb.Value{pb.Status_HEALTHY, 90}}
	report = dt.NewReport("FE_2", "TS_1", metrics)
	store.AddReport(report, true)
	metrics = map[string]*pb.Value{"cpu": &pb.Value{pb.Status_HEALTHY, 70}}
	report = dt.NewReport("FE_2", "TS_1", metrics)
	store.AddReport(report, true)
	metrics = map[string]*pb.Value{"cpu": &pb.Value{pb.Status_UNHEALTHY, 30}}
	report = dt.NewReport("FE_2", "TS_1", metrics)
	store.AddReport(report, true)

	ret := store.GetLatestReport("TS_1")
	if ret.Observer != "FE_2" {
		t.Errorf("Wrong subject in the latest report: %s\n", *ret)
	}
	metric, ok := ret.Observation.Metrics["cpu"]
	if !ok {
		t.Error("The latest report have a CPU metric")
	}
	if metric.Value.Status != pb.Status_UNHEALTHY || metric.Value.Score != 30 {
		t.Errorf("Wrong metric in the latest report: %s\n", metric)
	}

	time.Sleep(200 * time.Millisecond)
	metrics = map[string]*pb.Value{"memory": &pb.Value{pb.Status_UNHEALTHY, 20}}
	report = dt.NewReport("FE_4", "TS_1", metrics)
	store.AddReport(report, true)
	ret = store.GetLatestReport("TS_1")
	if ret.Observer != "FE_4" {
		t.Errorf("Wrong subject in the latest report: %s\n", *ret)
	}
	metric, ok = ret.Observation.Metrics["memory"]
	if !ok {
		t.Error("The latest report have a memory metric")
	}
	if metric.Value.Status != pb.Status_UNHEALTHY || metric.Value.Score != 20 {
		t.Errorf("Wrong metric in the latest report: %s\n", metric)
	}

	time.Sleep(200 * time.Millisecond)
	metrics = map[string]*pb.Value{"network": &pb.Value{pb.Status_HEALTHY, 80}}
	report = dt.NewReport("FE_5", "TS_1", metrics)
	store.AddReport(report, true)
	metrics = map[string]*pb.Value{"memory": &pb.Value{pb.Status_HEALTHY, 70}}
	report = dt.NewReport("FE_1", "TS_1", metrics)
	store.AddReport(report, true)
	ret = store.GetLatestReport("TS_1")
	if ret.Observer != "FE_1" {
		t.Errorf("Wrong subject in the latest report: %s\n", *ret)
	}
	metric, ok = ret.Observation.Metrics["memory"]
	if !ok {
		t.Error("The latest report have a memory metric")
	}
	if metric.Value.Status != pb.Status_HEALTHY || metric.Value.Score != 70 {
		t.Errorf("Wrong metric in the latest report: %s\n", metric)
	}
}

func TestTruncate(t *testing.T) {
	du.SetLogLevel(du.InfoLevel)
	store := NewRawHealthStorage("TS_1", "TS_2")

	for i := 0; i < 20; i++ {
		metrics := map[string]*pb.Value{"cpu": &pb.Value{pb.Status_UNHEALTHY, float32(i)}}
		report := dt.NewReport("FE_2", "TS_1", metrics)
		store.AddReport(report, false)
	}
	ret := store.GetLatestReport("TS_1")
	metric, ok := ret.Observation.Metrics["cpu"]
	if !ok {
		t.Error("The latest report have a cpu metric")
	}
	if metric.Value.Status != pb.Status_UNHEALTHY || metric.Value.Score != 19 {
		t.Errorf("Wrong metric in the latest report: %s\n", metric)
	}
	pano := store.GetPanorama("TS_1")
	for observer, view := range pano.Value.Views {
		if observer != "FE_2" {
			t.Errorf("Only expecting observations by FE_2, got %s", observer)
		}
		if len(view.Observations) != MaxReportPerView {
			t.Errorf("Expecting observations to be truncated to %d", MaxReportPerView)
		}
		for i, ob := range view.Observations {
			metric, _ := ob.Metrics["cpu"]
			expected := 20 - MaxReportPerView + i
			if metric.Value.Status != pb.Status_UNHEALTHY || metric.Value.Score != float32(expected) {
				t.Errorf("Expecting metric score of %d, got %f", expected, metric.Value.Score)
			}
		}
	}
}

func addReports(store *RawHealthStorage, start int, end int, t *testing.T) {
	for i := start; i < end; i++ {
		t.Logf("Making observation %d", i)
		metrics := map[string]*pb.Value{
			"cpu":     &pb.Value{pb.Status_HEALTHY, 100},
			"disk":    &pb.Value{pb.Status_HEALTHY, 90},
			"network": &pb.Value{pb.Status_UNHEALTHY, 10},
			"memory":  &pb.Value{pb.Status_MAYBE_UNHEALTHY, 30},
		}
		observer := fmt.Sprintf("FE_1")
		subject := fmt.Sprintf("TS_2")
		report := dt.NewReport(observer, subject, metrics)
		_, err := store.AddReport(report, false)
		if err != nil {
			t.Errorf("Fail to add report %s", report)
		}
	}
}

func TestGC(t *testing.T) {
	store := NewRawHealthStorage()
	addReports(store, 0, 5, t)
	t.Logf("Sleep 5 seconds before submitting new reports")
	time.Sleep(5 * time.Second)
	addReports(store, 5, 8, t)
	retired := store.GC(3*time.Second, true)
	r, ok := retired["TS_2"]
	if !ok || r != 5 {
		t.Errorf("Should retire 5 observations for TS_2")
	}
	pano := store.GetPanorama("TS_2")
	t.Logf("New panorama: %s", dt.PanoramaString(pano.Value))
	time.Sleep(3 * time.Second)
	retired = store.GC(2*time.Second, false)
	r, ok = retired["TS_2"]
	if !ok || r != 3 {
		t.Errorf("Should retire 3 observations for TS_2")
	}
}
