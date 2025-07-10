package process

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/prometheus/client_golang/prometheus"
)

type Consumer[T any] interface {
	fmt.Stringer
	Consume(*stopper.Context, T) error
}

type ConsumerPool[T any] struct {
	Consumers []Consumer[T]
	Observer  prometheus.Observer
	Duration  time.Duration
	Repeats   int
}

func (c *ConsumerPool[T]) Start(ctx *stopper.Context, wg *sync.WaitGroup, data <-chan T) {

	for _, consumer := range c.Consumers {
		wg.Add(1)
		ok := ctx.Go(func(ctx *stopper.Context) error {
			defer wg.Done()
			ticker := time.NewTicker(c.Duration)
			defer ticker.Stop()
			for {
				select {
				case item := <-data:
					start := time.Now()
					for i := 0; i < c.Repeats; i++ {
						err := consumer.Consume(ctx, item)
						if err != nil {
							return err
						}
						c.Observer.Observe(float64(time.Since(start).Seconds()))
					}
				case <-ticker.C:
					log.Printf("done %s", consumer)
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
