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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

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
	// Additional parameters to be included with the token request.
	AuthorizationParams url.Values
	// A hard-coded bearer token for debugging, the token will not be refreshed
	// so the caller is responsible for providing a valid token.
	Token string `json:"-" yaml:"-" env:"STORMFORGE_TOKEN"`
	// Flag indicating that unauthorized errors are tolerated. If left false (the default),
	// unauthorized errors will trigger an exit(77). The default behavior is intended
	// to prevent unattended, unauthorized software from repeatedly attempting to access the
	// API: e.g. if a client ID/secret is revoked, the software will terminate rather than
	// continuously attempt to re-authorize.
	AllowUnauthorized bool
}

// Address returns the API server address. The canonical value will be slash-terminated,
// however it is not guaranteed and callers are responsible for sanitizing the value.
func (cfg *Config) Address() string {
	return cfg.Server
}

// Transport wraps the supplied round tripper based on the current state of the configuration.
func (cfg *Config) Transport(ctx context.Context, base http.RoundTripper) http.RoundTripper {
	return &transport{
		Transport: oauth2.Transport{
			Source: &lazyTokenSource{init: func() oauth2.TokenSource { return cfg.TokenSource(ctx) }},
			Base:   base,
		},
		Audience: cfg.Server,
	}
}

// TokenSource returns a new source for obtaining tokens. The token source may be
// nil if there is insufficient configuration available, typically this would
// indicate the API server does not require authorization.
func (cfg *Config) TokenSource(ctx context.Context) oauth2.TokenSource {
	var result oauth2.TokenSource
	switch {

	case cfg.Token != "":
		result = oauth2.StaticTokenSource(&oauth2.Token{
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
			EndpointParams: cfg.AuthorizationParams,
			AuthStyle:      oauth2.AuthStyleInParams,
		}

		if cc.EndpointParams == nil {
			cc.EndpointParams = url.Values{}
		}
		cc.EndpointParams.Set("audience", cfg.Server)

		result = cc.TokenSource(ctx)

	}

	// Force unauthorized responses to exit
	if !cfg.AllowUnauthorized && result != nil {
		result = &exitTokenSource{src: result}
	}

	return result
}

// lazyTokenSource is a token source whose initialization is deferred, allowing
// consuming code to establish configuration even after constructing the token
// source. This is relevant for Cobra as it allows the token source to be
// established _before_ the environment variables are parsed, thereby preventing
// a common bug related to observability during the initialization sequence.
type lazyTokenSource struct {
	init   func() oauth2.TokenSource
	doInit sync.Once
	src    oauth2.TokenSource
}

// Token returns a token from the wrapped token source.
func (ts *lazyTokenSource) Token() (*oauth2.Token, error) {
	ts.doInit.Do(func() {
		ts.src = ts.init()
	})
	return ts.src.Token()
}

// exitTokenSource forces an os.Exit if an attempt to fetch a token fails with
// an "Unauthorized" (401) status.
type exitTokenSource struct {
	src oauth2.TokenSource
}

// Token retrieves a token from the wrapped source. If an OAuth2 error is returned
// with a 401 status, the program exits with a status of 77 (EX_NOPERM).
func (ts *exitTokenSource) Token() (*oauth2.Token, error) {
	t, err := ts.src.Token()
	if err != nil {
		var oauthErr *oauth2.RetrieveError
		if errors.As(err, &oauthErr) && oauthErr.Response.StatusCode == http.StatusUnauthorized {
			// Note that no attempt is made to "log" an error message as it is
			// assumed the exit status code is unique enough to this condition
			// to identify the root cause of the failure.
			os.Exit(77)
		}
		return nil, err
	}
	return t, nil
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
	// TODO This is a hack, really NONE of the requests through the OAuth2 client should return true here
	if strings.Contains(u.Path, "/oauth/") {
		return false
	}

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
