package generator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"crdb-ory-load-test/cmd/process"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/hydra"
	"crdb-ory-load-test/internal/metrics"
)

type clientCredentials struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
}

type hydraObserver struct {
	delegate prometheus.Observer
	mu       struct {
		sync.Mutex
		counter int
	}
}

func (o *hydraObserver) Observe(v float64) {
	o.delegate.Observe(v)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.mu.counter++
}

func (o *hydraObserver) Total() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.mu.counter
}

type hydraReader struct {
	hydra            *hydra.Hydra
	name             string
	dryRun           bool
	active, inactive int
}

func (r *hydraReader) Consume(ctx *stopper.Context, c clientCredentials) error {
	active := false
	var err error
	if !r.dryRun {
		active, err = r.hydra.IntrospectToken(ctx, c.AccessToken)
		if err != nil {
			log.Printf("error calling hydra %s", err)
			metrics.ErrorCounter.WithLabelValues("hydra", "introspect", r.name).Inc()
			time.Sleep(100 * time.Millisecond)
		}
		if active {
			r.active++
			metrics.OAuthTokenCheckCounter.WithLabelValues("active").Inc()
		} else {
			r.inactive++
			metrics.OAuthTokenCheckCounter.WithLabelValues("inactive").Inc()
		}
	}
	return nil
}

func (r *hydraReader) String() string {
	return r.name
}

type hydraWriter struct {
	hydra    *hydra.Hydra
	clientID string
	secret   string
	name     string
	dryRun   bool
}

var _ process.Producer[clientCredentials] = &hydraWriter{}

func (w *hydraWriter) Produce(ctx *stopper.Context) (clientCredentials, error) {
	if !w.dryRun {
		for {
			token, err := w.hydra.GrantClientCredentials(ctx, w.clientID, w.secret)
			if err != nil || token == "" {
				log.Printf("error calling hydra %s", err)
				metrics.ErrorCounter.WithLabelValues("hydra", "grant", w.name).Inc()
				time.Sleep(100 * time.Millisecond)
			} else {
				return clientCredentials{
					ClientID:     w.clientID,
					ClientSecret: w.secret,
					AccessToken:  token}, nil
			}
		}
	}
	return clientCredentials{}, nil
}

func (w *hydraWriter) String() string {
	return w.name
}
func RunHydraWorkload(ctx *stopper.Context, dryRun bool) {
	cfg := config.AppConfig
	duration := cfg.Duration()
	gofakeit.Seed(0)

	hydra := hydra.New()
	log.Printf("Hydra Load generation for %v with %d writers, %d readers, %d ratio",
		duration, cfg.Writers(), cfg.Readers(), cfg.Workload.ReadRatio)
	writers := make([]process.Producer[clientCredentials], config.AppConfig.Writers())
	for idx := range writers {
		writer := &hydraWriter{
			hydra:    hydra,
			clientID: uuid.New().String(),
			secret:   gofakeit.Password(true, true, true, true, false, 26),
			name:     fmt.Sprintf("hydra-load-test-client-%d", idx),
		}
		writers[idx] = writer
		if !dryRun {
			created, err := hydra.CreateOAuth2Client(ctx, writer.clientID, writer.name, writer.secret)
			if err != nil || !created {
				panic(err)
			}
			log.Printf("Hydra OAuth2 Client Created with ID: %s", writer.name)
		}
	}
	readers := make([]process.Consumer[clientCredentials], cfg.Readers())
	for idx := range readers {
		readers[idx] = &hydraReader{
			hydra: hydra,
			name:  fmt.Sprintf("reader %d", idx),
		}
	}
	start := time.Now()
	var wg sync.WaitGroup
	credentialsChannel := make(chan clientCredentials, 10)
	defer close(credentialsChannel)
	readerObs := &hydraObserver{
		delegate: metrics.OAuthTokenCheckHistogram.WithLabelValues("reader"),
	}
	consumerPool := process.ConsumerPool[clientCredentials]{
		Consumers: readers,
		Observer:  readerObs,
		Duration:  duration,
		Repeats:   cfg.Workload.ReadRatio,
	}
	consumerPool.Start(ctx, &wg, credentialsChannel)

	writerObs := &hydraObserver{
		delegate: metrics.OAuthTokenCheckHistogram.WithLabelValues("writer"),
	}
	producerPool := process.ProducerPool[clientCredentials]{
		Producers: writers,
		Observer:  writerObs,
		Duration:  duration,
	}
	producerPool.Start(ctx, &wg, credentialsChannel)

	wg.Wait()

	inactive := 0
	for _, r := range readers {
		inactive = inactive + r.(*hydraReader).inactive
	}
	active := 0
	for _, r := range readers {
		//log.Printf("%s %d", r.(*hydraReader).name, r.(*hydraReader).active)
		active = active + r.(*hydraReader).active
	}

	log.Println("Hydra Load generation and access token introspections complete")
	log.Printf("Duration:               %v %v", duration, time.Since(start))
	log.Printf("Concurrency:            %d", cfg.Writers()+cfg.Readers())
	log.Printf("Checks/sec:             %.1f", float64(readerObs.Total())/float64(cfg.Workload.DurationSec))
	log.Printf("Mode:                   %s", map[bool]string{true: "DRY RUN", false: "LIVE"}[dryRun])
	log.Printf("Writes:                 %d", writerObs.Total())
	log.Printf("Reads:                  %d", readerObs.Total())
	log.Printf("Active:                 %d", active)
	log.Printf("Inactive:               %d", inactive)

	if dryRun {
		log.Println("Dry-run mode: No tuples were written to Hydra.")
	}

}
