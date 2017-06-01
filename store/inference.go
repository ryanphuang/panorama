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
	SubjectCh chan string

	raw   *RawHealthStorage
	algo  dd.InferenceAlgo
	mu    *sync.RWMutex
	alive bool
}

func NewHealthInferenceStorage(raw *RawHealthStorage, algo dd.InferenceAlgo) *HealthInferenceStorage {
	storage := &HealthInferenceStorage{
		Results:   make(InferMap),
		Workbooks: make(map[string]InferMap),
		ReportCh:  make(chan *pb.Report, 50),
		SubjectCh: make(chan string, 50),
		raw:       raw,
		algo:      algo,
		mu:        &sync.RWMutex{},
		alive:     true,
	}
	return storage
}

func (self *HealthInferenceStorage) InferSubjectAsync(subject string) error {
	// simply sent it to channel and return
	self.SubjectCh <- subject
	return nil
}

func (self *HealthInferenceStorage) InferReportAsync(report *pb.Report) error {
	// simply sent it to channel and return
	self.ReportCh <- report
	return nil
}

func (self *HealthInferenceStorage) InferSubject(subject string) (*pb.Inference, error) {
	pano := self.raw.GetPanorama(subject)
	if pano == nil {
		return nil, fmt.Errorf("cannot get panorama for %s\n", subject)
	}
	workbook, ok := self.Workbooks[subject]
	if !ok {
		workbook = make(InferMap)
		self.Workbooks[subject] = workbook
	}
	pano.RLock()
	inference := self.algo.InferPano(pano.Value, workbook)
	pano.RUnlock()
	if inference == nil {
		return nil, fmt.Errorf("could not compute inference for %s\n", subject)
	}
	du.LogD(itag, "inference result for %s: %s", subject, dt.ObservationString(inference.Observation))
	self.mu.Lock()
	self.Results[subject] = inference
	self.mu.Unlock()
	return inference, nil
}

func (self *HealthInferenceStorage) InferReport(report *pb.Report) (*pb.Inference, error) {
	// For now, we just re-do the inference
	// TODO: support incremental inference
	return self.InferSubject(report.Subject)
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

func (self *HealthInferenceStorage) DumpInference() map[string]*pb.Inference {
	return self.Results
}

func (self *HealthInferenceStorage) Start() error {
	go func() {
		for self.alive {
			select {
			case subject := <-self.SubjectCh:
				du.LogD(itag, "perform inference on subject for %s", subject)
				self.InferSubject(subject)
			case report := <-self.ReportCh:
				du.LogD(itag, "received report for %s for inference", report.Subject)
				self.InferReport(report)
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
