package generator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/google/uuid"

	"crdb-ory-load-test/cmd/process"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/hydra"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
)

func RunHydraWorkload(ctx *stopper.Context, cfg *config.Config) error {
	duration := cfg.Duration()
	gofakeit.Seed(0)

	client := hydra.New(cfg)

	if err := cfg.CheckHydra(); err != nil {
		return err
	}
	if err := client.HealthCheck(ctx); err != nil {
		return err
	}

	log.Printf("Hydra Load generation for %v with %d writers, %d readers, %d ratio",
		duration, cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)
	writers := make([]process.Producer[hydra.Credentials], cfg.Writers())
	for idx := range writers {
		writer := &hydra.Writer{
			Client: client,
			ID:     uuid.New().String(),
			Secret: gofakeit.Password(true, true, true, true, false, 26),
			Name:   fmt.Sprintf("hydra-load-test-client-%d", idx),
		}
		writers[idx] = writer
		created, err := client.CreateOAuth2Client(ctx, writer)
		if err != nil || !created {
			panic(err)
		}
		log.Printf("Hydra OAuth2 Client Created with ID: %s", writer.Name)
	}
	readers := make([]process.Consumer[hydra.Credentials], cfg.Readers())
	for idx := range readers {
		readers[idx] = &hydra.Reader{
			Client: client,
			Name:   fmt.Sprintf("reader %d", idx),
		}
	}

	start := time.Now()
	var wg sync.WaitGroup
	credentialsChannel := make(chan hydra.Credentials, 100)
	defer close(credentialsChannel)
	readerObs := &observer.Observer{
		Delegate: metrics.OAuthTokenCheckHistogram.WithLabelValues("reader"),
	}
	consumerPool := process.ConsumerPool[hydra.Credentials]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  duration,
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, credentialsChannel)
	writerObs := &observer.Observer{
		Delegate: metrics.OAuthTokenCheckHistogram.WithLabelValues("writer"),
	}
	producerPool := process.ProducerPool[hydra.Credentials]{
		Producers: writers,
		Observer:  writerObs,
		Duration:  duration,
	}
	producerPool.Start(ctx, &wg, credentialsChannel)
	wg.Wait()

	inactive := 0
	for _, r := range readers {
		inactive = inactive + r.(*hydra.Reader).Inactive
	}
	active := 0
	for _, r := range readers {
		active = active + r.(*hydra.Reader).Active
	}
	log.Println("Hydra Load generation and access token introspections complete")
	log.Printf("Requested Duration:     %s", duration)
	log.Printf("Effective Duration:     %s", time.Since(start))
	log.Printf("Concurrency:            %d", cfg.Writers()+cfg.Readers())
	log.Printf("Checks/sec:             %.2f", float64(readerObs.Total())/float64(cfg.Workload.DurationSec))
	log.Printf("Writes:                 %d", writerObs.Total())
	log.Printf("Reads:                  %d", readerObs.Total())
	log.Printf("Active:                 %d", active)
	log.Printf("Inactive:               %d", inactive)
	return nil
}
