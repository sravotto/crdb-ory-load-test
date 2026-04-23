package workload

import (
	"log"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"golang.org/x/time/rate"

	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
	"crdb-ory-load-test/internal/process"
)

func Run[T any](ctx *stopper.Context, cfg *config.Config, client process.Client[T]) error {
	if cfg.Workload.MaxRate > 0 {
		log.Printf("Load generation for %v with %d writers, %d readers, %d ratio, %d max ops/sec",
			cfg.Duration(), cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio, cfg.Workload.MaxRate)
	} else {
		log.Printf("Load generation for %v with %d writers, %d readers, %d ratio (unlimited rate)",
			cfg.Duration(), cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)
	}

	writers, err := client.BuildWriters(ctx, cfg)
	if err != nil {
		return err
	}
	readers, err := client.BuildReaders(ctx, cfg)
	if err != nil {
		return err
	}

	var limiter *rate.Limiter
	if cfg.Workload.MaxRate > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.Workload.MaxRate), max(1, cfg.Workload.MaxRate))
	}

	var wg sync.WaitGroup
	messages := make(chan T, cfg.Readers())
	defer close(messages)
	readerObs, err := observer.NewObserver(client.String()+"-reads",
		metrics.OryLatencyHistogram.WithLabelValues(client.String(), "read"),
	)
	if err != nil {
		return err
	}
	consumerPool := process.ConsumerPool[T]{
		Consumers:      readers,
		Observer:       readerObs,
		Duration:       cfg.Duration(),
		Repeats:        cfg.Workload.ReadRatio,
		TolerateErrors: cfg.Workload.TolerateErrors,
		Limiter:        limiter,
	}
	consumerPool.Start(ctx, &wg, messages)
	writerObs, err := observer.NewObserver(client.String()+"-writes",
		metrics.OryLatencyHistogram.WithLabelValues(client.String(), "write"),
	)
	if err != nil {
		return err
	}
	producerPool := process.ProducerPool[T]{
		Producers:      writers,
		Observer:       writerObs,
		Duration:       cfg.Duration(),
		TolerateErrors: cfg.Workload.TolerateErrors,
		Limiter:        limiter,
	}
	start := time.Now()
	printProgress(ctx, []observer.Observer{readerObs, writerObs})
	producerPool.Start(ctx, &wg, messages)
	wg.Wait()
	printSummary(cfg, start, readerObs, writerObs)
	return nil
}
