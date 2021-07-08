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
	"flag"
	"net/http"

	"github.com/thestormforge/optimize-go/pkg/api"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// ClientConfiguration is used to gather configuration for an integration testing API client.
type ClientConfiguration struct {
	// The URL of the StormForge server.
	Address string
	// A static token to use for authorization.
	StaticToken oauth2.Token
	// Configuration to use a client credentials grant for authorization.
	ClientCredentials clientcredentials.Config
}

// Authorization returns a round tripper for handling request authorization. May
// return `nil` to allow for accessing unprotected endpoints.
func (c *ClientConfiguration) Authorization(ctx context.Context) http.RoundTripper {
	switch {
	case c.StaticToken.AccessToken != "":
		return &oauth2.Transport{Source: oauth2.StaticTokenSource(&c.StaticToken)}
	case c.ClientCredentials.ClientID != "":
		return &oauth2.Transport{Source: c.ClientCredentials.TokenSource(ctx)}
	}
	return nil
}

// NewClient returns a new API client from the default configuration.
func NewClient(ctx context.Context) (api.Client, error) {
	// TODO Should we return a nil client if the address is HTTPS and both the access token and client ID are empty?
	return api.NewClient(DefaultConfig.Address, DefaultConfig.Authorization(ctx))
}

// DefaultConfig is a client configuration to use for integration testing. It's default values are populated using flags.
var DefaultConfig = ClientConfiguration{
	ClientCredentials: clientcredentials.Config{
		EndpointParams: map[string][]string{"audience": {"https://api.carbonrelay.io/v1/"}},
		AuthStyle:      oauth2.AuthStyleInParams,
	},
}

// init sets the values for integration testing via flags.
func init() {
	flag.StringVar(&DefaultConfig.Address, "stormforge.address", "https://api.stormforge.dev/", "the `url` of the StormForge API server")
	flag.StringVar(&DefaultConfig.ClientCredentials.TokenURL, "stormforge.token-url", "https://auth.stormforge.dev/oauth/token", "the `url` of the StormForge token endpoint")
	flag.StringVar(&DefaultConfig.ClientCredentials.ClientID, "stormforge.client-id", "", "the client `identifier` used to obtain an access token")
	flag.StringVar(&DefaultConfig.ClientCredentials.ClientSecret, "stormforge.client-secret", "", "the client `secret` used to obtain an access token")
	flag.StringVar(&DefaultConfig.StaticToken.AccessToken, "stormforge.access-token", "", "the bearer `token` to authorize requests with")
}
