package process

import (
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/prometheus/client_golang/prometheus"
)

type Producer[T any] interface {
	fmt.Stringer
	Produce(*stopper.Context) (T, error)
}

type ProducerPool[T any] struct {
	Producers []Producer[T]
	Observer  prometheus.Observer
	Duration  time.Duration
}

func (p *ProducerPool[T]) Start(ctx *stopper.Context, wg *sync.WaitGroup, data chan<- T) {
	for _, producer := range p.Producers {
		wg.Add(1)
		ok := ctx.Go(func(ctx *stopper.Context) error {
			defer wg.Done()
			ticker := time.NewTicker(p.Duration)
			defer ticker.Stop()
			for {
				start := time.Now()
				item, err := producer.Produce(ctx)
				if err != nil {
					return err
				}
				p.Observer.Observe(float64(time.Since(start).Seconds()))
				select {
				case data <- item:
				case <-ticker.C:
					return nil
				case <-ctx.Stopping():
					return ctx.Err()
				}
			}
		})
		if !ok {
			wg.Done()
		}
	}
}
