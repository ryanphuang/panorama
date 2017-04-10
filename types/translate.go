package types

import (
	"time"

	pb "deephealth/build/gen"
	"github.com/golang/protobuf/ptypes"
)

func StatusFromStr(status string) Status {
	switch status {
	case "n":
		return NA
	case "u":
		return UNHEALTHY
	case "h":
		return HEALTHY
	case "m":
		return MAYBE_UNHEALTHY
	case "d":
		return DYING
	case "dd":
		return DEAD
	default:
		return INVALID
	}
}

func StatusFromPb(in pb.Status) Status {
	switch in {
	case pb.Status_NA:
		return NA
	case pb.Status_UNHEALTHY:
		return UNHEALTHY
	case pb.Status_HEALTHY:
		return HEALTHY
	case pb.Status_MAYBE_UNHEALTHY:
		return MAYBE_UNHEALTHY
	case pb.Status_DYING:
		return DYING
	case pb.Status_DEAD:
		return DEAD
	default:
		return INVALID
	}
}

func StatusToPb(in Status) pb.Status {
	switch in {
	case NA:
		return pb.Status_NA
	case UNHEALTHY:
		return pb.Status_UNHEALTHY
	case HEALTHY:
		return pb.Status_HEALTHY
	case MAYBE_UNHEALTHY:
		return pb.Status_MAYBE_UNHEALTHY
	case DYING:
		return pb.Status_DYING
	case DEAD:
		return pb.Status_DEAD
	default:
		return pb.Status_INVALID
	}
}

func MetricToPb(in *Metric) *pb.Metric {
	status := StatusToPb(in.Status)
	return &pb.Metric{
		Name:   in.Name,
		Status: status,
		Score:  in.Score,
	}
}

func ObservationToPb(in *Observation) *pb.Observation {
	ts, err := ptypes.TimestampProto(in.Ts)
	if err != nil {
		return nil
	}
	metrics := make(map[string]*pb.Metric)
	for k, v := range in.Metrics {
		metrics[k] = MetricToPb(v)
	}
	return &pb.Observation{
		Ts:      ts,
		Metrics: metrics,
	}
}

func ReportToPb(in *Report) *pb.Report {
	if in == nil {
		return nil
	}
	observation := ObservationToPb(in.Observation)
	if observation == nil {
		return nil
	}
	return &pb.Report{
		Observer:    string(in.Observer),
		Subject:     string(in.Subject),
		Observation: observation,
	}
}

func MetricFromPb(in *pb.Metric) *Metric {
	status := StatusFromPb(in.Status)
	return &Metric{in.Name, Value{status, in.Score}}
}

func ObservationFromPb(in *pb.Observation) *Observation {
	ts, err := ptypes.Timestamp(in.Ts)
	if err != nil {
		return nil
	}
	metrics := make(Metrics)
	for k, v := range in.Metrics {
		metrics[k] = MetricFromPb(v)
	}
	return &Observation{
		Ts:      ts,
		Metrics: metrics,
	}
}

func ReportFromPb(in *pb.Report) *Report {
	observation := ObservationFromPb(in.Observation)
	if observation == nil {
		return nil
	}
	return &Report{
		Observer:    EntityId(in.Observer),
		Subject:     EntityId(in.Subject),
		Observation: observation,
	}
}

func NewPbObservationSingleMetric(t time.Time, name string, status pb.Status, score float32) *pb.Observation {
	ts, err := ptypes.TimestampProto(t)
	if err != nil {
		return nil
	}
	metrics := make(map[string]*pb.Metric)
	metrics[name] = &pb.Metric{name, status, score}
	return &pb.Observation{
		Ts:      ts,
		Metrics: metrics,
	}
}
