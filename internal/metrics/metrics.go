package metrics

import (
	"log"
	"net/http"
	_ "net/http/pprof" // enabling debugging

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	OAuthTokenCheckCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ory_token_check_total",
			Help: "Total oauth token checks run",
		},
		[]string{"result"},
	)

	PermissionCheckCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ory_permission_check_total",
			Help: "Total permission checks run",
		},
		[]string{"result"},
	)

	IdentityCheckCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ory_identity_check_total",
			Help: "Total identity checks run",
		},
		[]string{"result"},
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
			Name: "ory_request_sec",
			Help: "the length of time spent in a ory request",
		},
		[]string{"service", "operation"},
	)
)

func Init() {
	prometheus.MustRegister(ErrorCounter)
	prometheus.MustRegister(OryLatencyHistogram)
	prometheus.MustRegister(OAuthTokenCheckCounter)
	prometheus.MustRegister(IdentityCheckCounter)
	prometheus.MustRegister(PermissionCheckCounter)
	// Health and metrics endpoints
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("Starting metrics HTTP server on :2112")
		if err := http.ListenAndServe("0.0.0.0:2112", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()
}
