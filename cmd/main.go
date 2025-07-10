package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"crdb-ory-load-test/cmd/generator"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/metrics"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

func main() {
	scope := flag.String("scope", "all", "Scope of Workload Simulation (valid values: hydra, kratos, keto, all)")
	duration := flag.Int("duration-sec", 0, "Override duration in seconds")
	readRatio := flag.Int("read-ratio", 0, "Override read/write ratio (e.g. 100 = 100:1)")
	dryRun := flag.Bool("dry-run", false, "Simulate workload without API calls")
	workloadConfig := flag.String("workload-config", "config/config.yaml", "Path to workload config")
	logFile := flag.String("log-file", "", "Path to log output file")
	serveMetrics := flag.Bool("serve-metrics", false, "Keep Prometheus metrics endpoint alive after run")
	verbose := flag.Bool("verbose", true, "Enable verbose logging")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `
📦 crdb-ory-load-test: Workload simulator for Ory + CockroachDB

Usage:
  ./crdb-ory-load-test [flags]

Options:
  -scope               Scope of Workload Simulation (valid values: hydra, kratos, keto. Default: all)
  -checks-per-second   Max permission checks per second (overrides config file)
  -duration-sec        Run for this many seconds (default from config file)
  -read-ratio          Read-to-write ratio (e.g. 100 means 100 reads per 1 write)
  -workload-config     Path to workload config file (default: config/config.yaml)
  -log-file            Path to write logs to (default: stdout only)
  -serve-metrics       Keep Prometheus metrics endpoint alive after run (default: false)
  -dry-run             Skip actual writes and permission checks
  -help                Show this help message

🔒 This tool assumes Ory + CockroachDB Sandbox is deployed and reachable.
📖 See install docs: https://github.com/amineelkouhen/crdb-ory-sandbox/?tab=readme-ov-file#-deployment
`)
	}

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	if err := config.LoadConfig(*workloadConfig); err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	if *duration > 0 {
		config.AppConfig.Workload.DurationSec = *duration
	}
	if *readRatio > 0 {
		config.AppConfig.Workload.ReadRatio = *readRatio
	}

	if *logFile != "" {
		f, err := os.Create(*logFile)
		if err != nil {
			log.Fatalf("❌ Failed to create log file: %v", err)
		}
		defer f.Close()

		if *verbose {
			log.SetOutput(io.MultiWriter(os.Stdout, f))
		} else {
			log.SetOutput(f)
		}
	} else if !*verbose {
		log.SetOutput(io.Discard)
	}

	gracePeriod := 5 * time.Second
	ctx := stopper.WithContext(context.Background())
	// Stop cleanly on interrupt.
	ctx.Go(func(stop *stopper.Context) error {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()
		select {
		case <-ctx.Done():
			log.Print("Interrupted")
			stop.Stop(gracePeriod)
		case <-stop.Stopping():
			// Nothing to do.
		}
		return nil
	})

	switch strings.ToLower(*scope) {
	case "hydra":
		if !*dryRun {
			checkHydra()
		}
		metrics.Init("hydra")
		generator.RunHydraWorkload(ctx, *dryRun)
	case "kratos":
		if !*dryRun {
			checkKratos()
		}
		metrics.Init("kratos")
		generator.RunKratosWorkload(*dryRun)
	case "keto":
		if !*dryRun {
			checkKeto()
		}
		metrics.Init("keto")
		generator.RunKetoWorkload(*dryRun)
	default:
		if !*dryRun {
			checkHydra()
			checkKratos()
			checkKeto()
		}
		metrics.Init("all")
		generator.RunHydraWorkload(ctx, *dryRun)
		generator.RunKratosWorkload(*dryRun)
		generator.RunKetoWorkload(*dryRun)
	}

	if *serveMetrics {
		fmt.Println("📊 Prometheus metrics available at http://localhost:2112/metrics")
		fmt.Println("🔁 Waiting indefinitely for Prometheus to scrape. Ctrl+C to exit.")
		select {}
	}
}

func checkHydra() {
	if config.AppConfig.Hydra.AdminAPI == nil {
		log.Fatalf("❌ Hydra Admin Endpoint is Missing")
		os.Exit(-1)
	}
	if config.AppConfig.Hydra.PublicAPI == nil {
		log.Fatalf("❌ Hydra Public Endpoint is Missing")
		os.Exit(-1)
	}

	healthURL := *config.AppConfig.Hydra.AdminAPI + "/health/alive"
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil || resp.StatusCode != 200 {
		log.Fatalf(`❌ Unable to reach Ory Hydra at %s.

        Make sure Ory Hydra is running and reachable.
        Refer to: https://www.ory.sh/docs/hydra/install

        Details:
        - Error: %v
        - HTTP Status: %v
        `, config.AppConfig.Hydra.AdminAPI, err, resp.StatusCode)
	}
}

func checkKratos() {
	if config.AppConfig.Kratos.AdminAPI == nil {
		log.Fatalf("❌ Kratos Admin Endpoint is Missing")
		os.Exit(-1)
	}
	if config.AppConfig.Kratos.PublicAPI == nil {
		log.Fatalf("❌ Kratos Public Endpoint is Missing")
		os.Exit(-1)
	}

	healthURL := *config.AppConfig.Kratos.AdminAPI + "/health/alive"
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil || resp.StatusCode != 200 {
		log.Fatalf(`❌ Unable to reach Ory Kratos at %s.

        Make sure Ory Kratos is running and reachable.
        Refer to: https://www.ory.sh/docs/kratos/install

        Details:
        - Error: %v
        - HTTP Status: %v
        `, config.AppConfig.Kratos.AdminAPI, err, resp.StatusCode)
	}
}

func checkKeto() {
	if config.AppConfig.Keto.ReadAPI == nil {
		log.Fatalf("❌ Keto Read Endpoint is Missing")
		os.Exit(-1)
	}
	if config.AppConfig.Keto.WriteAPI == nil {
		log.Fatalf("❌ Keto Write Endpoint is Missing")
		os.Exit(-1)
	}

	healthURL := *config.AppConfig.Keto.ReadAPI + "/health/alive"
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil || resp.StatusCode != 200 {
		log.Fatalf(`❌ Unable to reach Ory Keto at %s.

        Make sure Ory Keto is running and reachable.
        Refer to: https://www.ory.sh/docs/keto/install

        Details:
        - Error: %v
        - HTTP Status: %v
        `, config.AppConfig.Keto.ReadAPI, err, resp.StatusCode)
	}
}
