package decision

import (
	"sync"

	dt "deephealth/types"
)

type InferenceAlgo interface {
	summarize(panorama *dt.Panorama, lock *sync.Mutex) *dt.Status
}
