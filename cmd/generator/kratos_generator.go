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

type identity struct {
	Email     string
	FirstName string
	LastName  string
}

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
	readerObs := &observer.Observer{
		Name:     "kratos-reads",
		Delegate: metrics.OryLatencyHistogram.WithLabelValues("kratos", "read"),
	}
	consumerPool := process.ConsumerPool[*kratos.Identity]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  cfg.Duration(),
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, tupleChannel)
	writerObs := &observer.Observer{
		Name:     "kratos-writs",
		Delegate: metrics.OryLatencyHistogram.WithLabelValues("kratos", "write"),
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
