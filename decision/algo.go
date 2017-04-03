package decision

import (
	dt "deephealth/types"
)

type InferenceAlgo interface {
	InferPano(panorama *dt.Panorama, workbook map[dt.EntityId]*dt.Inference) *dt.Inference
	InferView(view *dt.View) *dt.Inference
}
