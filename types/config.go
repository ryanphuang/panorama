package types

type HealthServerConfig struct {
	Addr             string
	Owner            EntityId
	Subjects         []EntityId
	Peers            map[EntityId]string // all peers' id and address
	FilterSubmission bool                // whether to filter submitted report based on the subject id
}
