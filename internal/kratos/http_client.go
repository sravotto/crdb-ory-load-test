package kratos

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

type RegistrationRequest struct {
	Method   string `json:"method"`
	Password string `json:"password"`
	Traits   struct {
		Email string `json:"email"`
		Name  struct {
			First string `json:"first"`
			Last  string `json:"last"`
		} `json:"name"`
	} `json:"traits"`
}

var registrationFlowResponse map[string]interface{}

type RegistrationResponse struct {
	Continue string `json:"continue_with"`
	Identity struct {
		Identifier     string `json:"id"`
		SchemaID       string `json:"schema_id"`
		SchemaURL      string `json:"schema_url"`
		State          string `json:"state"`
		StateChangedAt string `json:"state_changed_at"`
		Traits         struct {
			Email string `json:"email"`
			Name  struct {
				First string `json:"first"`
				Last  string `json:"last"`
			} `json:"name"`
		} `json:"traits"`
		MetadataPublic string    `json:"metadata_public"`
		OrganizationID string    `json:"organization_id"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	} `json:"identity"`
}

type CheckIdentityResponse struct {
	Identifier     string `json:"id"`
	SchemaID       string `json:"schema_id"`
	SchemaURL      string `json:"schema_url"`
	State          string `json:"state"`
	StateChangedAt string `json:"state_changed_at"`
	Traits         struct {
		Email string `json:"email"`
		Name  struct {
			First string `json:"first"`
			Last  string `json:"last"`
		} `json:"name"`
	} `json:"traits"`
	MetadataPublic string    `json:"metadata_public"`
	MetadataAdmin  string    `json:"metadata_admin"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	OrganizationID string    `json:"organization_id"`
}

func createRegistrationFlow() (string, error) {
	url := config.AppConfig.Kratos.PublicAPI + "/self-service/registration/api"
	client := &http.Client{Timeout: 5 * time.Second}

	var resp *http.Response
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = client.Get(url)
		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if attempt < 3 {
			fmt.Printf("🔁 Retry %d: Kratos self-service registration flow failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil || resp == nil {
		fmt.Printf("❌   Final failure: Kratos self-service registration flow after 3 attempts. Error: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️  Unexpected status from Kratos: %d\nResponse body: %s\n", resp.StatusCode, string(body))
		return "", errors.New("⚠️  Unexpected status from Kratos")
	}

	body, _ := io.ReadAll(resp.Body)
	e := json.Unmarshal([]byte(body), &registrationFlowResponse)
	if e != nil {
		fmt.Printf("❌ Error decoding Kratos registration flow response: %v\n", e)
		return "", e
	}

	return registrationFlowResponse["id"].(string), nil
}

func registrationIdentity(flowID, email, firstName, lastName, password string) (bool, error) {
	var reqBody RegistrationRequest
	reqBody.Method = "password"
	reqBody.Password = password
	reqBody.Traits.Email = email
	reqBody.Traits.Name.First = firstName
	reqBody.Traits.Name.Last = lastName

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("❌ Error marshaling registration request: %v\n", err)
		return false, err
	}

	url := config.AppConfig.Kratos.PublicAPI + "/self-service/registration?flow=" + flowID
	client := &http.Client{Timeout: 5 * time.Second}

	var resp *http.Response
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if attempt < 3 {
			fmt.Printf("🔁 Retry %d: Kratos self-service registration failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil || resp == nil {
		fmt.Printf("❌   Final failure: Kratos self-service registration after 3 attempts. Error: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️  Unexpected status from Kratos: %d\nResponse body: %s\n", resp.StatusCode, string(body))
		return false, errors.New("⚠️  Unexpected status from Kratos")
	}

	var registrationResponse RegistrationResponse
	if e := json.NewDecoder(resp.Body).Decode(&registrationResponse); e != nil {
		fmt.Printf("❌ Error decoding Kratos registration response: %v\n", e)
		return false, e
	}

	fmt.Printf("🪪  Identity %s registered with identifier: %s\n", email, registrationResponse.Identity.Identifier)
	return true, nil
}

func CheckIdentity(email string) (bool, error) {
	url := config.AppConfig.Kratos.AdminAPI + "/admin/identities?email=" + email
	client := &http.Client{Timeout: 60 * time.Second}

	var resp *http.Response
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = client.Get(url)

		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if attempt < 3 {
			fmt.Printf("🔁 Retry %d: Kratos check sessions failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil || resp == nil {
		fmt.Printf("❌   Final failure: Kratos check sessions after 3 attempts. Error: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️  Unexpected status from Kratos: %d\nResponse body: %s\n", resp.StatusCode, string(body))
		return false, errors.New("⚠️  Unexpected status from Kratos")
	}

	checkIdentityResponse := make([]CheckIdentityResponse, 1)
	body, e1 := io.ReadAll(resp.Body)
	if e1 != nil {
		fmt.Printf("❌   Error reading response body: %v\n", e1)
		return false, e1
	}
	e2 := json.Unmarshal([]byte(body), &checkIdentityResponse)
	if e2 != nil {
		fmt.Printf("❌   Error decoding check identity response: %v\n", e2)
		return false, e2
	}
	firstIdentity := checkIdentityResponse[0]

	if firstIdentity.State == "active" {
		return true, nil
	}

	return false, nil
}

func RegisterIdentity(email, firstName, lastName, password string) (bool, error) {
	var err error
	regFlowId, err := createRegistrationFlow()
	if err != nil || regFlowId == "" {
		fmt.Printf("❌   Cannot get a registration flowID from Kratos. Error: %v\n", err)
		return false, err
	}

	created, err := registrationIdentity(regFlowId, email, firstName, lastName, password)
	if err != nil || !created {
		fmt.Printf("❌   Cannot get a create an identity for %s. Error: %v\n", email, err)
		return false, err
	}

	return created, nil
}

func getStatus(resp *http.Response) int {
	if resp != nil {
		return resp.StatusCode
	}
	return 0
}
