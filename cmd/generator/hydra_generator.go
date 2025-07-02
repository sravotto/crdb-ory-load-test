package generator

import (
	"log"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"

	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/hydra"
	"crdb-ory-load-test/internal/metrics"
)

type clientCredentials struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
}

type adminCredentials struct {
	ID     string
	Secret string
	Name   string
}

var clients []adminCredentials

func RunHydraWorkload(dryRun bool) {
	cfg := config.AppConfig.Workload
	duration := time.Duration(cfg.DurationSec) * time.Second
	endTime := time.Now().Add(duration)
	gofakeit.Seed(0)

	writeWorkers := 2
	readWorkers := cfg.ReadRatio
	totalWorkers := writeWorkers + readWorkers

	clients = make([]adminCredentials, writeWorkers)
	for i := 0; i < writeWorkers; i++ {
		clients[i].ID = uuid.New().String()
		clients[i].Name = "hydra-load-test-client"
		clients[i].Secret = gofakeit.Password(true, true, true, true, false, 26)

		log.Printf("🚧 Hydra Load generation for %v with %d total workers (%d writers, %d readers)...",
			duration, totalWorkers, writeWorkers, readWorkers)

		if !dryRun {
			created, err := hydra.CreateOAuth2Client(clients[i].ID, clients[i].Name, clients[i].Secret)
			if err != nil || !created {
				log.Printf("❌ OAuth2 client creation failed: %v", err)
				return
			}
			log.Printf("🏛️ Hydra OAuth2 Client Created with ID: %s", clients[i].ID)
		}
	}

	var wg sync.WaitGroup
	credentialsChannel := make(chan clientCredentials, 10000)

	var activeTokenCount, inactiveTokenCount, failedReads, failedWrites, readCount, writeCount int64

	// Phase 1: Start write worker(s)
	for i := 0; i < writeWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(endTime) {
				if !dryRun {
					token, err := hydra.GrantClientCredentials(clients[i].ID, clients[i].Secret)
					if err != nil || token == "" {
						log.Printf("❌  Client Credentials Grant failed: %v", err)
						failedWrites++
					} else {
						//log.Printf("🎟️  Access Token generated for Client %s %s", clients[i].ID, token)
						// Push the same identity read_ratio times
						for j := 0; j < cfg.ReadRatio; j++ {
							credentialsChannel <- clientCredentials{ClientID: clients[i].ID, ClientSecret: clients[i].Secret, AccessToken: token}
						}
						writeCount++
					}
				}
			}
		}(i)
	}

	// Phase 2: Start read workers
	for i := 0; i < readWorkers; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for time.Now().Before(endTime) {
				select {
				case t := <-credentialsChannel:
					active := false
					var err error
					if !dryRun {
						active, err = hydra.IntrospectToken(t.AccessToken)
						if active {
							//log.Printf("👀 Token introspection: Access Token for client %s is Active=%v", t.ClientID, active)
						} else if err != nil {
							failedReads++
						}
					}

					if active {
						metrics.OAuthTokenCheckCounter.WithLabelValues("active").Inc()
						activeTokenCount++
					}
					if !active && err == nil {
						metrics.OAuthTokenCheckCounter.WithLabelValues("inactive").Inc()
						inactiveTokenCount++
					}
					readCount++
				default:
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	log.Println("🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧")
	log.Println("✅  Hydra Load generation and access token introspections complete")
	log.Printf("⏱️  Duration:               %v", duration)
	log.Printf("⚙️  Concurrency:            %d", totalWorkers)
	log.Printf("🚦 Checks/sec:             %.1f", float64(readCount)/float64(cfg.DurationSec))
	log.Printf("🧪 Mode:                   %s", map[bool]string{true: "DRY RUN", false: "LIVE"}[dryRun])
	log.Printf("🟢 Active:                 %d", activeTokenCount)
	log.Printf("🔴 Inactive:               %d", inactiveTokenCount)
	log.Printf("✏️  Writes:                 %d", writeCount)
	log.Printf("👁️  Reads:                  %d", readCount)
	if writeCount > 0 {
		log.Printf("📊 Read/Write ratio:       %.1f:1", float64(readCount)/float64(writeCount))
	}
	log.Printf("🚨 Failed writes to Hydra: %d", failedWrites)
	log.Printf("🚨 Failed reads to Hydra:  %d", failedReads)

	if dryRun {
		log.Println("⚠️  Dry-run mode: No tuples were written to Hydra.")
	}

	log.Println("🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧🚧")
}
