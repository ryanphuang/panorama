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

func TestInferPending(t *testing.T) {
	raw := NewRawHealthStorage()
	var majority decision.SimpleMajorityInference
	infs := NewHealthInferenceStorage(raw, majority)
	subject := "TS_3"
	observer := "FE_2"

	metrics1 := metrics_t{
		"request.100": &pb.Value{pb.Status_PENDING, 40},
		"request.103": &pb.Value{pb.Status_HEALTHY, 60},
		"request.105": &pb.Value{pb.Status_HEALTHY, 80},
		"request.106": &pb.Value{pb.Status_PENDING, 40},
	}

	metrics2 := metrics_t{
		"request.105": &pb.Value{pb.Status_PENDING, 40},
		"request.103": &pb.Value{pb.Status_PENDING, 30},
	}

	metrics3 := metrics_t{
		"request.105": &pb.Value{pb.Status_PENDING, 20},
		"request.103": &pb.Value{pb.Status_PENDING, 40},
	}

	metrics4 := metrics_t{
		"request.105": &pb.Value{pb.Status_PENDING, 30},
		"request.103": &pb.Value{pb.Status_HEALTHY, 80},
	}

	var err error

	r := dt.NewReport(observer, subject, metrics1)
	result, err := raw.AddReport(r, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", r)
	}
	inference, err := infs.InferSubject(subject)
	if err != nil {
		t.Errorf("Fail to infer reports")
	}
	if inference.Subject != subject {
		t.Fatalf("Get wrong inference")
	}
	metric, ok := inference.Observation.Metrics["request.100"]
	if !ok {
		t.Fatalf("Missing metric in inference")
	}
	if metric.Value.Status != pb.Status_PENDING {
		t.Fatalf("Should infer request.100 PENDING")
	}
	if metric.Value.Score != 40 {
		t.Fatalf("request.100 health score should be 40")
	}
	r = dt.NewReport(observer, subject, metrics2)
	raw.AddReport(r, false)
	r = dt.NewReport(observer, subject, metrics3)
	raw.AddReport(r, false)
	r = dt.NewReport(observer, subject, metrics4)
	result, err = raw.AddReport(r, false)
	if err != nil || result != REPORT_ACCEPTED {
		t.Fatalf("Fail to add report %s", r)
	}
	inference, err = infs.InferSubject(subject)
	metric, _ = inference.Observation.Metrics["request.103"]
	if metric.Value.Status != pb.Status_HEALTHY {
		t.Fatalf("Should infer request.103 HEALTHY")
	}
	if metric.Value.Score != 70 {
		t.Fatalf("request.103 health score should be 70")
	}
	metric, _ = inference.Observation.Metrics["request.105"]
	if metric.Value.Status != pb.Status_PENDING {
		t.Fatalf("Should infer request.105 PENDING")
	}
	if metric.Value.Score != 25 {
		t.Fatalf("request.105 health score should be 30, got %f", metric.Value.Score)
	}
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
