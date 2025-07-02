package hydra

import (
	"bytes"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
    "net/url"

	"crdb-ory-load-test/internal/config"
)

type createClientRequest struct {
	AccessTokenStrategy string `json:"access_token_strategy,omitempty"`
	AllowedCorsOrigins  []string `json:"allowed_cors_origins,omitempty"`
	Audience  []string `json:"audience,omitempty"`
	AuthCodeGrantAccessTokenLifespan string `json:"authorization_code_grant_access_token_lifespan,omitempty"`
	AuthCodeGrantIdTokenLifespan string `json:"authorization_code_grant_id_token_lifespan,omitempty"`
	AuthCodeGrantCodeGrantRefreshTokenLifespan string `json:"authorization_code_grant_refresh_token_lifespan,omitempty"`
	BackchannelLogoutSessionRequired bool `json:"backchannel_logout_session_required,omitempty"`
	BackchannelLogoutURI string `json:"backchannel_logout_uri,omitempty"`
	ClientCredentialsGrantAccessTokenLifespan string `json:"client_credentials_grant_access_token_lifespan,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	ClientSecretExpiresAt int64 `json:"client_secret_expires_at,omitempty"`
	ClientURI string `json:"client_uri,omitempty"`
	Contacts []string `json:"contacts,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	FrontchannelLogoutSessionRequired bool `json:"frontchannel_logout_session_required,omitempty"`
	FrontchannelLogoutURI string `json:"frontchannel_logout_uri,omitempty"`
	GrantTypes []string `json:"grant_types"`
	ImplicitGrantAccessTokenLifespan string `json:"implicit_grant_access_token_lifespan,omitempty"`
	ImplicitGrantIdTokenLifespan string `json:"implicit_grant_id_token_lifespan,omitempty"`
	JWKS string `json:"jwks,omitempty"`
	JWTBearerGrantAccessTokenLifspan string `json:"jwt_bearer_grant_access_token_lifespan,omitempty"`
	LogoURI string `json:"logo_uri,omitempty"`
	Metadata string `json:"metadata,omitempty"`
	Owner string `json:"owner,omitempty"`
	PolicyURI string `json:"policy_uri,omitempty"`
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris,omitempty"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	RefreshTokenGrantAccessTokenLifespan string `json:"refresh_token_grant_access_token_lifespan,omitempty"`
	RefreshTokenGrantIdTokenLifespan string `json:"refresh_token_grant_id_token_lifespan,omitempty"`
	RefreshTokenGrantRefreshTokenLifespan string `json:"refresh_token_grant_refresh_token_lifespan,omitempty"`
	RegistrationAccessToken string `json:"registration_access_token,omitempty"`
	RegistrationClientURI string `json:"registration_client_uri,omitempty"`
	RequestObjectSigningAlgorithm string `json:"request_object_signing_alg,omitempty"`
	RequestURIs []string `json:"request_uris,omitempty"`
	ResponseTypes []string `json:"response_types,omitempty"`
	Scope string `json:"scope,omitempty"`
	SectorIdentifierURI string `json:"sector_identifier_uri,omitempty"`
	SkipContent bool `json:"skip_consent,omitempty"`
	SkipLogoutConsent bool `json:"skip_logout_consent,omitempty"`
	SubjectType string `json:"subject_type,omitempty"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`
	TokenEndpointAuthSigningAlgorithm string `json:"token_endpoint_auth_signing_alg,omitempty"`
	TosURI string `json:"tos_uri,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	UserinfoSignedResponseAlgorithm string `json:"userinfo_signed_response_alg,omitempty"`
	PkceEnforced bool `json:"pkce_enforced,omitempty"`
}


func CreateOAuth2Client(id, name, secret string) (bool, error) {
    var reqBody createClientRequest
    reqBody.AccessTokenStrategy = "jwt"
    reqBody.ClientID = id
    reqBody.ClientName = name
    reqBody.ClientSecret = secret
    reqBody.ClientSecretExpiresAt = 0
    reqBody.GrantTypes = []string{"client_credentials"}
    reqBody.ResponseTypes = []string{"code"}
    reqBody.RequestObjectSigningAlgorithm = "RS256"
    reqBody.Scope = "offline_access offline openid"
    reqBody.TokenEndpointAuthMethod = "client_secret_post"

	jsonData, e := json.Marshal(reqBody)
	if e != nil {
		fmt.Printf("❌ Error marshaling create client request: %v\n", e)
		return false, e
	}

	url := *config.AppConfig.Hydra.AdminAPI + "/clients"
	client := &http.Client{Timeout: 60 * time.Second}

	var resp *http.Response
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err == nil && resp != nil && resp.StatusCode == 201 {
			break
		}
		if attempt < 3 {
			fmt.Printf("🔁 Retry %d: Hydra OAuth2 client creation failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil || resp == nil {
		fmt.Printf("❌   Final failure: Hydra OAuth2 client creation after 3 attempts. Error: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️  Unexpected status from Hydra: %d\nResponse body: %s\n", resp.StatusCode, string(body))
		return false, errors.New("⚠️  Unexpected status from Hydra")
	}

	return true, nil
}

func GrantClientCredentials(clientID, clientSecret string) (string, error) {
    endpoint := *config.AppConfig.Hydra.PublicAPI + "/oauth2/token"
    data := url.Values{}
    data.Set("grant_type", "client_credentials")
    data.Set("client_id", clientID)
    data.Set("client_secret", clientSecret)

	req, e := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    if e != nil {
        fmt.Printf("❌ Error creating grant request: %v\n", e)
        return "", e
    }

    client := &http.Client{Timeout: 60 * time.Second}

    var resp *http.Response
    var err error
    for attempt := 1; attempt <= 3; attempt++ {
        resp, err = client.Do(req)
        if err == nil && resp != nil && resp.StatusCode == 200 {
            break
        }
        if attempt < 3 {
            fmt.Printf("🔁 Retry %d: Hydra OAuth2 client credentials grant failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
            time.Sleep(100 * time.Millisecond)
        }
    }

    if err != nil || resp == nil {
        fmt.Printf("❌   Final failure: Hydra OAuth2 client credentials grant after 3 attempts. Error: %v\n", err)
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        fmt.Printf("⚠️  Unexpected status from Hydra: %d\nResponse body: %s\n", resp.StatusCode, string(body))
        return "", errors.New("⚠️  Unexpected status from Hydra")
    }

    body, _ := io.ReadAll(resp.Body)
    var grantClientCredentialsResponse map[string]interface{}
    ex := json.Unmarshal([]byte(body), &grantClientCredentialsResponse)
    if e != nil {
        fmt.Printf("❌ Error decoding Hydra Client Credentials grant response: %v\n", ex)
        return "", ex
    }

    return grantClientCredentialsResponse["access_token"].(string), nil
}

func IntrospectToken(token string) (bool, error) {
        endpoint := *config.AppConfig.Hydra.AdminAPI + "/oauth2/introspect"
        data := url.Values{}
        data.Set("token", token)

    	req, e := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))

        if e != nil {
            fmt.Printf("❌ Error creating introspect request: %v\n", e)
            return false, e
        }

    	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

        client := &http.Client{Timeout: 60 * time.Second}

        var resp *http.Response
        var err error
        for attempt := 1; attempt <= 3; attempt++ {
            resp, err = client.Do(req)
            if err == nil && resp != nil && resp.StatusCode == 200 {
                break
            }
            if attempt < 3 {
                fmt.Printf("🔁 Retry %d: Hydra OAuth2 token introspection failed (status=%v, error=%v)\n", attempt, getStatus(resp), err)
                time.Sleep(100 * time.Millisecond)
            }
        }

        if err != nil || resp == nil {
            fmt.Printf("❌   Final failure: Hydra OAuth2 token introspection after 3 attempts. Error: %v\n", err)
            return false, err
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
            body, _ := io.ReadAll(resp.Body)
            fmt.Printf("⚠️  Unexpected status from Hydra: %d\nResponse body: %s\n", resp.StatusCode, string(body))
            return false, errors.New("⚠️  Unexpected status from Hydra")
        }

        body, _ := io.ReadAll(resp.Body)
        var tokenIntrospectionResponse map[string]interface{}
        ex := json.Unmarshal([]byte(body), &tokenIntrospectionResponse)
        if ex != nil {
            fmt.Printf("❌ Error decoding Hydra Client Credentials grant response: %v\n", ex)
            return false, ex
        }
        return tokenIntrospectionResponse["active"].(bool), nil
}

func getStatus(resp *http.Response) int {
	if resp != nil {
		return resp.StatusCode
	}
	return 0
}
