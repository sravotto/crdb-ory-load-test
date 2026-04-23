package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
)

type Consumer[T any] interface {
	fmt.Stringer
	Consume(*stopper.Context, T) error
}

type ConsumerPool[T any] struct {
	Consumers      []Consumer[T]
	Observer       prometheus.Observer
	Duration       time.Duration
	Repeats        int
	TolerateErrors bool
	Limiter        *rate.Limiter
}

func (c *ConsumerPool[T]) Start(ctx *stopper.Context, wg *sync.WaitGroup, data <-chan T) {

	for _, consumer := range c.Consumers {
		wg.Add(1)
		ok := ctx.Go(func(ctx *stopper.Context) error {
			defer wg.Done()
			timeout := time.NewTicker(c.Duration)
			defer timeout.Stop()

			// Create a context that cancels when Stopping() fires,
			// so rate.Limiter.Wait unblocks promptly on shutdown.
			waitCtx, waitCancel := context.WithCancel(context.Background())
			go func() {
				select {
				case <-ctx.Stopping():
					waitCancel()
				case <-waitCtx.Done():
				}
			}()
			defer waitCancel()

			for {
				select {
				case item, ok := <-data:
					if !ok {
						return nil
					}
					start := time.Now()
					for range c.Repeats {
						if c.Limiter != nil {
							if err := c.Limiter.Wait(waitCtx); err != nil {
								return nil
							}
						}
						err := consumer.Consume(ctx, item)
						if err != nil {
							if c.TolerateErrors {
								break
							}
							return err
						}
						c.Observer.Observe(float64(time.Since(start).Seconds()))
					}
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
