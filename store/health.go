package store

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
	name   string  // name of the metric, e.g., CPU, Network
	status HStatus // status for this metric
	score  float32 // actual score for this metric
}

type HObservation struct {
	ts     time.Time  // time when the observation was made
	vector []*HMetric // actual scores for each metric
}

type HReport struct {
	observer    EntityId     // the entity that made the report
	subject     EntityId     // the entity whose health is being reported by the observer
	observation HObservation // the observation that reflects an entity's health
}

type HView struct {
	observer     EntityId   // who made the observation
	subject      EntityId   // the entity whose health is being reported by the observer
	observations *list.List // all the observations for this subject reported by the observer
}

type HTable struct {
	subject EntityId            // the entity whose health information is stored
	views   map[EntityId]*HView // various observers' reports about the subject
}

func (self *HObservation) SetMetric(name string, status HStatus, score float32) bool {
	for _, metric := range self.vector {
		if metric.name == name {
			metric.status = status
			metric.score = score
			return true
		}
	}
	self.vector = append(self.vector, &HMetric{name: name, status: status, score: score})
	return true
}

func NewHObservation(time time.Time, names ...string) *HObservation {
	vector := make([]*HMetric, len(names))
	for i, name := range names {
		vector[i] = &HMetric{name: name, status: UNKNOWN, score: 0.0}
	}
	return &HObservation{ts: time, vector: vector}
}
