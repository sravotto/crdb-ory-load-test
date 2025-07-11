package generator

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/observer"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

var processFrequency = 10 * time.Second

func printSummary(cfg *config.Config, start time.Time, readerObs, writerObs *observer.Observer) {
	log.Printf("")
	log.Printf("Summary")
	log.Printf("")
	log.Printf("Requested Duration:     %s", cfg.Duration())
	log.Printf("Effective Duration:     %s", time.Since(start))
	log.Printf("Concurrency:            %d", cfg.Writers()+cfg.Readers())
	log.Printf("Checks/sec:             %.2f", float64(readerObs.Total())/float64(cfg.Workload.DurationSec))
	log.Printf("Writes:                 %d", writerObs.Total())
	log.Printf("Reads:                  %d", readerObs.Total())
}

func printProgress(ctx *stopper.Context, observers []*observer.Observer) {
	ticker := time.NewTicker(processFrequency)
	ctx.Go(func(ctx *stopper.Context) error {
		for {
			select {
			case <-ctx.Stopping():
				return nil
			case <-ticker.C:
				var sb strings.Builder
				for idx, obs := range observers {
					if idx > 0 {
						sb.WriteString("; ")
					}
					sb.WriteString(fmt.Sprintf("%s=%d", obs.Name, obs.Total()))
				}
				log.Printf("%s", sb.String())
			}
		}
	})
}
