package kratos

import (
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
	"log"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Reader struct {
	Client           *Kratos
	Name             string
	Active, Inactive int
}

var _ process.Consumer[*Identity] = &Reader{}

func (r *Reader) Consume(ctx *stopper.Context, identity *Identity) error {
	active, err := r.Client.CheckIdentity(ctx, identity.Email)
	if err != nil {
		log.Printf("error calling kratos %s", err)
		metrics.ErrorCounter.WithLabelValues("kratos", "check_identity", r.Name).Inc()
		return err
	}
	if active {
		metrics.MessageCounter.WithLabelValues("kratos", "check_identity_active", r.Name).Inc()
		r.Active++
		return nil
	}
	metrics.MessageCounter.WithLabelValues("kratos", "check_identity_inactive", r.Name).Inc()
	r.Inactive++
	return nil
}

func (r *Reader) String() string {
	return r.Name
}
