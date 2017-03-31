package decision

import (
	dt "deephealth/types"
)

type SimpleMajorityInference struct {
}

var _ InferenceAlgo = new(SimpleMajorityInference)

func (self SimpleMajorityInference) Infer(panorama *dt.Panorama) *dt.Inference {
	var summary dt.Inference
	// for observer, view := range panorama.Views {
	// }
	return &summary
}
