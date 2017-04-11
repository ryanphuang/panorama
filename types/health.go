package types

import (
	"bytes"
	"fmt"
	"time"

	pb "deephealth/build/gen"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func SetMetric(observation *pb.Observation, name string, status pb.Status, score float32) bool {
	metric, ok := observation.Metrics[name]
	if !ok {
		return false
	}
	metric.Value.Status = status
	metric.Value.Score = score
	return true
}

func GetMetric(observation *pb.Observation, name string) *pb.Metric {
	metric, ok := observation.Metrics[name]
	if !ok {
		return nil
	}
	return metric
}

func AddMetric(observation *pb.Observation, name string, status pb.Status, score float32) *pb.Observation {
	metric, ok := observation.Metrics[name]
	if !ok {
		observation.Metrics[name] = &pb.Metric{name, &pb.Value{status, score}}
	} else {
		metric.Value.Status = status
		metric.Value.Score = score
	}
	return observation
}

func NewObservationSingleMetric(t time.Time, name string, status pb.Status, score float32) *pb.Observation {
	metrics := make(map[string]*pb.Metric)
	metrics[name] = &pb.Metric{name, &pb.Value{status, score}}
	if pts, err := ptypes.TimestampProto(t); err == nil {
		return &pb.Observation{pts, metrics}
	}
	return nil
}

func NewObservation(t time.Time, names ...string) *pb.Observation {
	metrics := make(map[string]*pb.Metric)
	for _, name := range names {
		metrics[name] = &pb.Metric{name, &pb.Value{pb.Status_INVALID, 0.0}}
	}
	if pts, err := ptypes.TimestampProto(t); err == nil {
		return &pb.Observation{pts, metrics}
	}
	return nil
}

func NewReport(observer string, subject string, metrics map[string]*pb.Value) *pb.Report {
	o := NewObservation(time.Now())
	for k, v := range metrics {
		o.Metrics[k] = &pb.Metric{k, v}
	}
	return &pb.Report{
		Observer:    observer,
		Subject:     subject,
		Observation: o,
	}
}

func CompareTimestamp(a *timestamp.Timestamp, b *timestamp.Timestamp) int32 {
	if a.Seconds < b.Seconds {
		return -1
	} else if a.Seconds > b.Seconds {
		return 1
	}
	return a.Nanos - b.Nanos
}

func ObservationString(ob *pb.Observation) string {
	var buf bytes.Buffer
	buf.WriteString(ptypes.TimestampString(ob.Ts) + " { ")
	for name, metric := range ob.Metrics {
		buf.WriteString(fmt.Sprintf("%s: %s, %.1f; ", name, metric.Value.Status.String(), metric.Value.Score))
	}
	buf.WriteString("}")
	return buf.String()
}
