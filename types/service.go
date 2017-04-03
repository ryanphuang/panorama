package types

import (
	"sync"
)

type HealthStorage interface {
	// Add a subject to the observing subject list
	AddSubject(subject EntityId) bool

	// Stop observing a particular subject, all the reports
	// concerning this subject will be ignored
	RemoveSubject(subject EntityId, clean bool) bool

	// Add a report to the view storage
	AddReport(report *Report, filter bool) (int, error)

	// Get the latest report for a subject
	GetLatestReport(subject EntityId) *Report

	// Get the whole panorama for a subject
	GetPanorama(subject EntityId) (*Panorama, *sync.Mutex)
}

type HealthInference interface {
	// Infer the health of a subject
	Infer(subject EntityId) (*Inference, error)

	// Get the health inference of a subject
	GetInference(subject EntityId) *Inference

	// Start the inference service
	Start() error

	// Stop the inference service
	Stop() error
}

type HealthGossip interface {
	// Gossip a report to other peers
	GossipReport(report *Report) int
}
