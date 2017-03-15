package store

type HStatus uint8

const (
	UNKNOWN HStatus = iota
	HEALTHY
	UNHEALTHY
	DYING
	DEAD
)

type HScore struct {
	status HStatus
	score  float32
}

type HSchema struct {
	names []string
}

type HVector struct {
	schema HSchema
	scores []HScore
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
