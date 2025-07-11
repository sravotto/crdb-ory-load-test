package keto

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/process"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/google/uuid"
)

type Writer struct {
	Client *Keto
	Name   string
}

var _ process.Producer[*RelationTuple] = &Writer{}

func (w *Writer) Produce(ctx *stopper.Context) (*RelationTuple, error) {
	ticker := time.NewTicker(defaultDelay)
	defer ticker.Stop()
	for {
		tuple := &RelationTuple{
			Namespace: namespace,
			Object:    uuid.New().String(),
			SubjectID: "user:" + uuid.New().String(),
			Relation:  relation,
		}
		err := w.Client.WriteTuple(ctx, tuple)
		if err != nil {
			log.Printf("WriteTuple failed: %v", err)
		} else {
			return tuple, nil
		}
		select {
		case <-ctx.Stopping():
			return &RelationTuple{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Writer) String() string {
	return w.Name
}

func BuildWriters(ctx *stopper.Context, cfg *config.Config, client *Keto) []process.Producer[*RelationTuple] {
	writers := make([]process.Producer[*RelationTuple], cfg.Writers())
	for idx := range writers {
		writer := &Writer{
			Client: client,
			Name:   fmt.Sprintf("keto-writer-%d", idx),
		}
		writers[idx] = writer
	}
	return writers
}
