package keto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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

func CheckPermission(namespace, object, relation, subjectID string) (bool, error) {
	reqBody := CheckRequest{
		Namespace: namespace,
		Object:    object,
		Relation:  relation,
		SubjectID: subjectID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("❌ Error marshaling check request: %v\n", err)
		return false, err
	}

	url := config.AppConfig.Keto.ReadAPI + "/relation-tuples/check"
	client := &http.Client{Timeout: 5 * time.Second}

	var resp *http.Response
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if attempt < 3 {
			fmt.Printf("🔁 Retry %d: Keto check failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil || resp == nil {
		fmt.Printf("❌ Final failure: Keto check failed after 3 attempts. Error: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️  Unexpected status from Keto: %d\nResponse body: %s\n", resp.StatusCode, string(body))
		return false, errors.New("⚠️  Unexpected status from Keto")
	}

	var checkResp CheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		fmt.Printf("❌ Error decoding Keto check response: %v\n", err)
		return false, err
	}

	return checkResp.Allowed, nil
}

func WriteTuple(namespace, object, relation, subjectID string) error {
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

	url := config.AppConfig.Keto.WriteAPI + "/admin/relation-tuples"
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT failed: status=%v body=%s", resp.StatusCode, string(body))
	}

	fmt.Printf("🔑  Permission %s granted to %s for object %s\n", tuple.Relation, tuple.SubjectID, tuple.Object)
	return nil
}

func getStatus(resp *http.Response) int {
	if resp != nil {
		return resp.StatusCode
	}
	return 0
}
