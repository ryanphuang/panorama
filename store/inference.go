package store

import (
	"fmt"
	"sync"

	dd "deephealth/decision"
	dt "deephealth/types"
)

type HealthInferenceStorage struct {
	Tenants map[dt.EntityId]*dt.Inference

	raw  *RawHealthStorage
	algo dd.InferenceAlgo
	mu   *sync.Mutex
}

func NewHealthInferenceStorage(raw *RawHealthStorage, algo dd.InferenceAlgo) *HealthInferenceStorage {
	storage := &HealthInferenceStorage{
		Tenants: make(map[dt.EntityId]*dt.Inference),
		raw:     raw,
		algo:    algo,
		mu:      &sync.Mutex{},
	}
	return storage
}

func (self *HealthInferenceStorage) Infer(subject dt.EntityId) (*dt.Inference, error) {
	panorama, l := self.raw.GetPanorama(subject)
	if panorama == nil || l == nil {
		return nil, fmt.Errorf("cannot get panorama for %s\n", subject)
	}
	l.Lock()
	inference := self.algo.Infer(panorama)
	l.Unlock()
	if inference == nil {
		return nil, fmt.Errorf("could not compute inference for %s\n", subject)
	}
	self.mu.Lock()
	self.Tenants[subject] = inference
	self.mu.Unlock()
	return inference, nil
}

func (self *HealthInferenceStorage) GetInference(subject dt.EntityId) *dt.Inference {
	self.mu.Lock()
	inference := self.Tenants[subject]
	self.mu.Unlock()
	return inference
}
