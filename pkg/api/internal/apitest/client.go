/*
Copyright 2021 GramLabs, Inc.

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

package apitest

import (
	"context"
	"net/http"
	"os"

	"github.com/thestormforge/optimize-go/pkg/api"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// NewClient returns a new API client from the default configuration.
func NewClient(ctx context.Context) (api.Client, error) {
	address := os.Getenv("STORMFORGE_SERVER")

	transport := &userAgentTransport{}

	if clientID := os.Getenv("STORMFORGE_CLIENT_ID"); clientID != "" {
		cc := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: os.Getenv("STORMFORGE_CLIENT_SECRET"),
			TokenURL:     os.Getenv("STORMFORGE_ISSUER") + "oauth/token",
			AuthStyle:    oauth2.AuthStyleInParams,
			EndpointParams: map[string][]string{
				"audience": {address},
			},
		}
		transport.Base = &oauth2.Transport{
			Source: cc.TokenSource(ctx),
		}
	} else if accessToken := os.Getenv("STORMFORGE_TOKEN"); accessToken != "" {
		transport.Base = &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
		}
	}

	return api.NewClient(address, transport)
}

type uaKey struct{}

// WithUserAgent updates the value of the User-Agent header to send with the supplied context.
func WithUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, uaKey{}, ua)
}

type userAgentTransport struct {
	Base http.RoundTripper
}

// RoundTrip updates the User-Agent header based on the current context value.
func (u *userAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Check the context for a User-Agent value
	if ua, ok := r.Context().Value(uaKey{}).(string); ok {
		r = r.WithContext(r.Context())
		r.Header.Set("User-Agent", ua)
	}

	// Finish the request
	if u.Base != nil {
		return u.Base.RoundTrip(r)
	}
	return http.DefaultTransport.RoundTrip(r)
}
