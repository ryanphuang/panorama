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
	ts     Time      // time when the observation was made
	vector []HMetric // actual scores for each metric
}

type HObservations *list.List // list of observations

type HReport struct {
	observer    EntityId     // the entity that made the report
	subject     EntityId     // the entity whose health is being reported by the observer
	observation HObservation // the observation that reflects an entity's health
}

type HView struct {
	observer     EntityId      // who made the observation
	subject      EntityId      // the entity whose health is being reported by the observer
	observations HObservations // all the observations for this subject reported by the observer
}

type HTable struct {
	subject EntityId            // the entity whose health information is stored
	views   map[EntityId]*HView // various observers' reports about the subject
}

func NewHVector(names ...string) *HVector {
	var schema HSchema
	schema.names = make([]string, len(names))
	scores := make([]HScore, len(names))
	for i, name := range names {
		schema.names[i] = name
		scores[i] = HScore{status: UNKNOWN, score: 0.0}
	}
	vector := new(HVector)
	vector.schema = schema
	vector.scores = scores
	return vector
}
