package workload

import (
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/observer"
	"fmt"
	"log"

	"strings"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var processFrequency = 10 * time.Second

func printSummary(cfg *config.Config, start time.Time, readerObs, writerObs observer.Observer) {
	duration := time.Since(start)
	fmt.Printf("\n\n")
	defer fmt.Printf("\n")
	fmt.Printf("Summary\n")
	fmt.Printf("\n")
	fmt.Printf("%-30s %s\n", "Requested Duration:", cfg.Duration())
	fmt.Printf("%-30s %s\n", "Effective Duration:", duration)
	fmt.Printf("%-30s %d\n", "Concurrency:", cfg.Writers()+cfg.Readers())
	fmt.Printf("%-30s %d\n", "Reads:", readerObs.Total())
	fmt.Printf("%-30s %.0f\n", "Read avg ms:", readerObs.Mean()*1000)
	fmt.Printf("%-30s %d\n", "Writes:", writerObs.Total())
	fmt.Printf("%-30s %.0f\n", "Write avg ms:", writerObs.Mean()*1000)

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		fmt.Printf("Unable to retrieve metrics\n")
		return
	}
	for _, family := range families {
		if strings.HasPrefix(family.GetName(), "ory") && family.GetType() == dto.MetricType_COUNTER {
			ops := make(map[string]float64)
			for _, metric := range family.GetMetric() {
				op := ""
				for _, l := range metric.Label {
					if *l.Name == "operation" {
						op = *l.Value
					}
				}
				if op != "" {
					ops[op] = ops[op] + metric.Counter.GetValue()
				}
			}
			for k, v := range ops {
				if family.GetName() == metrics.ErrorCounterName {
					k = k + "_errors"
				} else {
					k = k + "_total"
				}
				fmt.Printf("%-30s %.0f\n", k+":", v)
			}
		}
	}
}

func printProgress(ctx *stopper.Context, observers []observer.Observer) {
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
					sb.WriteString(fmt.Sprintf("%16s: %8d", obs.Name(), obs.Total()))
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
