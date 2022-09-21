/*
Copyright 2022 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Config is a simple top level configuration object for client configuration.
type Config struct {
	// The API server address, this should correspond exactly to value of the
	// audience specified during token exchanges.
	Server string `json:"server" yaml:"server" env:"STORMFORGE_SERVER" envDefault:"https://api.stormforge.io/"`
	// The API authorization server address, this should correspond exactly to
	// the expected issuer claim of the tokens being used.
	Issuer string `json:"issuer,omitempty" yaml:"issuer,omitempty" env:"STORMFORGE_ISSUER" envDefault:"https://auth.stormforge.io/"`
	// The client ID used to obtain tokens via a client credentials grant.
	ClientID string `json:"client_id,omitempty" yaml:"client_id,omitempty" env:"STORMFORGE_CLIENT_ID"`
	// The client secret used to obtain tokens via a client credentials grant.
	ClientSecret string `json:"client_secret,omitempty" yaml:"client_secret,omitempty" env:"STORMFORGE_CLIENT_SECRET"`
	// The list of scopes to request during token exchanges.
	Scopes []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	// A hard-coded bearer token for debugging, the token will not be refreshed
	// so the caller is responsible for providing a valid token.
	Token string `json:"-" yaml:"-" env:"STORMFORGE_TOKEN"`
}

// Address returns the API server address. The canonical value will be slash-terminated,
// however it is not guaranteed and callers are responsible for sanitizing the value.
func (cfg *Config) Address() string {
	return cfg.Server
}

// Transport wraps the supplied round tripper (presumably the `http.DefaultTransport`)
// based on the current state of the configuration.
func (cfg *Config) Transport(ctx context.Context, base http.RoundTripper) http.RoundTripper {
	return &transport{
		Transport: oauth2.Transport{
			Source: cfg.TokenSource(ctx),
			Base:   base,
		},
		Audience: cfg.Server,
	}
}

// TokenSource returns a new source for obtaining tokens. The token source may be
// nil if there is insufficient configuration available, typically this would
// indicate the API server does not require authorization.
func (cfg *Config) TokenSource(ctx context.Context) oauth2.TokenSource {
	switch {

	case cfg.Token != "":
		return oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: cfg.Token,
		})

	case cfg.ClientID != "":
		tokenURL, err := url.Parse(cfg.Issuer)
		if err != nil {
			return &errorTokenSource{err: err}
		}
		if tokenURL.Scheme != "https" {
			return &errorTokenSource{err: fmt.Errorf("issuer is required and must be HTTPS")}
		}
		tokenURL, err = tokenURL.Parse("oauth/token")
		if err != nil {
			return &errorTokenSource{err: err}
		}

		cc := clientcredentials.Config{
			ClientID:       cfg.ClientID,
			ClientSecret:   cfg.ClientSecret,
			TokenURL:       tokenURL.String(),
			Scopes:         cfg.Scopes,
			EndpointParams: url.Values{"audience": []string{cfg.Server}},
			AuthStyle:      oauth2.AuthStyleInParams,
		}
		return cc.TokenSource(ctx)

	default:
		return nil
	}
}

// transport wraps a stock OAuth2 transport with a check that ensures outbound
// requests only include tokens if they match the configured audience.
type transport struct {
	// The standard OAuth2 transport.
	oauth2.Transport
	// The audience used to filter request URLs.
	Audience string
}

// RoundTrip ensures the audience value matches the request before adding tokens.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Transport.Source != nil && t.requiresAuthorization(req.URL) {
		return t.Transport.RoundTrip(req)
	}

	if t.Base != nil {
		return t.Base.RoundTrip(req)
	}

	return http.DefaultTransport.RoundTrip(req)
}

// requiresAuthorization tests the supplied URL to see if it matches the
// effective audience.
func (t *transport) requiresAuthorization(u *url.URL) bool {
	// Check the actual configured audience value
	if strings.HasPrefix(u.String(), t.Audience) {
		return true
	}

	// Support an alternate audience for testing the application service
	if endpoint := os.Getenv("STORMFORGE_APPLICATIONS_ENDPOINT"); endpoint != "" {
		if strings.HasPrefix(u.String(), endpoint) {
			return true
		}

		// Special case other resources directly under /v2/
		if c, err := url.Parse(endpoint); err == nil {
			c.Path = path.Join(c.Path, "..", "clusters")
			if strings.HasPrefix(u.String(), c.String()) {
				return true
			}
			c.Path = path.Join(c.Path, "..", "application-activity")
			if strings.HasPrefix(u.String(), c.String()) {
				return true
			}
		}
	}

	// Support an alternate audience for testing the experiment service
	if endpoint := os.Getenv("STORMFORGE_EXPERIMENTS_ENDPOINT"); endpoint != "" {
		if strings.HasPrefix(u.String(), endpoint) {
			return true
		}
	}

	return false
}

// errorTokenSource is a TokenSource that always returns an error.
type errorTokenSource struct {
	err error
}

// Token always returns a non-nil error.
func (ts *errorTokenSource) Token() (*oauth2.Token, error) {
	if ts.err == nil {
		panic("errorTokenSource created with nil error")
	}
	return nil, ts.err
}
