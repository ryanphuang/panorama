package service

import (
	dt "deephealth/types"
)

type HealthServerConfig struct {
	Addr             string
	Owner            dt.EntityId
	Subjects         []dt.EntityId
	Peers            map[dt.EntityId]string // all peers' name and address
	FilterSubmission bool                   // whether to filter submitted report based on the subject id
}
