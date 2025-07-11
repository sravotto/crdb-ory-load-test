package observer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Observer struct {
	Delegate prometheus.Observer
	mu       struct {
		sync.Mutex
		counter int
	}
}

func (o *Observer) Observe(v float64) {
	o.Delegate.Observe(v)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.mu.counter++
}

func (o *Observer) Total() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.mu.counter
}
