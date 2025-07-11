package generator

import (
	"log"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"

	"crdb-ory-load-test/cmd/process"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/hydra"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
)

func RunHydraWorkload(ctx *stopper.Context, cfg *config.Config) error {
	client := hydra.New(cfg)
	if err := cfg.CheckHydra(); err != nil {
		return err
	}
	if err := client.HealthCheck(ctx); err != nil {
		return err
	}

	log.Printf("Hydra Load generation for %v with %d writers, %d readers, %d ratio",
		cfg.Duration(), cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)

	writers, err := hydra.BuildWriters(ctx, cfg, client)
	if err != nil {
		return err
	}
	readers := hydra.BuildReaders(ctx, cfg, client)

	var wg sync.WaitGroup
	credentialsChannel := make(chan hydra.Credentials, 100)
	defer close(credentialsChannel)
	readerObs := &observer.Observer{
		Delegate: metrics.OryLatencyHistogram.WithLabelValues("hydra", "read"),
	}
	consumerPool := process.ConsumerPool[hydra.Credentials]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  cfg.Duration(),
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, credentialsChannel)
	writerObs := &observer.Observer{
		Delegate: metrics.OryLatencyHistogram.WithLabelValues("hydra", "write"),
	}
	producerPool := process.ProducerPool[hydra.Credentials]{
		Producers: writers,
		Observer:  writerObs,
		Duration:  cfg.Duration(),
	}
	start := time.Now()
	producerPool.Start(ctx, &wg, credentialsChannel)
	wg.Wait()
	printSummary(cfg, start, readerObs, writerObs)
	return nil
}
