package kratos

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/process"
	"fmt"
	"log"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Writer struct {
	Client *Kratos
	Name   string
}

var _ process.Producer[*Identity] = &Writer{}

func (w *Writer) Produce(ctx *stopper.Context) (*Identity, error) {
	ticker := time.NewTicker(defaultDelay)
	defer ticker.Stop()
	for {
		identity := &Identity{
			Email:     gofakeit.Email(),
			FirstName: gofakeit.FirstName(),
			LastName:  gofakeit.LastName(),
		}
		err := w.Client.RegisterIdentity(ctx, identity, gofakeit.Password(true, true, true, true, false, 8))
		if err != nil {
			log.Printf("RegisterIdentity failed: %v", err)
		} else {
			return identity, nil
		}
		select {
		case <-ctx.Stopping():
			return &Identity{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Writer) String() string {
	return w.Name
}

func BuildWriters(ctx *stopper.Context, cfg *config.Config, client *Kratos) []process.Producer[*Identity] {
	writers := make([]process.Producer[*Identity], cfg.Writers())
	for idx := range writers {
		writer := &Writer{
			Client: client,
			Name:   fmt.Sprintf("kratos-writer-%d", idx),
		}
		writers[idx] = writer
	}
	return writers
}
