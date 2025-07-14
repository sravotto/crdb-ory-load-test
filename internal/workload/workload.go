package workload

import (
	"log"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"

	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
	"crdb-ory-load-test/internal/process"
)

func Run[T any](ctx *stopper.Context, cfg *config.Config, client process.Client[T]) error {
	log.Printf("Load generation for %v with %d writers, %d readers, %d ratio",
		cfg.Duration(), cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)

	writers, err := client.BuildWriters(ctx, cfg)
	if err != nil {
		return err
	}
	readers, err := client.BuildReaders(ctx, cfg)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	messages := make(chan T, 100)
	defer close(messages)
	readerObs, err := observer.NewObserver(client.String()+"-reads",
		metrics.OryLatencyHistogram.WithLabelValues(client.String(), "read"),
	)
	if err != nil {
		return err
	}
	consumerPool := process.ConsumerPool[T]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  cfg.Duration(),
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, messages)
	writerObs, err := observer.NewObserver(client.String()+"-writes",
		metrics.OryLatencyHistogram.WithLabelValues(client.String(), "write"),
	)
	if err != nil {
		return err
	}
	producerPool := process.ProducerPool[T]{
		Producers: writers,
		Observer:  writerObs,
		Duration:  cfg.Duration(),
	}
	start := time.Now()
	printProgress(ctx, []*observer.Observer{readerObs, writerObs})
	producerPool.Start(ctx, &wg, messages)
	wg.Wait()
	printSummary(cfg, start, readerObs, writerObs)
	return nil
}
