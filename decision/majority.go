package decision

import (
	"time"

	dt "deephealth/types"
)

type SimpleMajorityInference struct {
}

var _ InferenceAlgo = new(SimpleMajorityInference)

func (self SimpleMajorityInference) InferPano(panorama *dt.Panorama) *dt.Inference {
	var summary dt.Inference
	// for observer, view := range panorama.Views {

	// }
	return &summary
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
