package kratos

import (
	"crdb-ory-load-test/internal/client"
	"crdb-ory-load-test/internal/config"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/cockroachdb/field-eng-powertools/stopper"
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

type Kratos struct {
	admin  *client.Client
	public *client.Client
}

func New(config *config.Config) *Kratos {
	return &Kratos{
		admin:  client.New(config.Keto.ReadAPI),
		public: client.New(config.Keto.WriteAPI),
	}
}

func (k *Kratos) CheckIdentity(ctx *stopper.Context, email string) (bool, error) {
	body, err := k.admin.Get(ctx, "/admin/identities?email="+email)
	if err != nil {
		return false, errors.Wrap(err, "request to relation-tuples/check failed")
	}
	checkIdentityResponse := make([]CheckIdentityResponse, 1)
	if err := json.Unmarshal([]byte(body), &checkIdentityResponse); err != nil {
		fmt.Printf("error decoding check identity response: %v\n", err)
		return false, err
	}
	firstIdentity := checkIdentityResponse[0]
	if firstIdentity.State == "active" {
		return true, nil
	}
	return false, nil
}

func (k *Kratos) HealthCheck(ctx *stopper.Context) error {
	if err := k.admin.HealthCheck(ctx); err != nil {
		return err
	}
	if err := k.public.HealthCheck(ctx); err != nil {
		return err
	}
	return nil
}

func (k *Kratos) RegisterIdentity(ctx *stopper.Context, email, firstName, lastName, password string) (bool, error) {
	var err error
	regFlowId, err := k.createRegistrationFlow(ctx)
	if err != nil || regFlowId == "" {
		return false, err
	}
	created, err := k.registrationIdentity(ctx, regFlowId, email, firstName, lastName, password)
	if err != nil || !created {
		return false, err
	}
	return created, nil
}

func (k *Kratos) createRegistrationFlow(ctx *stopper.Context) (string, error) {
	body, err := k.public.Get(ctx, "self-service/registration/api")
	if err != nil {
		return "", errors.Wrap(err, "request to relation-tuples/check failed")
	}
	var registrationFlowResponse map[string]interface{}
	if err := json.Unmarshal(body, &registrationFlowResponse); err != nil {
		fmt.Printf("Error decoding Kratos registration flow response: %v\n", err)
		return "", err
	}
	return registrationFlowResponse["id"].(string), nil
}

func (k *Kratos) registrationIdentity(ctx *stopper.Context, flowID, email, firstName, lastName, password string) (bool, error) {
	var reqBody RegistrationRequest
	reqBody.Method = "password"
	reqBody.Password = password
	reqBody.Traits.Email = email
	reqBody.Traits.Name.First = firstName
	reqBody.Traits.Name.Last = lastName

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshaling registration request: %v\n", err)
		return false, err
	}
	body, err := k.public.PostJson(ctx, "self-service/registration/api", jsonData)
	if err != nil {
		return false, errors.Wrap(err, "request to relation-tuples/check failed")
	}
	var registrationResponse RegistrationResponse
	if err := json.Unmarshal(body, &registrationResponse); err != nil {
		fmt.Printf("Error decoding Kratos registration response: %v\n", err)
		return false, err
	}
	return true, nil
}
