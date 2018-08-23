package decision

import (
	"github.com/golang/protobuf/ptypes/timestamp"

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
	metrics := make(map[string]*pb.Metric)
	statmap := make(map[string]*valueStat)
	du.LogD(mtag, "infer panorama for %s:%s", panorama.Subject, dt.PanoramaString(panorama))
	var pts *timestamp.Timestamp = nil
	for observer, view := range panorama.Views {
		summary.Observers[i] = observer
		inference, ok := workbook[observer]
		if !ok {
			inference = self.InferView(view)
			if inference == nil {
				du.LogD(mtag, "empty view from %s", observer)
				continue
			}
			du.LogD(mtag, "summarized view from %s: %s", observer, dt.ObservationString(inference.Observation))
			workbook[observer] = inference
		} else {
			du.LogD(mtag, "found summary view from %s in workbook: %s", observer, dt.ObservationString(inference.Observation))
		}
		if pts == nil || dt.CompareTimestamp(pts, inference.Observation.Ts) < 0 {
			pts = inference.Observation.Ts
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
		metrics[name] = &pb.Metric{
			Name:  name,
			Value: &pb.Value{Status: maxstatus, Score: stat.ScoreSum / float32(stat.Cnt)},
		}
	}
	if pts == nil {
		// no observation found, no summary
		return nil
	}
	summary.Observation = &pb.Observation{Ts: pts, Metrics: metrics}
	return summary
}

func (self SimpleMajorityInference) InferView(view *pb.View) *pb.Inference {
  du.LogD(mtag, "inferring %d observations from %s", len(view.Observations), view.Observer)
	i := len(view.Observations) - 1
	if i < 0 {
		return nil
	}
	summary := &pb.Inference{
		Subject:   view.Subject,
		Observers: []string{view.Observer},
	}
	metrics := make(map[string]*pb.Metric)
	pts := view.Observations[i].Ts
	aggs := make(map[string]*aggCnt)
	for ; i >= 0; i-- {
		val := view.Observations[i]
    du.LogD(mtag, "[%s] observation %d: %s", view.Observer, i, dt.ObservationString(val))
		for name, metric := range val.Metrics {
			// fmt.Printf("time %v, name %s, metric %v\n", val.Ts, name, metric)
			agg, ok := aggs[name]
			if !ok {
				agg = &aggCnt{cnt: 0, stop: false}
				aggs[name] = agg
			}
			if agg.stop || agg.cnt >= VIEW_METRIC_HISTORY_SIZE {
        du.LogD(mtag, "[%s] observation %d: stop aggregating metric %s", view.Observer, i, name)
				// don't aggregate this metric any more
				continue
			}
			if !ok {
				metrics[name] = metric
				agg.cnt = agg.cnt + 1
        du.LogD(mtag, "[%s] observation %d: new metric %s", view.Observer, i, name)
			} else {
				m1 := metrics[name]
        du.LogD(mtag, "[%s] observation %d: previous metric for %s: %v", view.Observer, i, name, m1)
				if metric.Value.Status == pb.Status_PENDING && m1.Value.Status == pb.Status_HEALTHY {
					// if the current status is healthy and the older status is pending,
					// then the two statuses get merged to healthy because the pending status
					// is only a temporary status
          du.LogI(mtag, "[%s] observation %d: resolved a pending status for metric %s", view.Observer, i, name)

					// here, we don't increment agg cnt, which means that we will keep resolving
					// TODO: it may be necessary to set a limit on the resolving
					continue
				} else if m1.Value.Status != metric.Value.Status {
					// if the two metrics have different statuses
					// the recent one always override the old one.
					// the look back stops.
          du.LogI(mtag, "[%s] observation %d: metric %s from a previous observation has a different status: now %s, previous %s", 
              view.Observer, i, name, metric.Value.Status.String(), m1.Value.Status.String())
					agg.stop = true
					continue
				} else {
          du.LogD(mtag, "[%s] observation %d: aggregating metric %s of %s status with score %.1f", view.Observer, i, name, metric.Value.Status.String(), metric.Value.Score)
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
	summary.Observation = &pb.Observation{Ts: pts, Metrics: metrics}
	return summary
}
