package types

import "time"
import "container/list"

type Status uint8
type EntityId string

const (
	INVALID Status = iota
	NA
	HEALTHY
	MAYBE_UNHEALTHY
	UNHEALTHY
	DYING
	DEAD
)

// A value is a measurement of a particular metric
type Value struct {
	Status Status  // status for this metric
	Score  float32 // actual score for this metric
}

// A metric is a single aspect of an entity's health
type Metric struct {
	Name string // name of the metric, e.g., CPU, Network
	Value
}

type Values []*Value
type Metrics map[string]*Metric

// An observation is a collection of a metrics measuring
// an entity's health at a particular time
type Observation struct {
	Ts      time.Time // time when the observation was made
	Metrics Metrics   // actual scores for each metric
}

// A report is an observation attached with the observer and the observed (subject)
type Report struct {
	Observer    EntityId     // the entity that made the report
	Subject     EntityId     // the entity whose health is being reported by the observer
	Observation *Observation // the observation that reflects an entity's health
}

// A view is a continuous stream of reports made by an observer for a subject
type View struct {
	Observer     EntityId   // who made the observation
	Subject      EntityId   // the entity whose health is being reported by the observer
	Observations *list.List // all the observations for this subject reported by the observer
}

// A panorama is a collection of views from different observers about the same subject
type Panorama struct {
	Subject EntityId           // the entity whose health information is stored
	Views   map[EntityId]*View // various observers' reports about the subject
}

// An inference is a final summary of a entity's health based on the observations
// from different entities
type Inference struct {
	Subject     EntityId     // the entity whose health information is stored
	Observers   []EntityId   // the set of entities from whom the status was computed from
	Observation *Observation // the observation that reflects an entity's health
}

func (self *Observation) SetMetric(name string, status Status, score float32) bool {
	metric, ok := self.Metrics[name]
	if !ok {
		return false
	}
	metric.Status = status
	metric.Score = score
	return true
}

func (self *Observation) GetMetric(name string) *Metric {
	metric, ok := self.Metrics[name]
	if !ok {
		return nil
	}
	return metric
}

func (self *Observation) AddMetric(name string, status Status, score float32) *Observation {
	metric, ok := self.Metrics[name]
	if !ok {
		self.Metrics[name] = &Metric{name, Value{status, score}}
	} else {
		metric.Status = status
		metric.Score = score
	}
	return self
}

func NewObservation(time time.Time, names ...string) *Observation {
	metrics := make(Metrics)
	for _, name := range names {
		metrics[name] = &Metric{name, Value{INVALID, 0.0}}
	}
	return &Observation{Ts: time, Metrics: metrics}
}

func NewReport(observer EntityId, subject EntityId, metrics map[string]Value) *Report {
	o := NewObservation(time.Now())
	for k, v := range metrics {
		o.AddMetric(k, v.Status, v.Score)
	}
	return &Report{
		Observer:    observer,
		Subject:     subject,
		Observation: o,
	}
}
