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
var mtag = "majority"

const (
	// within a given metric in a view, only look back at the last N values
	VIEW_METRIC_HISTORY_SIZE = 2
)

type valueStat struct {
	ScoreSum   float32
	Cnt        uint32
	StatusHist map[pb.Status]uint32
}

type aggCnt struct {
	cnt  uint32
	stop bool
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
			du.LogD(mtag, "summarized view from %s: %s", observer, dt.ObservationString(inference.Observation))
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
		du.LogD(mtag, "stat for metric %s: score_sum=%f,cnt=%d,status_hist=%v", name, stat.ScoreSum, stat.Cnt, stat.StatusHist)
		var maxcnt uint32 = 0
		maxstatus := pb.Status_HEALTHY
		for status, cnt := range stat.StatusHist {
			if cnt > maxcnt {
				maxcnt = cnt
				maxstatus = status
			} else if cnt == maxcnt && status > maxstatus {
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
	i := len(view.Observations) - 1
	if i < 0 {
		return summary
	}
	metrics := make(map[string]*pb.Metric)
	pts := view.Observations[i].Ts
	aggs := make(map[string]*aggCnt)
	for ; i >= 0; i-- {
		val := view.Observations[i]
		for name, metric := range val.Metrics {
			agg, ok := aggs[name]
			if !ok {
				agg = &aggCnt{cnt: 0, stop: false}
				aggs[name] = agg
			}
			if agg.stop || agg.cnt >= VIEW_METRIC_HISTORY_SIZE {
				// don't aggregate this metric any more
				continue
			}
			if !ok {
				metrics[name] = metric
				agg.cnt = agg.cnt + 1
			} else {
				m1 := metrics[name]
				if m1.Value.Status != metric.Value.Status {
					// if the two metrics have different statuses
					// the recent one always override the old one.
					// the look back stops.
					agg.stop = true
					continue
				} else {
					m1.Value.Score += metric.Value.Score
					agg.cnt = agg.cnt + 1
				}
			}
		}
	}
	for name, metric := range metrics {
		if aggs[name].cnt > 1 {
			du.LogD(mtag, "average score for metric %s: %f/%d", metric.Name, metric.Value.Score, aggs[name].cnt)
			metric.Value.Score = metric.Value.Score / float32(aggs[name].cnt)
		}
	}
	summary.Observation = &pb.Observation{pts, metrics}
	return summary
}
