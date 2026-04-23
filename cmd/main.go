package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"

	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/hydra"
	"crdb-ory-load-test/internal/keto"
	"crdb-ory-load-test/internal/kratos"
	"crdb-ory-load-test/internal/metrics"
	"crdb-ory-load-test/internal/workload"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

func main() {
	scope := flag.String("scope", "hydra", "Scope of Workload Simulation (valid values: hydra, kratos, keto)")
	duration := flag.Int("duration-sec", 0, "Override duration in seconds")
	flag.IntVar(duration, "duration", 0, "Override duration in seconds (synonym for -duration-sec)")
	readRatio := flag.Int("read-ratio", 0, "Override read/write ratio (e.g. 100 = 100:1)")
	workloadConfig := flag.String("workload-config", "config/config.yaml", "Path to workload config")
	logFile := flag.String("log-file", "", "Path to log output file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	tolerateErrors := flag.Bool("tolerate-errors", false, "Continue in case of errors")
	maxRate := flag.Int("max-rate", 0, "Max requests per second across all workers (0 = unlimited)")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `
crdb-ory-load-test: Workload simulator for Ory + CockroachDB

Usage:
  ./crdb-ory-load-test [flags]

Options:
  -scope               Scope of Workload Simulation (valid values: hydra, kratos, keto)
  -max-rate            Max requests per second across all workers (0 = unlimited)
  -duration-sec        Run for this many seconds (default from config file)
  -duration            Synonym for -duration-sec
  -read-ratio          Read-to-write ratio (e.g. 100 means 100 reads per 1 write)
  -workload-config     Path to workload config file (default: config/config.yaml)
  -log-file            Path to write logs to (default: stdout only)
  -tolerate-errors     Continue in case of errors
  -help                Show this help message

`)
	}

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	config, err := config.LoadConfig(*workloadConfig)
	if err != nil {
		slog.Error("workload failed", err)
		return
	}

	if *duration > 0 {
		config.Workload.DurationSec = *duration
	}
	if *readRatio > 0 {
		config.Workload.ReadRatio = *readRatio
	}

	if !*verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	if *maxRate > 0 {
		config.Workload.MaxRate = *maxRate
	}
	config.Workload.TolerateErrors = *tolerateErrors
	if *logFile != "" {
		f, err := os.Create(*logFile)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, f))
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
			slog.Info("Interrupted")
			stop.Stop(gracePeriod)
		case <-stop.Stopping():
			// Nothing to do.
		}
		return nil
	})
	metrics.Init()
	switch strings.ToLower(*scope) {
	case "hydra":
		if err = config.CheckHydra(); err == nil {
			client := hydra.New(config)
			err = workload.Run(ctx, config, client)
		}

	case "keto":
		if err = config.CheckKeto(); err == nil {
			client := keto.New(config)
			err = workload.Run(ctx, config, client)
		}
	case "kratos":
		if err = config.CheckKratos(); err == nil {
			client := kratos.New(config)
			err = workload.Run(ctx, config, client)
		}
	default:
		err = errors.Newf("scope not implemented %s", *scope)
	}
	if err != nil {
		slog.Error("workload failed", err)
	}
}
