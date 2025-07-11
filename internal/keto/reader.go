package keto

import (
	"crdb-ory-load-test/cmd/process"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"
	"fmt"
	"log"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Reader struct {
	Client          *Keto
	Name            string
	Allowed, Denied int
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
		metrics.PermissionCheckCounter.WithLabelValues("allowed").Inc()
		r.Allowed++
		return nil
	}
	metrics.PermissionCheckCounter.WithLabelValues("denied").Inc()
	r.Denied++
	return nil
}

func (r *Reader) String() string {
	return r.Name
}

func BuildReaders(ctx *stopper.Context, cfg *config.Config, client *Keto) []process.Consumer[*RelationTuple] {
	readers := make([]process.Consumer[*RelationTuple], cfg.Writers())
	for idx := range readers {
		reader := &Reader{
			Client: client,
			Name:   fmt.Sprintf("keto-writer-%d", idx),
		}
		readers[idx] = reader
	}
	return readers
}
