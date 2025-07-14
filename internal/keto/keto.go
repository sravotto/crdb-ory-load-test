package keto

import (
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/pkg/errors"

	"crdb-ory-load-test/internal/client"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/process"
)

type Keto struct {
	read  *client.Client
	write *client.Client
}

var _ process.Client[*RelationTuple] = &Keto{}

func New(config *config.Config) process.Client[*RelationTuple] {
	return &Keto{
		read:  client.New(config.Keto.ReadAPI),
		write: client.New(config.Keto.WriteAPI),
	}
}

func (k *Keto) BuildReaders(ctx *stopper.Context, cfg *config.Config) ([]process.Consumer[*RelationTuple], error) {
	readers := make([]process.Consumer[*RelationTuple], cfg.Readers())
	for idx := range readers {
		reader := &Reader{
			Client: k,
			Name:   fmt.Sprintf("keto-reader-%d", idx),
		}
		readers[idx] = reader
	}
	return readers, nil
}

func (k *Keto) BuildWriters(ctx *stopper.Context, cfg *config.Config) ([]process.Producer[*RelationTuple], error) {
	writers := make([]process.Producer[*RelationTuple], cfg.Writers())
	for idx := range writers {
		writer := &Writer{
			Client: k,
			Name:   fmt.Sprintf("keto-writer-%d", idx),
		}
		writers[idx] = writer
	}
	return writers, nil
}
func (k *Keto) CheckPermission(ctx *stopper.Context, tuple *RelationTuple) (bool, error) {
	jsonData, err := json.Marshal(tuple)
	if err != nil {
		return false, err
	}
	body, err := k.read.PostJson(ctx, "relation-tuples/check", jsonData)
	if err != nil {
		return false, errors.Wrap(err, "request to relation-tuples/check failed")
	}
	var checkResp CheckResponse
	err = json.Unmarshal(body, &checkResp)
	if err != nil {
		return false, errors.Wrap(err, "invalid tuple")
	}
	return checkResp.Allowed, nil
}

func (k *Keto) HealthCheck(ctx *stopper.Context) error {
	if err := k.read.HealthCheck(ctx); err != nil {
		return err
	}
	if err := k.write.HealthCheck(ctx); err != nil {
		return err
	}
	return nil
}

func (k *Keto) String() string {
	return "keto"
}
func (k *Keto) WriteTuple(ctx *stopper.Context, tuple *RelationTuple) error {
	jsonData, err := json.Marshal(tuple)
	if err != nil {
		return fmt.Errorf("failed to marshal tuple: %w", err)
	}
	_, err = k.write.PutJson(ctx, "admin/relation-tuples", jsonData)
	if err != nil {
		return errors.Wrap(err, "request to admin/relation-tuples failed")
	}
	return nil
}
