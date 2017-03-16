package health

import "time"
import "container/list"

type HStatus uint8
type EntityId string

const (
	UNKNOWN HStatus = iota
	HEALTHY
	MAYBE_UNHEALTHY
	UNHEALTHY
	DYING
	DEAD
)

type HMetric struct {
	Name   string  // name of the metric, e.g., CPU, Network
	Status HStatus // status for this metric
	Score  float32 // actual score for this metric
}

type HObservation struct {
	Ts     time.Time  // time when the observation was made
	Vector []*HMetric // actual scores for each metric
}

type HReport struct {
	Observer    EntityId     // the entity that made the report
	Subject     EntityId     // the entity whose health is being reported by the observer
	Observation HObservation // the observation that reflects an entity's health
}

type HView struct {
	Observer     EntityId   // who made the observation
	Subject      EntityId   // the entity whose health is being reported by the observer
	Observations *list.List // all the observations for this subject reported by the observer
}

type HTable struct {
	Subject EntityId            // the entity whose health information is stored
	Views   map[EntityId]*HView // various observers' reports about the subject
}

func (self *HObservation) SetMetric(name string, status HStatus, score float32) bool {
	for _, metric := range self.Vector {
		if metric.Name == name {
			metric.Status = status
			metric.Score = score
			return true
		}
	}
	self.Vector = append(self.Vector, &HMetric{Name: name, Status: status, Score: score})
	return true
}

func NewHObservation(time time.Time, names ...string) *HObservation {
	vector := make([]*HMetric, len(names))
	for i, name := range names {
		vector[i] = &HMetric{Name: name, Status: UNKNOWN, Score: 0.0}
	}
	return &HObservation{Ts: time, Vector: vector}
}
