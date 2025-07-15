package metrics

import (
	"log"
	"math"
	"net/http"
	_ "net/http/pprof" // enabling debugging

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	MessageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ory_message_counter",
			Help: "Total errors contacting ory services",
		},
		[]string{"service", "operation", "process_id"},
	)

	ErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ory_error_counter",
			Help: "Total errors contacting ory services",
		},
		[]string{"service", "operation", "process_id"},
	)

	OryLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ory_request_sec",
			Help:    "the length of time spent in a ory request",
			Buckets: buckets(0.01, 10),
		},
		[]string{"service", "operation"},
	)
)

func buckets(base, max float64) []float64 {
	var ret []float64
	for {
		for i := 0; i < 9; i++ {
			next := math.FMA(float64(i), base, base)
			if next > max {
				return ret
			}
			next = math.Round(next*1000) / 1000
			ret = append(ret, next)
		}
		base *= 10
	}
}

func Init() {
	prometheus.MustRegister(ErrorCounter)
	prometheus.MustRegister(OryLatencyHistogram)
	prometheus.MustRegister(MessageCounter)
	// Health and metrics endpoints
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.Handle("/", promhttp.Handler())
	go func() {
		log.Println("Starting metrics HTTP server on :26260")
		if err := http.ListenAndServe("0.0.0.0:26260", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()
}
