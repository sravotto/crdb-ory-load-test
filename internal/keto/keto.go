package keto

import (
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/pkg/errors"

	"crdb-ory-load-test/internal/client"
	"crdb-ory-load-test/internal/config"
)

type CheckRequest struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	SubjectID string `json:"subject_id"`
}

type CheckResponse struct {
	Allowed bool `json:"allowed"`
}

type RelationTuple struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	SubjectID string `json:"subject_id"`
}

type Keto struct {
	read  *client.Client
	write *client.Client
}

func New(config *config.Config) *Keto {
	return &Keto{
		read:  client.New(config.Keto.ReadAPI),
		write: client.New(config.Keto.WriteAPI),
	}
}

func (k *Keto) CheckPermission(ctx *stopper.Context, namespace, object, relation, subjectID string) (bool, error) {
	reqBody := CheckRequest{
		Namespace: namespace,
		Object:    object,
		Relation:  relation,
		SubjectID: subjectID,
	}
	jsonData, err := json.Marshal(reqBody)
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
		return false, errors.Wrap(err, "invalid token")
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

func (k *Keto) WriteTuple(ctx *stopper.Context, namespace, object, relation, subjectID string) error {
	tuple := RelationTuple{
		Namespace: namespace,
		Object:    object,
		Relation:  relation,
		SubjectID: subjectID,
	}

	jsonData, err := json.Marshal(tuple)
	if err != nil {
		return fmt.Errorf("failed to marshal tuple: %w", err)
	}

	_, err = k.write.PostJson(ctx, "admin/relation-tuples", jsonData)
	if err != nil {
		return errors.Wrap(err, "request to relation-tuples/check failed")
	}
	return nil
}
