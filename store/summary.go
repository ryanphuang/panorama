package store

import (
	"sync"

	dd "deephealth/decision"
	dt "deephealth/types"
)

type HealthSummaryStorage struct {
	Tenants map[dt.EntityId]*dt.Inference

	raw  *RawHealthStorage
	algo *dd.InferenceAlgo
	mu   *sync.Mutex
}

func NewHealthSummaryStorage(raw *RawHealthStorage, algo *dd.InferenceAlgo) *HealthSummaryStorage {
	storage := &HealthSummaryStorage{
		Tenants: make(map[dt.EntityId]*dt.Inference),
		raw:     raw,
		algo:    algo,
		mu:      &sync.Mutex{},
	}
	return storage
}

func (self *HealthSummaryStorage) Summarize(subject dt.EntityId) {

}
