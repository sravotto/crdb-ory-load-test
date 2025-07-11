package generator

import (
	"log"
	"sync"
	"time"

	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/keto"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
	"crdb-ory-load-test/internal/process"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

const namespace = "videos"

type tuple struct {
	Subject string
	Object  string
}

func RunKetoWorkload(ctx *stopper.Context, cfg *config.Config) error {
	if err := cfg.CheckKeto(); err != nil {
		return err
	}
	client := keto.New(cfg)
	if err := client.HealthCheck(ctx); err != nil {
		return err
	}

	log.Printf("Keto Load generation for %v with %d writers, %d readers, %d ratio",
		cfg.Duration(), cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)
	writers := keto.BuildWriters(ctx, cfg, client)
	readers := keto.BuildReaders(ctx, cfg, client)

	var wg sync.WaitGroup
	tupleChannel := make(chan *keto.RelationTuple, 100)
	defer close(tupleChannel)
	readerObs := &observer.Observer{
		Name:     "keto-reads",
		Delegate: metrics.OryLatencyHistogram.WithLabelValues("keto", "read"),
	}
	consumerPool := process.ConsumerPool[*keto.RelationTuple]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  cfg.Duration(),
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, tupleChannel)
	writerObs := &observer.Observer{
		Name:     "keto-writes",
		Delegate: metrics.OryLatencyHistogram.WithLabelValues("keto", "write"),
	}
	producerPool := process.ProducerPool[*keto.RelationTuple]{
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
