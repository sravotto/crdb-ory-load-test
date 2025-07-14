package workload

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
		quantiles := []float64{.50, .90, .95, .99}
		n := 0
		for {
			if n%40 == 0 {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("%16s: %8s", "operation", "count"))
				for _, q := range quantiles {
					sb.WriteString(fmt.Sprintf(", p%-2d ms", int(q*100)))
				}
				log.Printf("%s", sb.String())
			}
			select {
			case <-ctx.Stopping():
				return nil
			case <-ticker.C:
				n++
				for _, obs := range observers {
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("%16s: %8d", obs.Name, obs.Total()))
					values, _ := obs.GetValuesAtQuantiles(quantiles)
					for _, v := range values {
						sb.WriteString(fmt.Sprintf(", %6d", int(v*1000)))
					}
					log.Printf("%s", sb.String())
				}

			}
		}
	})
}
