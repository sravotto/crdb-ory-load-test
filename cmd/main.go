package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
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
	workloadConfig := flag.String("workload-config", "config/config.yaml", "Path to workload config")
	logFile := flag.String("log-file", "", "Path to log output file")
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
  -help                Show this help message

This tool assumes Ory + CockroachDB Sandbox is deployed and reachable.
See install docs: https://github.com/amineelkouhen/crdb-ory-sandbox/?tab=readme-ov-file#-deployment
`)
	}

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	config, err := config.LoadConfig(*workloadConfig)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *duration > 0 {
		config.Workload.DurationSec = *duration
	}
	if *readRatio > 0 {
		config.Workload.ReadRatio = *readRatio
	}

	if *logFile != "" {
		f, err := os.Create(*logFile)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
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
	metrics.Init()
	switch strings.ToLower(*scope) {
	case "hydra":
		err = generator.RunHydraWorkload(ctx, config)
	case "kratos":
		err = generator.RunKratosWorkload(ctx, config)
	case "keto":
		err = generator.RunKetoWorkload(ctx, config)
	default:
		panic("scope not implemented")
	}
	if err != nil {
		log.Fatalf("workload failed %s", err)
	}
}
