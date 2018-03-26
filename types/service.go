package types

import (
	"database/sql"
	"time"

	pb "deephealth/build/gen"
)

// Simple tuple about the local observer
// that will monitor health of a component
type ObserverModule struct {
	Module   string
	Observer string
}

type HealthStorage interface {
	// Associate database with the raw storage
	SetDB(db HealthDB)

	// Add a subject to the observing subject list
	AddSubject(subject string) bool

	// Stop observing a particular subject, all the reports
	// concerning this subject will be ignored
	RemoveSubject(subject string, clean bool) bool

	// Get the list of subjects that we have observed
	GetSubjects() map[string]time.Time

	// Add a report to the view storage
	AddReport(report *pb.Report, filter bool) (int, error)

	// Get the latest report for a subject
	GetLatestReport(subject string) *pb.Report

	// Get the view from an observer about a subject
	GetView(observer string, subject string) *pb.View

	// Get the whole panorama for a subject
	GetPanorama(subject string) *ConcurrentPanorama

	// Get all the panoramas for all observed subjects
	DumpPanorama() map[string]*pb.Panorama

	// Garbage collect stale observations from panoramas
	// Return number of reaped observations for a subject
	// When relative is true, the GC is based on elapsed
	// time with the most recent observation rather than
	// the absolute time now.
	GC(ttl time.Duration, relative bool) map[string]uint32
}

type HealthInference interface {
	// Associate database with the raw storage
	SetDB(db HealthDB)

	// Asynchronously infer the health of a subject
	InferSubjectAsync(subject string) error

	// Infer the health of a subject
	InferSubject(subject string) (*pb.Inference, error)

	// Asynchronously infer the health of a subject that has a new report
	// May support incremental inference
	InferReportAsync(report *pb.Report) error

	// Infer the health of a subject that has a new report
	// May support incremental inference
	InferReport(report *pb.Report) (*pb.Inference, error)

	// Get the health inference of a subject
	GetInference(subject string) *pb.Inference

	// Get all the health inference for all observed subjects
	DumpInference() map[string]*pb.Inference

	// Start the inference service
	Start() error

	// Stop the inference service
	Stop() error
}

type HealthDB interface {
	// Open or create a database with file name
	Open() (*sql.DB, error)

	// Insert a report into the database
	InsertReport(report *pb.Report) error

	// Insert an inference result into the database
	InsertInference(inf *pb.Inference) error

	// Close the database connection
	Close()
}

type HealthExchange interface {
	// Propagate a report to other peers
	Propagate(report *pb.Report) error

	// Ping one peer and get a response
	Ping(peer string) (*pb.PingReply, error)

	// Ping all peers and get response
	PingAll() (map[string]*pb.PingReply, error)

	// peer is interested in a particular subject
	Interested(peer string, subject string) bool

	// peer is not interested in a particular subject
	Uninterested(peer string, subject string) bool
}
