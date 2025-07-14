package kratos

import (
	"crdb-ory-load-test/internal/client"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/process"
	"encoding/json"
	"fmt"
	"log"
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

var _ process.Client[*Identity] = &Kratos{}

func New(config *config.Config) *Kratos {
	return &Kratos{
		admin:  client.New(config.Kratos.AdminAPI),
		public: client.New(config.Kratos.PublicAPI),
	}
}

func (k *Kratos) BuildReaders(ctx *stopper.Context, cfg *config.Config) ([]process.Consumer[*Identity], error) {
	readers := make([]process.Consumer[*Identity], cfg.Writers())
	for idx := range readers {
		reader := &Reader{
			Client: k,
			Name:   fmt.Sprintf("kratos-writer-%d", idx),
		}
		readers[idx] = reader
	}
	return readers, nil
}

func (k *Kratos) BuildWriters(ctx *stopper.Context, cfg *config.Config) ([]process.Producer[*Identity], error) {
	writers := make([]process.Producer[*Identity], cfg.Writers())
	for idx := range writers {
		writer := &Writer{
			Client: k,
			Name:   fmt.Sprintf("kratos-writer-%d", idx),
		}
		writers[idx] = writer
	}
	return writers, nil
}

func (k *Kratos) CheckIdentity(ctx *stopper.Context, email string) (bool, error) {
	body, err := k.admin.Get(ctx, "identities", "email="+email)
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

func (k *Kratos) RegisterIdentity(ctx *stopper.Context, identity *Identity, password string) error {
	var err error
	regFlowId, err := k.createRegistrationFlow(ctx)
	if err != nil {
		return err
	}
	if regFlowId == "" {
		return errors.New("registration flow failed")
	}
	return k.registrationIdentity(ctx, regFlowId, identity, password)
}

func (k *Kratos) String() string {
	return "kratos"
}

func (k *Kratos) createRegistrationFlow(ctx *stopper.Context) (string, error) {
	body, err := k.public.Get(ctx, "self-service/registration/api")
	if err != nil {
		return "", errors.Wrap(err, "request to relation-tuples/check failed")
	}
	var registrationFlowResponse map[string]any
	if err := json.Unmarshal(body, &registrationFlowResponse); err != nil {
		log.Printf("Error decoding Kratos registration flow response: %v\n", err)
		return "", err
	}
	return registrationFlowResponse["id"].(string), nil
}

func (k *Kratos) registrationIdentity(ctx *stopper.Context, flowID string, identity *Identity, password string) error {
	var reqBody RegistrationRequest
	reqBody.Method = "password"
	reqBody.Password = password
	reqBody.Traits.Email = identity.Email
	reqBody.Traits.Name.First = identity.FirstName
	reqBody.Traits.Name.Last = identity.LastName

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Error marshaling registration request: %v\n", err)
		return err
	}
	body, err := k.public.PostJson(ctx, "self-service/registration", jsonData, "flow="+flowID)
	if err != nil {
		return errors.Wrap(err, "request to relation-tuples/check failed")
	}
	var registrationResponse map[string]any
	if err := json.Unmarshal(body, &registrationResponse); err != nil {
		log.Printf("Error decoding Kratos registration response: %v\n", err)
		return err
	}
	return nil
}
