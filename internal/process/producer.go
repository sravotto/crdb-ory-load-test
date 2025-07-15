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
	Producers      []Producer[T]
	Observer       prometheus.Observer
	Duration       time.Duration
	TolerateErrors bool
}

func (p *ProducerPool[T]) Start(ctx *stopper.Context, wg *sync.WaitGroup, data chan<- T) {
	for _, producer := range p.Producers {
		wg.Add(1)
		ok := ctx.Go(func(ctx *stopper.Context) error {
			defer wg.Done()
			timeout := time.NewTicker(p.Duration)
			defer timeout.Stop()
			delay := time.NewTicker(100 * time.Millisecond)
			defer delay.Stop()
			for {
				start := time.Now()
				item, err := producer.Produce(ctx)
				if err != nil {
					if p.TolerateErrors {
						select {
						case <-timeout.C:
							return nil
						case <-delay.C:
							continue
						case <-ctx.Stopping():
							return ctx.Err()
						}
					}
					return err
				}
				p.Observer.Observe(float64(time.Since(start).Seconds()))
				select {
				case data <- item:
				case <-timeout.C:
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
