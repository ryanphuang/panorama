package store

import (
	"testing"

	"deephealth/decision"
	dt "deephealth/types"
)

func TestInfer(t *testing.T) {
	raw := NewRawHealthStorage("TS_1", "TS_2")
	var majority decision.SimpleMajorityInference
	infs := NewHealthInferenceStorage(raw, majority)
	metrics := map[string]dt.Value{"cpu": dt.Value{dt.HEALTHY, 100}}
	report := dt.NewReport("FE_2", "TS_3", metrics)
	result, err := raw.AddReport(report, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", report)
	}
	metrics = map[string]dt.Value{"mem": dt.Value{dt.UNHEALTHY, 30}, "cpu": dt.Value{dt.UNHEALTHY, 60}}
	report = dt.NewReport("FE_1", "TS_3", metrics)
	result, err = raw.AddReport(report, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", report)
	}
	metrics = map[string]dt.Value{"cpu": dt.Value{dt.HEALTHY, 70}}
	report = dt.NewReport("FE_2", "TS_3", metrics)
	result, err = raw.AddReport(report, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", report)
	}
	metrics = map[string]dt.Value{"mem": dt.Value{dt.HEALTHY, 60}, "network": dt.Value{dt.HEALTHY, 70}, "cpu": dt.Value{dt.HEALTHY, 80}}
	report = dt.NewReport("FE_4", "TS_3", metrics)
	result, err = raw.AddReport(report, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", report)
	}
	_, err = infs.Infer(report)
	if err != nil {
		t.Errorf("Fail to infer reports")
	}
	inference := infs.GetInference(report.Subject)
	if inference == nil {
		t.Fatalf("No inference found")
	}
	if inference.Subject != report.Subject {
		t.Fatalf("Get wrong inference")
	}
	if len(inference.Observers) != 3 {
		t.Fatalf("Should have 3 observers at this moment")
	}
	metric, ok := inference.Observation.Metrics["cpu"]
	if !ok {
		t.Fatalf("Missing metric in inference")
	}
	if metric.Status != dt.HEALTHY {
		t.Fatalf("Should infer cpu HEALTHY")
	}
	infs.Stop()
}
