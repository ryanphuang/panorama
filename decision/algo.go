package decision

import (
	dt "deephealth/types"
)

type InferenceAlgo interface {
	Infer(panorama *dt.Panorama) *dt.Inference
}
