package hydra

import (
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
	"log"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Writer struct {
	Client *Hydra
	ID     string
	Secret string
	Name   string
}

var _ process.Producer[*Credentials] = &Writer{}

func (w *Writer) Produce(ctx *stopper.Context) (*Credentials, error) {
	ticker := time.NewTicker(defaultDelay)
	defer ticker.Stop()
	for {
		token, err := w.Client.GrantClientCredentials(ctx, w.ID, w.Secret)
		if err != nil || token == "" {
			log.Printf("error calling hydra %s", err)
			metrics.ErrorCounter.WithLabelValues("hydra", "grant", w.Name).Inc()
		} else {
			return &Credentials{
				ClientID:     w.ID,
				ClientSecret: w.Secret,
				AccessToken:  token}, nil
		}
		select {
		case <-ctx.Stopping():
			return &Credentials{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Writer) String() string {
	return w.Name
}
