package observer

import (
	"sync"

	"github.com/DataDog/sketches-go/ddsketch"

	"github.com/prometheus/client_golang/prometheus"
)

type Observer struct {
	Delegate prometheus.Observer
	Name     string
	mu       struct {
		sync.Mutex
		counter int
		sketch  *ddsketch.DDSketch
	}
}

func NewObserver(name string, obs prometheus.Observer) (*Observer, error) {
	relativeAccuracy := 0.001
	sketch, err := ddsketch.NewDefaultDDSketch(relativeAccuracy)
	if err != nil {
		return nil, err
	}
	res := &Observer{
		Delegate: obs,
		Name:     name,
	}
	res.mu.sketch = sketch
	return res, nil
}

func (o *Observer) Observe(v float64) {
	o.Delegate.Observe(v)
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mu.sketch != nil {
		o.mu.sketch.Add(v)
	}
	o.mu.counter++
}

func (o *Observer) Total() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.mu.counter
}

func (o *Observer) GetValuesAtQuantiles(qs []float64) ([]float64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mu.sketch == nil {
		return nil, nil
	}
	return o.mu.sketch.GetValuesAtQuantiles(qs)
}
