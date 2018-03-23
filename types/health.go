package types

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	pb "deephealth/build/gen"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type ConcurrentPanorama struct {
	sync.RWMutex
	Value *pb.Panorama
}

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

func NewMetrics(names ...string) map[string]*pb.Metric {
	metrics := make(map[string]*pb.Metric)
	for _, name := range names {
		metrics[name] = &pb.Metric{name, &pb.Value{pb.Status_INVALID, 0.0}}
	}
	return metrics
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

func SubtractTimestamp(a *timestamp.Timestamp, b *timestamp.Timestamp) int64 {
	return (a.Seconds-b.Seconds)*1000000000 + int64(a.Nanos-b.Nanos)
}

func CompareTimestamp(a *timestamp.Timestamp, b *timestamp.Timestamp) int32 {
	if a.Seconds < b.Seconds {
		return -1
	} else if a.Seconds > b.Seconds {
		return 1
	}
	return a.Nanos - b.Nanos
}

func MetricsString(metrics map[string]*pb.Metric) string {
	var buf bytes.Buffer
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		metric := metrics[key]
		buf.WriteString(fmt.Sprintf("%s: %s, %.1f; ", key, metric.Value.Status.String(), metric.Value.Score))
	}
	return buf.String()
}

func ObservationString(ob *pb.Observation) string {
	if ob.Ts == nil || len(ob.Metrics) == 0 {
		return "{}"
	}
	mStr := MetricsString(ob.Metrics)
	return ptypes.TimestampString(ob.Ts) + " {" + mStr + "}"
}

func DumpPanorama(w io.Writer, pano *pb.Panorama) {
	for observer, view := range pano.Views {
		fmt.Fprintf(w, "[[... %s->%s (%d observations) ...]]\n", observer, pano.Subject, len(view.Observations))
		DumpView(w, view)
	}
}

func DumpView(w io.Writer, view *pb.View) {
	for _, ob := range view.Observations {
		fmt.Fprintf(w, "  |%s| %s\n", view.Observer, ObservationString(ob))
	}
}

func PanoramaString(pano *pb.Panorama) string {
	var buf bytes.Buffer
	for observer, view := range pano.Views {
		buf.WriteString(fmt.Sprintf("\n[[... %s->%s (%d observations) ...]]\n", observer, pano.Subject, len(view.Observations)))
		buf.WriteString(ViewString(view))
	}
	return buf.String()
}

func ViewString(view *pb.View) string {
	var buf bytes.Buffer
	for i, ob := range view.Observations {
		buf.WriteString(fmt.Sprintf("\t|%s| %s", view.Observer, ObservationString(ob)))
		if i != len(view.Observations)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func InferenceString(inf *pb.Inference) string {
	return fmt.Sprintf("%s ==> %s: %s", inf.Observers, inf.Subject, ObservationString(inf.Observation))
}

func StatusFromFullStr(status string) pb.Status {
	status = strings.ToLower(status)
	switch status {
	case "na":
		return pb.Status_NA
	case "unhealthy":
		return pb.Status_UNHEALTHY
	case "healthy":
		return pb.Status_HEALTHY
	case "maybe_unhealthy":
		return pb.Status_MAYBE_UNHEALTHY
	case "dying":
		return pb.Status_DYING
	case "dead":
		return pb.Status_DEAD
	default:
		return pb.Status_INVALID
	}
}

func StatusFromStr(status string) pb.Status {
	switch status {
	case "n":
		return pb.Status_NA
	case "u":
		return pb.Status_UNHEALTHY
	case "h":
		return pb.Status_HEALTHY
	case "m":
		return pb.Status_MAYBE_UNHEALTHY
	case "d":
		return pb.Status_DYING
	case "dd":
		return pb.Status_DEAD
	default:
		return pb.Status_INVALID
	}
}
