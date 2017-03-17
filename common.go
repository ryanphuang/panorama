package deephealth

type HealthStorage interface {
	// Add a subject to the observing subject list
	ObserveSubject(subject EntityId, reply *bool) error

	// Stop observing a particular subject, all the reports
	// concerning this subject will be ignored
	StopObservingSubject(subject EntityId, reply *bool) error

	// Add a report to the view storage
	AddReport(report *Report, reply *int) error
}

type HealthGossip interface {
	// Gossip a report to other peers
	GossipReport(report *Report, reply *int) error
}

type HealthStatus interface {
	// Query the health status of an entity
	GetReport(subject EntityId, reply *Report) error
}

type HealthService interface {
	HealthStorage
	HealthGossip
	HealthStatus
}
