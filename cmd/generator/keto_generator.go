package generator

import (
	"log"
	"sync"
	"time"

	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/keto"
	"crdb-ory-load-test/internal/metrics"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/google/uuid"
)

type tuple struct {
	Subject string
	Object  string
}

func RunKetoWorkload(ctx *stopper.Context, config *config.Config) error {
	if err := config.CheckKeto(); err != nil {
		return err
	}
	client := keto.New(config)
	if err := client.HealthCheck(ctx); err != nil {
		return err
	}
	cfg := config.Workload

	duration := time.Duration(cfg.DurationSec) * time.Second
	endTime := time.Now().Add(duration)

	writeWorkers := 1
	readWorkers := cfg.ReadRatio
	totalWorkers := writeWorkers + readWorkers

	var wg sync.WaitGroup
	tupleChannel := make(chan tuple, 10000)

	var allowedCount, deniedCount, failedReads, failedWrites, readCount, writeCount int64

	log.Printf("🚧 Keto Load generation for %v with %d total workers (%d writers, %d readers)...",
		duration, totalWorkers, writeWorkers, readWorkers)

	// Phase 1: Start write worker(s)
	for i := 0; i < writeWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(endTime) {
				objectID := uuid.New().String()
				subjectID := uuid.New().String()
				subjectFull := "user:" + subjectID

				err := client.WriteTuple(ctx, "documents", objectID, "viewer", subjectFull)
				if err != nil {
					log.Printf("❌ WriteTuple failed: %v", err)
					failedWrites++
				} else {
					// Push the same tuple read_ratio times
					for j := 0; j < cfg.ReadRatio; j++ {
						tupleChannel <- tuple{Subject: subjectFull, Object: objectID}
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
				case t := <-tupleChannel:
					allowed := false
					var err error

					allowed, err = client.CheckPermission(ctx, "documents", t.Object, "viewer", t.Subject)
					if allowed {
						log.Printf("🔒 Permission check result: subject=%s, object=%s, allowed=%v", t.Subject, t.Object, allowed)
					} else if err != nil {
						failedReads++
					}

					if allowed {
						metrics.PermissionCheckCounter.WithLabelValues("allowed").Inc()
						allowedCount++
					}
					if !allowed && err == nil {
						metrics.PermissionCheckCounter.WithLabelValues("denied").Inc()
						deniedCount++
					}
					readCount++
				default:
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	log.Println("🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧")
	log.Println("✅  Keto Load generation and permission checks complete")
	log.Printf("⏱️  Duration:              %v", duration)
	log.Printf("⚙️  Concurrency:           %d", totalWorkers)
	log.Printf("🚦 Checks/sec:            %.1f", float64(readCount)/float64(cfg.DurationSec))
	log.Printf("✔️  Allowed:               %d", allowedCount)
	log.Printf("🚫 Denied:                %d", deniedCount)
	log.Printf("✏️  Writes:                %d", writeCount)
	log.Printf("👁️  Reads:                 %d", readCount)
	if writeCount > 0 {
		log.Printf("📊 Read/Write ratio:      %.1f:1", float64(readCount)/float64(writeCount))
	}
	log.Printf("🚨 Failed writes to Keto: %d", failedWrites)
	log.Printf("🚨 Failed reads to Keto:  %d", failedReads)

	log.Println("🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧")
	return nil
}
