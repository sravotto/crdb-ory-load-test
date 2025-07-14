package generator

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/kratos"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
	"crdb-ory-load-test/internal/process"
	"log"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

func RunKratosWorkload(ctx *stopper.Context, cfg *config.Config) error {
	if err := cfg.CheckKratos(); err != nil {
		return err
	}
	client := kratos.New(cfg)

	if err := client.HealthCheck(ctx); err != nil {
		return err
	}

	log.Printf("Kratos Load generation for %v with %d writers, %d readers, %d ratio",
		cfg.Duration(), cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)
	writers := kratos.BuildWriters(ctx, cfg, client)
	readers := kratos.BuildReaders(ctx, cfg, client)

	var wg sync.WaitGroup
	tupleChannel := make(chan *kratos.Identity, 100)
	defer close(tupleChannel)
	readerObs, err := observer.NewObserver("kratos-reads",
		metrics.OryLatencyHistogram.WithLabelValues("kratos", "read"),
	)
	if err != nil {
		return err
	}
	consumerPool := process.ConsumerPool[*kratos.Identity]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  cfg.Duration(),
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, tupleChannel)
	writerObs, err := observer.NewObserver("kratos-writes",
		metrics.OryLatencyHistogram.WithLabelValues("kratos", "writes"),
	)
	if err != nil {
		return err
	}
	producerPool := process.ProducerPool[*kratos.Identity]{
		Producers: writers,
		Observer:  writerObs,
		Duration:  cfg.Duration(),
	}
	start := time.Now()
	printProgress(ctx, []*observer.Observer{readerObs, writerObs})
	producerPool.Start(ctx, &wg, tupleChannel)
	wg.Wait()
	printSummary(cfg, start, readerObs, writerObs)
	return nil

}
