package kratos

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
	"fmt"
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
		metrics.IdentityCheckCounter.WithLabelValues("active").Inc()
		r.Active++
		return nil
	}
	metrics.IdentityCheckCounter.WithLabelValues("inactive").Inc()
	r.Inactive++
	return nil
}

func (r *Reader) String() string {
	return r.Name
}

func BuildReaders(ctx *stopper.Context, cfg *config.Config, client *Kratos) []process.Consumer[*Identity] {
	readers := make([]process.Consumer[*Identity], cfg.Writers())
	for idx := range readers {
		reader := &Reader{
			Client: client,
			Name:   fmt.Sprintf("kratos-writer-%d", idx),
		}
		readers[idx] = reader
	}
	return readers
}
