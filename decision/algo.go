package decision

import (
	dt "deephealth/types"
)

type InferenceAlgo interface {
	InferPano(panorama *dt.Panorama) *dt.Inference
	InferView(view *dt.View) *dt.Inference
}
