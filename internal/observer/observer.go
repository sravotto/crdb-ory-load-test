package observer

import (
	"sync"

	"github.com/DataDog/sketches-go/ddsketch"

	"github.com/prometheus/client_golang/prometheus"
)

type Observer interface {
	GetValuesAtQuantiles([]float64) ([]float64, error)
	Mean() float64
	Name() string
	Observe(float64)
	Total() int
}
type observer struct {
	delegate prometheus.Observer
	name     string
	mu       struct {
		sync.Mutex
		sketch *ddsketch.DDSketch
	}
}

func NewObserver(name string, obs prometheus.Observer) (Observer, error) {
	relativeAccuracy := 0.001
	sketch, err := ddsketch.NewDefaultDDSketch(relativeAccuracy)
	if err != nil {
		return nil, err
	}
	res := &observer{
		delegate: obs,
		name:     name,
	}
	res.mu.sketch = sketch
	return res, nil
}

func (o *observer) GetValuesAtQuantiles(qs []float64) ([]float64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.mu.sketch.GetValuesAtQuantiles(qs)
}

func (o *observer) Mean() float64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.mu.sketch.GetSum() / o.mu.sketch.GetCount()
}

func (o *observer) Name() string {
	return o.name
}

func (o *observer) Observe(v float64) {
	o.delegate.Observe(v)
	o.mu.Lock()
	defer o.mu.Unlock()
	o.mu.sketch.Add(v)
}

func (o *observer) Total() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return int(o.mu.sketch.GetCount())
}
