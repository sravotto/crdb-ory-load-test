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

	OAuthTokenCheckHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ory_token_auth_request_sec",
			Help: "the length of time spent in a token authorization request",
		},
		[]string{"operation"},
	)
)

func Init(scope string) {

	switch scope {
	case "hydra":
		// Metrics from Hydra
		prometheus.MustRegister(ErrorCounter)
		prometheus.MustRegister(OAuthTokenCheckCounter)
		prometheus.MustRegister(OAuthTokenCheckHistogram)
	case "kratos":
		// Metrics from Kratos
		prometheus.MustRegister(IdentityCheckCounter)
	case "keto":
		// Metrics from Keto
		prometheus.MustRegister(PermissionCheckCounter)
	default:
		// Metrics from all
		prometheus.MustRegister(OAuthTokenCheckCounter)
		prometheus.MustRegister(IdentityCheckCounter)
		prometheus.MustRegister(PermissionCheckCounter)
	}

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
