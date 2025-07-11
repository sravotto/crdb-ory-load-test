package hydra

import "time"

var defaultTimeout = 100 * time.Millisecond

type Credentials struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
}
