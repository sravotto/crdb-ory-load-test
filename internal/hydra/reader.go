package hydra

import (
	"crdb-ory-load-test/cmd/process"
	"crdb-ory-load-test/internal/metrics"
	"log"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Reader struct {
	Client           *Hydra
	Name             string
	Active, Inactive int
}

var _ process.Consumer[Credentials] = &Reader{}

func (r *Reader) Consume(ctx *stopper.Context, c Credentials) error {
	active := false
	var err error
	active, err = r.Client.IntrospectToken(ctx, c.AccessToken)
	if err != nil {
		log.Printf("error calling hydra %s", err)
		metrics.ErrorCounter.WithLabelValues("hydra", "introspect", r.Name).Inc()
	}
	if active {
		r.Active++
		metrics.OAuthTokenCheckCounter.WithLabelValues("active").Inc()
	} else {
		r.Inactive++
		metrics.OAuthTokenCheckCounter.WithLabelValues("inactive").Inc()
	}
	return nil
}

func (r *Reader) String() string {
	return r.Name
}
