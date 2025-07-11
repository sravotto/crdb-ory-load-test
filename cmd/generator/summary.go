package generator

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/observer"
	"log"
	"time"
)

func printSummary(cfg *config.Config, start time.Time, readerObs, writerObs *observer.Observer) {
	log.Printf("Requested Duration:     %s", cfg.Duration())
	log.Printf("Effective Duration:     %s", time.Since(start))
	log.Printf("Concurrency:            %d", cfg.Writers()+cfg.Readers())
	log.Printf("Checks/sec:             %.2f", float64(readerObs.Total())/float64(cfg.Workload.DurationSec))
	log.Printf("Writes:                 %d", writerObs.Total())
	log.Printf("Reads:                  %d", readerObs.Total())
}
