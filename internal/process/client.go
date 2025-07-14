package process

import (
	"crdb-ory-load-test/internal/config"
	"fmt"

	"github.com/cockroachdb/field-eng-powertools/stopper"
)

type Client[T any] interface {
	fmt.Stringer
	HealthChecker
	Builder[T]
}

type Builder[T any] interface {
	BuildReaders(*stopper.Context, *config.Config) ([]Consumer[T], error)
	BuildWriters(*stopper.Context, *config.Config) ([]Producer[T], error)
}

type HealthChecker interface {
	HealthCheck(ctx *stopper.Context) error
}
