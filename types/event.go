package types

import (
	"time"
)

type Event struct {
	Time    time.Time
	Id      string
	Subject string
	Context string
	Extra   string
}

type EventParser interface {
	ParseLine(line string) *Event
}
