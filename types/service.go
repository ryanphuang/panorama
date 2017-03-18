package types

type HealthStorage interface {
	// Add a subject to the observing subject list
	ObserveSubject(subject EntityId) bool

	// Stop observing a particular subject, all the reports
	// concerning this subject will be ignored
	StopObservingSubject(subject EntityId) bool

	// Add a report to the view storage
	AddReport(report *Report) (int, error)
}

type HealthGossip interface {
	// Gossip a report to other peers
	GossipReport(report *Report) int
}

type HealthStatus interface {
	// Query the health status of an entity
	GetReport(subject EntityId) *Report
}

type HealthServerConfig struct {
	Addr     string
	Owner    EntityId
	Subjects []EntityId
}
