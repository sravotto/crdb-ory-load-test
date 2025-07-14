package keto

import (
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/process"
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
			metrics.ErrorCounter.WithLabelValues("keto", "write_tuple", w.Name).Inc()
			log.Printf("WriteTuple failed: %v", err)
		} else {
			metrics.MessageCounter.WithLabelValues("keto", "write_tuple", w.Name).Inc()
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
