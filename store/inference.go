package store

import (
	"fmt"
	"sync"

	dh "deephealth"
	dd "deephealth/decision"
	dt "deephealth/types"
)

const (
	itag = "inference"
)

type HealthInferenceStorage struct {
	Tenants  map[dt.EntityId]*dt.Inference
	ReportCh chan *dt.Report

	raw   *RawHealthStorage
	algo  dd.InferenceAlgo
	mu    *sync.Mutex
	alive bool
}

func NewHealthInferenceStorage(raw *RawHealthStorage, algo dd.InferenceAlgo) *HealthInferenceStorage {
	storage := &HealthInferenceStorage{
		Tenants:  make(map[dt.EntityId]*dt.Inference),
		ReportCh: make(chan *dt.Report, 10),
		raw:      raw,
		algo:     algo,
		mu:       &sync.Mutex{},
		alive:    true,
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

func (self *HealthInferenceStorage) Start() error {
	go func() {
		for self.alive {
			select {
			case report := <-self.ReportCh:
				dh.LogD(itag, "received report for %s for inference", report.Subject)
			}
		}
	}()
	return nil
}

func (self *HealthInferenceStorage) Stop() error {
	self.alive = false
	var report dt.Report
	select {
	case self.ReportCh <- &report:
		dh.LogI(itag, "send empty report to stop the service")
	default:
	}
	return nil
}
