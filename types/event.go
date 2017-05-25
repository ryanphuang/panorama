package types

import (
	"time"

	pb "deephealth/build/gen"
)

type Event struct {
	Time    time.Time
	Id      string
	Subject string
	Context string
	Status  pb.Status
	Score   float32
	Extra   string
}

type EventMap struct {
	Fields map[string]string
}

type EventParser interface {
	ParseLine(line string) *Event
}
