package generator

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/kratos"
	"crdb-ory-load-test/internal/metrics"
	"log"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type identity struct {
	Email     string
	FirstName string
	LastName  string
}

func RunKratosWorkload(ctx *stopper.Context, config *config.Config) error {
	if err := config.CheckKratos(); err != nil {
		return err
	}
	client := kratos.New(config)

	if err := client.HealthCheck(ctx); err != nil {
		return err
	}

	cfg := config.Workload
	duration := time.Duration(cfg.DurationSec) * time.Second
	endTime := time.Now().Add(duration)
	gofakeit.Seed(0)

	writeWorkers := 1
	readWorkers := cfg.ReadRatio
	totalWorkers := writeWorkers + readWorkers

	var wg sync.WaitGroup
	identityChannel := make(chan identity, 10000)

	var activeIdentityCount, inactiveIdentityCount, failedReads, failedWrites, readCount, writeCount int64

	log.Printf("🚧 Kratos Load generation for %v with %d total workers (%d writers, %d readers)...",
		duration, totalWorkers, writeWorkers, readWorkers)

	// Phase 1: Start write worker(s)
	for i := 0; i < writeWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(endTime) {
				email := gofakeit.Email()
				firstName := gofakeit.FirstName()
				lastName := gofakeit.LastName()
				password := gofakeit.Password(true, true, true, true, false, 8)

				created, err := client.RegisterIdentity(ctx, email, firstName, lastName, password)
				if err != nil || !created {
					log.Printf("❌ Write Identity failed: %v", err)
					failedWrites++
				} else {
					// Push the same identity read_ratio times
					for j := 0; j < cfg.ReadRatio; j++ {
						identityChannel <- identity{Email: email, FirstName: firstName, LastName: lastName}
					}
					writeCount++
				}

			}
		}(i)
	}

	// Phase 2: Start read workers
	for i := 0; i < readWorkers; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for time.Now().Before(endTime) {
				select {
				case t := <-identityChannel:
					active := false
					var err error

					active, err = client.CheckIdentity(ctx, t.Email)
					if active {
						log.Printf("🔒 Identity check result: email=%s, firstName=%s, lastName=%s, active=%v", t.Email, t.FirstName, t.LastName, active)
					} else if err != nil {
						failedReads++
					}

					if active {
						metrics.IdentityCheckCounter.WithLabelValues("active").Inc()
						activeIdentityCount++
					}
					if !active && err == nil {
						metrics.IdentityCheckCounter.WithLabelValues("inactive").Inc()
						inactiveIdentityCount++
					}
					readCount++
				default:
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	log.Println("🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧")
	log.Println("✅  Kratos Load generation and identity checks complete")
	log.Printf("⏱️  Duration:                %v", duration)
	log.Printf("⚙️  Concurrency:             %d", totalWorkers)
	log.Printf("🚦 Checks/sec:              %.1f", float64(readCount)/float64(cfg.DurationSec))
	log.Printf("🟢 Active:                  %d", activeIdentityCount)
	log.Printf("🔴 Inactive:                %d", inactiveIdentityCount)
	log.Printf("✏️  Writes:                  %d", writeCount)
	log.Printf("👁️  Reads:                   %d", readCount)
	if writeCount > 0 {
		log.Printf("📊 Read/Write ratio:        %.1f:1", float64(readCount)/float64(writeCount))
	}
	log.Printf("🚨 Failed writes to Kratos: %d", failedWrites)
	log.Printf("🚨 Failed reads to Kratos:  %d", failedReads)

	log.Println("🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧")
	return nil
}
