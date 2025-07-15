package hydra

import (
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
	"log"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Reader struct {
	Client *Hydra
	Name   string
}

var _ process.Consumer[*Credentials] = &Reader{}

func (r *Reader) Consume(ctx *stopper.Context, c *Credentials) error {
	active, err := r.Client.IntrospectToken(ctx, c.AccessToken)
	if err != nil {
		log.Printf("error calling hydra %s", err)
		metrics.ErrorCounter.WithLabelValues("hydra", "introspect", r.Name).Inc()
		return err
	}
	if active {
		metrics.MessageCounter.WithLabelValues("hydra", "introspect_active", r.Name).Inc()
		return nil
	}
	metrics.MessageCounter.WithLabelValues("hydra", "introspect_inactive", r.Name).Inc()
	return nil
}

func (r *Reader) String() string {
	return r.Name
}
