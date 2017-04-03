package decision

import (
	"time"

	dh "deephealth"
	dt "deephealth/types"
)

type SimpleMajorityInference struct {
	ViewSummaries map[dt.EntityId]*dt.Inference
}

var _ InferenceAlgo = new(SimpleMajorityInference)

type valueStat struct {
	ScoreSum   float32
	Cnt        uint32
	StatusHist map[dt.Status]uint32
}

func (self SimpleMajorityInference) InferPano(panorama *dt.Panorama, workbook map[dt.EntityId]*dt.Inference) *dt.Inference {
	summary := &dt.Inference{
		Subject:   panorama.Subject,
		Observers: make([]dt.EntityId, len(panorama.Views)),
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
					StatusHist: make(map[dt.Status]uint32),
				}
				statmap[name] = stat
			}
			stat.ScoreSum += metric.Score
			stat.Cnt++
			stat.StatusHist[metric.Status]++
		}
		i++
	}
	for name, stat := range statmap {
		dh.LogD("decision", "score sum for %s is %f", name, stat.ScoreSum)
		var maxcnt uint32 = 0
		maxstatus := dt.HEALTHY
		for status, cnt := range stat.StatusHist {
			if cnt > maxcnt {
				maxcnt = cnt
				maxstatus = status
			}
		}
		observation.Metrics[name] = &dt.Metric{name, dt.Value{maxstatus, stat.ScoreSum / float32(stat.Cnt)}}
	}
	summary.Observation = observation
	return summary
}

func (self SimpleMajorityInference) InferView(view *dt.View) *dt.Inference {
	summary := &dt.Inference{
		Subject:   view.Subject,
		Observers: []dt.EntityId{view.Observer},
	}
	observation := dt.NewObservation(time.Now())
	for e := view.Observations.Back(); e != nil; e = e.Prev() {
		val := e.Value.(*dt.Observation)
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
