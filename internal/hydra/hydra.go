package hydra

import (
	"crdb-ory-load-test/internal/client"
	"crdb-ory-load-test/internal/config"
	"crdb-ory-load-test/internal/process"
	"encoding/json"
	"fmt"

	"net/url"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/field-eng-powertools/stopper"
	"github.com/google/uuid"
)

type createClientRequest struct {
	AccessTokenStrategy                        string    `json:"access_token_strategy,omitempty"`
	AllowedCorsOrigins                         []string  `json:"allowed_cors_origins,omitempty"`
	Audience                                   []string  `json:"audience,omitempty"`
	AuthCodeGrantAccessTokenLifespan           string    `json:"authorization_code_grant_access_token_lifespan,omitempty"`
	AuthCodeGrantIdTokenLifespan               string    `json:"authorization_code_grant_id_token_lifespan,omitempty"`
	AuthCodeGrantCodeGrantRefreshTokenLifespan string    `json:"authorization_code_grant_refresh_token_lifespan,omitempty"`
	BackchannelLogoutSessionRequired           bool      `json:"backchannel_logout_session_required,omitempty"`
	BackchannelLogoutURI                       string    `json:"backchannel_logout_uri,omitempty"`
	ClientCredentialsGrantAccessTokenLifespan  string    `json:"client_credentials_grant_access_token_lifespan,omitempty"`
	ClientID                                   string    `json:"client_id,omitempty"`
	ClientName                                 string    `json:"client_name,omitempty"`
	ClientSecret                               string    `json:"client_secret,omitempty"`
	ClientSecretExpiresAt                      int64     `json:"client_secret_expires_at,omitempty"`
	ClientURI                                  string    `json:"client_uri,omitempty"`
	Contacts                                   []string  `json:"contacts,omitempty"`
	CreatedAt                                  time.Time `json:"created_at,omitempty"`
	FrontchannelLogoutSessionRequired          bool      `json:"frontchannel_logout_session_required,omitempty"`
	FrontchannelLogoutURI                      string    `json:"frontchannel_logout_uri,omitempty"`
	GrantTypes                                 []string  `json:"grant_types"`
	ImplicitGrantAccessTokenLifespan           string    `json:"implicit_grant_access_token_lifespan,omitempty"`
	ImplicitGrantIdTokenLifespan               string    `json:"implicit_grant_id_token_lifespan,omitempty"`
	JWKS                                       string    `json:"jwks,omitempty"`
	JWTBearerGrantAccessTokenLifspan           string    `json:"jwt_bearer_grant_access_token_lifespan,omitempty"`
	LogoURI                                    string    `json:"logo_uri,omitempty"`
	Metadata                                   string    `json:"metadata,omitempty"`
	Owner                                      string    `json:"owner,omitempty"`
	PolicyURI                                  string    `json:"policy_uri,omitempty"`
	PostLogoutRedirectURIs                     []string  `json:"post_logout_redirect_uris,omitempty"`
	RedirectURIs                               []string  `json:"redirect_uris,omitempty"`
	RefreshTokenGrantAccessTokenLifespan       string    `json:"refresh_token_grant_access_token_lifespan,omitempty"`
	RefreshTokenGrantIdTokenLifespan           string    `json:"refresh_token_grant_id_token_lifespan,omitempty"`
	RefreshTokenGrantRefreshTokenLifespan      string    `json:"refresh_token_grant_refresh_token_lifespan,omitempty"`
	RegistrationAccessToken                    string    `json:"registration_access_token,omitempty"`
	RegistrationClientURI                      string    `json:"registration_client_uri,omitempty"`
	RequestObjectSigningAlgorithm              string    `json:"request_object_signing_alg,omitempty"`
	RequestURIs                                []string  `json:"request_uris,omitempty"`
	ResponseTypes                              []string  `json:"response_types,omitempty"`
	Scope                                      string    `json:"scope,omitempty"`
	SectorIdentifierURI                        string    `json:"sector_identifier_uri,omitempty"`
	SkipContent                                bool      `json:"skip_consent,omitempty"`
	SkipLogoutConsent                          bool      `json:"skip_logout_consent,omitempty"`
	SubjectType                                string    `json:"subject_type,omitempty"`
	TokenEndpointAuthMethod                    string    `json:"token_endpoint_auth_method,omitempty"`
	TokenEndpointAuthSigningAlgorithm          string    `json:"token_endpoint_auth_signing_alg,omitempty"`
	TosURI                                     string    `json:"tos_uri,omitempty"`
	UpdatedAt                                  time.Time `json:"updated_at,omitempty"`
	UserinfoSignedResponseAlgorithm            string    `json:"userinfo_signed_response_alg,omitempty"`
	PkceEnforced                               bool      `json:"pkce_enforced,omitempty"`
}

type Hydra struct {
	admin  *client.Client
	public *client.Client
}

var _ process.Client[*Credentials] = &Hydra{}

func New(cfg *config.Config) *Hydra {
	return &Hydra{
		admin:  client.New(cfg.Hydra.AdminAPI, cfg.Readers()),
		public: client.New(cfg.Hydra.PublicAPI, cfg.Writers()),
	}
}

func (h *Hydra) BuildReaders(ctx *stopper.Context, cfg *config.Config) ([]process.Consumer[*Credentials], error) {
	readers := make([]process.Consumer[*Credentials], cfg.Readers())
	for idx := range readers {
		readers[idx] = &Reader{
			Client: h,
			Name:   fmt.Sprintf("hydra-reader-%d", idx),
		}
	}
	return readers, nil
}

func (h *Hydra) BuildWriters(
	ctx *stopper.Context,
	cfg *config.Config,
) ([]process.Producer[*Credentials], error) {
	writers := make([]process.Producer[*Credentials], cfg.Writers())
	for idx := range writers {
		writer := &Writer{
			Client: h,
			ID:     uuid.New().String(),
			Secret: uuid.New().String(),
			Name:   fmt.Sprintf("hydra-writer-%d", idx),
		}
		writers[idx] = writer
		created, err := h.CreateOAuth2Client(ctx, writer)
		if err != nil || !created {
			return nil, errors.Join(err, errors.New("failed to create writer"))
		}
	}
	return writers, nil
}

func (h *Hydra) CreateOAuth2Client(ctx *stopper.Context, w *Writer) (bool, error) {
	var reqBody createClientRequest
	reqBody.AccessTokenStrategy = "jwt"
	reqBody.ClientID = w.ID
	reqBody.ClientName = w.Name
	reqBody.ClientSecret = w.Secret
	reqBody.ClientSecretExpiresAt = 0
	reqBody.GrantTypes = []string{"client_credentials"}
	reqBody.ResponseTypes = []string{"code"}
	reqBody.RequestObjectSigningAlgorithm = "RS256"
	reqBody.Scope = "offline_access offline openid"
	reqBody.TokenEndpointAuthMethod = "client_secret_post"
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}
	res, err := h.admin.PostJson(ctx, "clients", jsonData)
	return res != nil, err
}

func (h *Hydra) GrantClientCredentials(ctx *stopper.Context, clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	body, err := h.public.PostForm(ctx, "oauth2/token", data)
	if err != nil {
		return "", errors.Wrap(err, "request to oauth2/token failed")
	}
	var grantClientCredentialsResponse map[string]interface{}
	if err = json.Unmarshal(body, &grantClientCredentialsResponse); err != nil {
		return "", errors.Wrap(err, "invalid credential")
	}
	return grantClientCredentialsResponse["access_token"].(string), nil
}

func (h *Hydra) HealthCheck(ctx *stopper.Context) error {
	if err := h.admin.HealthCheck(ctx); err != nil {
		return err
	}
	if err := h.public.HealthCheck(ctx); err != nil {
		return err
	}
	return nil
}

func (h *Hydra) IntrospectToken(ctx *stopper.Context, token string) (bool, error) {
	data := url.Values{}
	data.Set("token", token)
	body, err := h.admin.PostForm(ctx, "oauth2/introspect", data)
	if err != nil {
		return false, errors.Wrap(err, "request to oauth2/introspect failed")
	}
	var tokenIntrospectionResponse map[string]interface{}
	err = json.Unmarshal(body, &tokenIntrospectionResponse)
	if err != nil {
		return false, errors.Wrap(err, "invalid token")
	}
	return tokenIntrospectionResponse["active"].(bool), nil
}

func (h *Hydra) String() string {
	return "hydra"
}
