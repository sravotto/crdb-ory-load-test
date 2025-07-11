package keto

import "time"

var defaultDelay = 100 * time.Millisecond

const (
	namespace = "videos"
	relation  = "viewer"
)

type CheckResponse struct {
	Allowed bool `json:"allowed"`
}

type RelationTuple struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	SubjectID string `json:"subject_id"`
}
