package hydra

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/google/uuid"
)

type Writer struct {
	Client *Hydra
	ID     string
	Secret string
	Name   string
}

var _ process.Producer[Credentials] = &Writer{}

func (w *Writer) Produce(ctx *stopper.Context) (Credentials, error) {
	ticker := time.NewTicker(defaultDelay)
	defer ticker.Stop()
	for {
		token, err := w.Client.GrantClientCredentials(ctx, w.ID, w.Secret)
		if err != nil || token == "" {
			log.Printf("error calling hydra %s", err)
			metrics.ErrorCounter.WithLabelValues("hydra", "grant", w.Name).Inc()
		} else {
			return Credentials{
				ClientID:     w.ID,
				ClientSecret: w.Secret,
				AccessToken:  token}, nil
		}
		select {
		case <-ctx.Stopping():
			return Credentials{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Writer) String() string {
	return w.Name
}

func BuildWriters(
	ctx *stopper.Context,
	cfg *config.Config,
	client *Hydra,
) ([]process.Producer[Credentials], error) {
	writers := make([]process.Producer[Credentials], cfg.Writers())
	for idx := range writers {
		writer := &Writer{
			Client: client,
			ID:     uuid.New().String(),
			Secret: uuid.New().String(),
			Name:   fmt.Sprintf("hydra-load-test-client-%d", idx),
		}
		writers[idx] = writer
		created, err := client.CreateOAuth2Client(ctx, writer)
		if err != nil || !created {
			return nil, errors.Join(err, errors.New("failed to create writer"))
		}
		log.Printf("Hydra OAuth2 Client Created with ID: %s", writer.Name)
	}
	return writers, nil
}
