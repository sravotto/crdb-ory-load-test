package keto

import (
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
	"log"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Reader struct {
	Client *Keto
	Name   string
}

var _ process.Consumer[*RelationTuple] = &Reader{}

func (r *Reader) Consume(ctx *stopper.Context, tuple *RelationTuple) error {
	allowed, err := r.Client.CheckPermission(ctx, tuple)
	if err != nil {
		log.Printf("error calling keto %s", err)
		metrics.ErrorCounter.WithLabelValues("keto", "check_permission", r.Name).Inc()
		return err
	}
	if allowed {
		metrics.MessageCounter.WithLabelValues("keto", "check_permission_allowed", r.Name).Inc()
		return nil
	}
	metrics.MessageCounter.WithLabelValues("keto", "check_permission_denied", r.Name).Inc()
	return nil
}

func (r *Reader) String() string {
	return r.Name
}
