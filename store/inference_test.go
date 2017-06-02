package store

import (
	"testing"

	pb "deephealth/build/gen"
	"deephealth/decision"
	dt "deephealth/types"
)

type metrics_t map[string]*pb.Value
type report_t struct {
	observer string
	subject  string
	metrics  metrics_t
}

func TestInfer(t *testing.T) {
	raw := NewRawHealthStorage()
	var majority decision.SimpleMajorityInference
	infs := NewHealthInferenceStorage(raw, majority)
	subject := "TS_3"
	reports := []report_t{
		{
			"FE_2",
			subject,
			metrics_t{
				"cpu": &pb.Value{pb.Status_HEALTHY, 100},
			},
		},
		{
			"FE_1",
			subject,
			metrics_t{
				"mem": &pb.Value{pb.Status_UNHEALTHY, 30},
				"cpu": &pb.Value{pb.Status_UNHEALTHY, 60},
			},
		},
		{
			"FE_2",
			subject,
			metrics_t{
				"cpu": &pb.Value{pb.Status_HEALTHY, 70},
			},
		},
		{
			"FE_4",
			subject,
			metrics_t{
				"mem":     &pb.Value{pb.Status_HEALTHY, 60},
				"network": &pb.Value{pb.Status_HEALTHY, 70},
				"cpu":     &pb.Value{pb.Status_HEALTHY, 80},
			},
		},
		{
			"FE_2",
			subject,
			metrics_t{
				"cpu": &pb.Value{pb.Status_HEALTHY, 70},
			},
		},
		{
			"FE_4",
			subject,
			metrics_t{
				"network": &pb.Value{pb.Status_HEALTHY, 60},
				"cpu":     &pb.Value{pb.Status_UNHEALTHY, 20},
			},
		},
		{
			"FE_5",
			subject,
			metrics_t{
				"snapshot": &pb.Value{pb.Status_DEAD, 0},
			},
		},
	}
	var err error

	for _, report := range reports {
		r := dt.NewReport(report.observer, report.subject, report.metrics)
		result, err := raw.AddReport(r, false)
		if err != nil || result != REPORT_ACCEPTED {
			t.Fatalf("Fail to add report %s", r)
		}
	}
	_, err = infs.InferSubject(subject)
	if err != nil {
		t.Errorf("Fail to infer reports")
	}
	inference := infs.GetInference(subject)
	if inference == nil {
		t.Fatalf("No inference found")
	}
	if inference.Subject != subject {
		t.Fatalf("Get wrong inference")
	}
	if len(inference.Observers) != 4 {
		t.Fatalf("Should have 4 observers at this moment")
	}
	metric, ok := inference.Observation.Metrics["cpu"]
	if !ok {
		t.Fatalf("Missing metric in inference")
	}
	if metric.Value.Status != pb.Status_UNHEALTHY {
		t.Fatalf("Should infer cpu UNHEALTHY")
	}
	metric, ok = inference.Observation.Metrics["mem"]
	if !ok {
		t.Fatalf("Missing metric in inference")
	}
	if metric.Value.Status != pb.Status_UNHEALTHY {
		t.Fatalf("Should infer mem UNHEALTHY")
	}

	metrics := metrics_t{"sync": &pb.Value{pb.Status_HEALTHY, 80}}
	r := dt.NewReport("FE_2", subject, metrics)
	result, err := raw.AddReport(r, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", r)
	}
	_, err = infs.InferReport(r)
	inference = infs.GetInference(subject)
	if len(inference.Observation.Metrics) != 5 {
		t.Fatalf("Should have 5 observers at this moment, got %d", len(inference.Observation.Metrics))
	}
	infs.Stop()
}
