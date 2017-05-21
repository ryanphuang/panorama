package store

import (
	"fmt"
	"sync"

	dd "deephealth/decision"
	dt "deephealth/types"
	du "deephealth/util"

	pb "deephealth/build/gen"
)

const (
	itag = "inference"
)

type InferMap map[string]*pb.Inference

type HealthInferenceStorage struct {
	Results   InferMap
	Workbooks map[string]InferMap
	ReportCh  chan *pb.Report

	raw   *RawHealthStorage
	algo  dd.InferenceAlgo
	mu    *sync.Mutex
	alive bool
}

func NewHealthInferenceStorage(raw *RawHealthStorage, algo dd.InferenceAlgo) *HealthInferenceStorage {
	storage := &HealthInferenceStorage{
		Results:   make(InferMap),
		Workbooks: make(map[string]InferMap),
		ReportCh:  make(chan *pb.Report, 10),
		raw:       raw,
		algo:      algo,
		mu:        &sync.Mutex{},
		alive:     true,
	}
	return storage
}

func (self *HealthInferenceStorage) Infer(report *pb.Report) (*pb.Inference, error) {
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
	du.LogD(itag, "inference result for %s: %s", report.Subject, dt.ObservationString(inference.Observation))
	self.mu.Lock()
	self.Results[report.Subject] = inference
	self.mu.Unlock()
	return inference, nil
}

func (self *HealthInferenceStorage) GetInference(subject string) *pb.Inference {
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
				du.LogD(itag, "received report for %s for inference", report.Subject)
				self.Infer(report)
			}
		}
	}()
	return nil
}

func (self *HealthInferenceStorage) Stop() error {
	self.alive = false
	var report pb.Report
	select {
	case self.ReportCh <- &report:
		du.LogI(itag, "send empty report to stop the service")
	default:
	}
	return nil
}
