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

type InferMap map[dt.EntityId]*dt.Inference

type HealthInferenceStorage struct {
	Results   InferMap
	Workbooks map[dt.EntityId]InferMap
	ReportCh  chan *dt.Report

	raw   *RawHealthStorage
	algo  dd.InferenceAlgo
	mu    *sync.Mutex
	alive bool
}

func NewHealthInferenceStorage(raw *RawHealthStorage, algo dd.InferenceAlgo) *HealthInferenceStorage {
	storage := &HealthInferenceStorage{
		Results:   make(InferMap),
		Workbooks: make(map[dt.EntityId]InferMap),
		ReportCh:  make(chan *dt.Report, 10),
		raw:       raw,
		algo:      algo,
		mu:        &sync.Mutex{},
		alive:     true,
	}
	return storage
}

func (self *HealthInferenceStorage) Infer(report *dt.Report) (*dt.Inference, error) {
	panorama, l := self.raw.GetPanorama(report.Subject)
	if panorama == nil || l == nil {
		return nil, fmt.Errorf("cannot get panorama for %s\n", report.Subject)
	}
	l.Lock()
	workbook, ok := self.Workbooks[report.Subject]
	if !ok {
		workbook = make(InferMap)
		self.Workbooks[report.Subject] = workbook
	}
	inference := self.algo.InferPano(panorama, workbook)
	l.Unlock()
	if inference == nil {
		return nil, fmt.Errorf("could not compute inference for %s\n", report.Subject)
	}
	dh.LogD(itag, "inference result for %s: %s", report.Subject, *inference.Observation)
	self.mu.Lock()
	self.Results[report.Subject] = inference
	self.mu.Unlock()
	return inference, nil
}

func (self *HealthInferenceStorage) GetInference(subject dt.EntityId) *dt.Inference {
	self.mu.Lock()
	inference, ok := self.Results[subject]
	self.mu.Unlock()
	if !ok {
		return nil
	}
	return inference
}

func (self *HealthInferenceStorage) Start() error {
	go func() {
		for self.alive {
			select {
			case report := <-self.ReportCh:
				dh.LogD(itag, "received report for %s for inference", report.Subject)
				self.Infer(report)
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
