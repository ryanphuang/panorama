package decision

import (
	"time"

	pb "deephealth/build/gen"
	dt "deephealth/types"
	du "deephealth/util"
)

type SimpleMajorityInference struct {
	ViewSummaries map[string]*pb.Inference
}

var _ InferenceAlgo = new(SimpleMajorityInference)

type valueStat struct {
	ScoreSum   float32
	Cnt        uint32
	StatusHist map[pb.Status]uint32
}

func (self SimpleMajorityInference) InferPano(panorama *pb.Panorama, workbook map[string]*pb.Inference) *pb.Inference {
	summary := &pb.Inference{
		Subject:   panorama.Subject,
		Observers: make([]string, len(panorama.Views)),
	}
	i := 0
	observation := dt.NewObservation(time.Now())
	statmap := make(map[string]*valueStat)
	for observer, view := range panorama.Views {
		summary.Observers[i] = observer
		inference, ok := workbook[observer]
		if !ok {
			inference = self.InferView(view)
			workbook[observer] = inference
		}
		for name, metric := range inference.Observation.Metrics {
			stat, ok := statmap[name]
			if !ok {
				stat = &valueStat{
					ScoreSum:   0.0,
					Cnt:        0,
					StatusHist: make(map[pb.Status]uint32),
				}
				statmap[name] = stat
			}
			stat.ScoreSum += metric.Value.Score
			stat.Cnt++
			stat.StatusHist[metric.Value.Status]++
		}
		i++
	}
	for name, stat := range statmap {
		du.LogD("decision", "score sum for %s is %f", name, stat.ScoreSum)
		var maxcnt uint32 = 0
		maxstatus := pb.Status_HEALTHY
		for status, cnt := range stat.StatusHist {
			if cnt > maxcnt {
				maxcnt = cnt
				maxstatus = status
			}
		}
		observation.Metrics[name] = &pb.Metric{name, &pb.Value{maxstatus, stat.ScoreSum / float32(stat.Cnt)}}
	}
	summary.Observation = observation
	return summary
}

func (self SimpleMajorityInference) InferView(view *pb.View) *pb.Inference {
	summary := &pb.Inference{
		Subject:   view.Subject,
		Observers: []string{view.Observer},
	}
	observation := dt.NewObservation(time.Now())
	for i := len(view.Observations) - 1; i >= 0; i-- {
		val := view.Observations[i]
		for name, metric := range val.Metrics {
			_, ok := observation.Metrics[name]
			if ok {
				continue
			}
			observation.Metrics[name] = metric
		}
	}
	summary.Observation = observation
	return summary
}
