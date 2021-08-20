/*
Copyright 2020 GramLabs, Inc.

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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"

	"github.com/thestormforge/optimize-go/pkg/oauth2/authorizationcode"
	"github.com/thestormforge/optimize-go/pkg/oauth2/devicecode"
	"github.com/thestormforge/optimize-go/pkg/oauth2/registration"
	"github.com/thestormforge/optimize-go/pkg/oauth2/tokenexchange"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Loader is used to initially populate an Optimize configuration
type Loader func(cfg *OptimizeConfig) error

// Change is used to apply a configuration change that should be persisted
type Change func(cfg *Config) error

// ClientIdentity is a mapping function that returns an OAuth 2.0 `client_id` given an authorization server issuer identifier
type ClientIdentity func(string) string

// OptimizeConfig is the structure used to manage configuration data
type OptimizeConfig struct {
	// Filename is the path to the configuration file; if left blank, it will be populated using XDG base directory conventions on the next Load
	Filename string
	// Overrides to the standard configuration
	Overrides Overrides
	// ClientIdentity is used to determine the OAuth 2.0 client identifier
	ClientIdentity ClientIdentity
	// AuthorizationParameters is used to provide additional parameters to the OAuth 2.0 endpoints
	AuthorizationParameters map[string][]string

	data        Config
	unpersisted []Change
}

// MarshalJSON ensures only the configuration data is marshalled
func (rsc *OptimizeConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(rsc.data)
}

// Load will populate the client configuration
func (rsc *OptimizeConfig) Load(extra ...Loader) error {
	var loaders []Loader
	loaders = append(loaders, fileLoader, envLoader, migrationLoader)
	loaders = append(loaders, extra...)
	loaders = append(loaders, defaultLoader)
	for i := range loaders {
		if err := loaders[i](rsc); err != nil {
			return err
		}
	}
	return nil
}

// Update will make a change to the configuration data that should be persisted on the next call to Write
func (rsc *OptimizeConfig) Update(change Change) error {
	if err := change(&rsc.data); err != nil {
		return err
	}
	rsc.unpersisted = append(rsc.unpersisted, change)
	return nil
}

// Write all unpersisted changes to disk
func (rsc *OptimizeConfig) Write() error {
	if rsc.Filename == "" || len(rsc.unpersisted) == 0 {
		return nil
	}

	f := file{filename: rsc.Filename}
	if err := f.read(); err != nil {
		return err
	}

	for i := range rsc.unpersisted {
		if err := rsc.unpersisted[i](&f.data); err != nil {
			return err
		}
	}

	if err := f.write(); err != nil {
		return err
	}

	rsc.unpersisted = nil
	return nil
}

// Merge combines the supplied data with what is already present in this client configuration; unlike Update, changes
// will not be persisted on the next write
func (rsc *OptimizeConfig) Merge(data *Config) {
	mergeConfig(&rsc.data, data)
}

// Reader returns a configuration reader for accessing information from the configuration
func (rsc *OptimizeConfig) Reader() Reader {
	return &overrideReader{overrides: &rsc.Overrides, delegate: &defaultReader{cfg: &rsc.data}}
}

// Environment returns the name of the execution environment
func (rsc *OptimizeConfig) Environment() string {
	if env := rsc.Overrides.Environment; env != "" {
		return env
	}
	if env := rsc.data.Environment; env != "" {
		return env
	}
	return "production"
}

// SystemNamespace returns the namespace where the Optimize Controller is/should be installed
func (rsc *OptimizeConfig) SystemNamespace() (string, error) {
	ctrl, err := CurrentController(rsc.Reader())
	if err != nil {
		return "", nil
	}
	return ctrl.Namespace, nil
}

// Kubectl returns an executable command for running kubectl
func (rsc *OptimizeConfig) Kubectl(ctx context.Context, arg ...string) (*exec.Cmd, error) {
	cstr, err := CurrentCluster(rsc.Reader())
	if err != nil {
		return nil, err
	}

	var globals []string

	if cstr.KubeConfig != "" {
		globals = appendIfNotPresent(globals, arg, "--kubeconfig", cstr.KubeConfig)
	}

	if cstr.Context != "" {
		globals = appendIfNotPresent(globals, arg, "--context", cstr.Context)
	}

	if cstr.Namespace != "" {
		globals = appendIfNotPresent(globals, arg, "--namespace", cstr.Namespace)
	}

	return exec.CommandContext(ctx, cstr.Bin, append(globals, arg...)...), nil
}

// appendIfNotPresent is meant to allow args coming to override globals rather then relying on unspecified behavior
func appendIfNotPresent(s []string, arg []string, flag, value string) []string {
	// This won't catch things like a global --namespace and a -n arg
	for i := range arg {
		if arg[i] == flag {
			return s
		}
	}
	return append(s, flag, value)
}

// RevocationInformation contains the information necessary to revoke an authorization credential
type RevocationInformation struct {
	// RevocationURL is the URL of the authorization server's revocation endpoint
	RevocationURL string
	// ClientID is the client identifier for the authorization
	ClientID string
	// Authorization is the credential that needs to be revoked
	Authorization Authorization

	// authorization name is used internally so revocation information can be a change
	authorizationName string
}

// String returns a string representation of this revocation
func (ri *RevocationInformation) String() string {
	return ri.authorizationName
}

// RemoveAuthorization returns a configuration change to clear the credentials for an authorization.
func (ri *RevocationInformation) RemoveAuthorization() Change {
	return func(cfg *Config) error {
		for i := range cfg.Authorizations {
			if cfg.Authorizations[i].Name == ri.authorizationName {
				cfg.Authorizations[i].Authorization.Credential = Credential{}
				return nil
			}
		}
		return nil
	}
}

// RevocationInfo returns the information necessary to revoke an authorization entry from the configuration
func (rsc *OptimizeConfig) RevocationInfo() (*RevocationInformation, error) {
	r := rsc.Reader()

	authorizationName, err := r.AuthorizationName(r.ContextName())
	if err != nil {
		return nil, err
	}
	az, err := r.Authorization(authorizationName)
	if err != nil {
		return nil, err
	}

	srv, err := CurrentServer(r)
	if err != nil {
		return nil, err
	}

	return &RevocationInformation{
		RevocationURL:     srv.Authorization.RevocationEndpoint,
		ClientID:          rsc.clientID(&srv),
		Authorization:     az,
		authorizationName: authorizationName,
	}, nil
}

// RegisterClient performs dynamic client registration
func (rsc *OptimizeConfig) RegisterClient(ctx context.Context, client *registration.ClientMetadata) (*registration.ClientInformationResponse, error) {
	// We can't use the initial token because we don't know if we have a valid token, instead we need to authorize the context client
	src, err := rsc.tokenSource(ctx)
	if err != nil {
		return nil, err
	}
	if src != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, oauth2.NewClient(ctx, src))
	}

	// Get the current server configuration for the registration endpoint address
	srv, err := CurrentServer(rsc.Reader())
	if err != nil {
		return nil, err
	}
	c := registration.Config{
		RegistrationURL: srv.Authorization.RegistrationEndpoint,
	}
	return c.Register(ctx, client)
}

// NewAuthorization creates a new authorization code flow with PKCE using the current context
func (rsc *OptimizeConfig) NewAuthorization() (*authorizationcode.Config, error) {
	srv, err := CurrentServer(rsc.Reader())
	if err != nil {
		return nil, err
	}

	c, err := authorizationcode.NewAuthorizationCodeFlowWithPKCE()
	if err != nil {
		return nil, err
	}

	c.ClientID = rsc.clientID(&srv)
	c.Endpoint.AuthURL = srv.Authorization.AuthorizationEndpoint
	c.Endpoint.TokenURL = srv.Authorization.TokenEndpoint
	c.Endpoint.AuthStyle = oauth2.AuthStyleInParams
	c.EndpointParams = rsc.AuthorizationParameters
	return c, nil
}

// NewDeviceAuthorization creates a new device authorization flow using the current context
func (rsc *OptimizeConfig) NewDeviceAuthorization() (*devicecode.Config, error) {
	srv, err := CurrentServer(rsc.Reader())
	if err != nil {
		return nil, err
	}

	return &devicecode.Config{
		Config: clientcredentials.Config{
			ClientID:  rsc.clientID(&srv),
			TokenURL:  srv.Authorization.TokenEndpoint,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		DeviceAuthorizationURL: srv.Authorization.DeviceAuthorizationEndpoint,
		EndpointParams:         rsc.AuthorizationParameters,
	}, nil
}

// Authorize configures the supplied transport
func (rsc *OptimizeConfig) Authorize(ctx context.Context, transport http.RoundTripper) (http.RoundTripper, error) {
	// Get the token source and use it to wrap the transport
	src, err := rsc.tokenSource(ctx)
	if err != nil {
		return nil, err
	}
	if src != nil {
		return &oauth2.Transport{Source: src, Base: transport}, nil
	}
	return transport, nil
}

// PerformanceAuthorization returns a source of authorization tokens for accessing Performance APIs.
func (rsc *OptimizeConfig) PerformanceAuthorization(ctx context.Context) (tokenexchange.ExchangeTokenSource, error) {
	r := rsc.Reader()
	srv, err := CurrentServer(r)
	if err != nil {
		return nil, err
	}

	ec := tokenexchange.Config{
		TokenURL:           srv.API.PerformanceTokenEndpoint,
		Resource:           "https://api.stormforger.com/",
		RequestedTokenType: tokenexchange.TokenTypeJWT,
		// NOTE: The "actor" should be a JWT exchange token where the "access_token"
		// is a "software statement" (https://datatracker.ietf.org/doc/html/rfc7591#section-2.3).
		// In the current implementation, the "id-token" type is overloaded to
		// mean a simple string ID rather than an actual OpenID identifier token.
		Actor: tokenexchange.StaticExchangeTokenSource(&tokenexchange.ExchangeToken{
			Token:           oauth2.Token{AccessToken: "forge-www"},
			IssuedTokenType: tokenexchange.TokenTypeIdentifierToken,
		}),
	}

	// The tokens produced by the configuration constitute the subject identity
	ts, err := rsc.tokenSource(ctx)
	if err != nil {
		return nil, err
	}
	if ts == nil {
		return nil, fmt.Errorf("missing required authorization source")
	}
	sub := tokenexchange.OAuth2ExchangeTokenSource(ts)

	return ec.TokenSource(ctx, sub), nil
}

func (rsc *OptimizeConfig) tokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	// TODO We could make OptimizeConfig implement the TokenSource interface, but we need a way to handle the context
	r := rsc.Reader()
	srv, err := CurrentServer(r)
	if err != nil {
		return nil, err
	}
	azName, err := r.AuthorizationName(r.ContextName())
	if err != nil {
		return nil, err
	}
	az, err := r.Authorization(azName)
	if err != nil {
		return nil, err
	}

	if az.Credential.ClientCredential != nil {
		cc := clientcredentials.Config{
			ClientID:       az.Credential.ClientID,
			ClientSecret:   az.Credential.ClientSecret,
			TokenURL:       srv.Authorization.TokenEndpoint,
			EndpointParams: url.Values(rsc.AuthorizationParameters),
			AuthStyle:      oauth2.AuthStyleInParams,
		}
		return cc.TokenSource(ctx), nil
	}

	if az.Credential.TokenCredential != nil {
		c := &oauth2.Config{
			ClientID: rsc.clientID(&srv),
			Endpoint: oauth2.Endpoint{
				AuthURL:   srv.Authorization.AuthorizationEndpoint,
				TokenURL:  srv.Authorization.TokenEndpoint,
				AuthStyle: oauth2.AuthStyleInParams,
			},
		}
		t := &oauth2.Token{
			AccessToken:  az.Credential.AccessToken,
			TokenType:    az.Credential.TokenType,
			RefreshToken: az.Credential.RefreshToken,
			Expiry:       az.Credential.Expiry,
		}
		return &updateTokenSource{
			src: c.TokenSource(ctx, t),
			cfg: rsc,
			az:  azName,
		}, nil
	}

	return nil, nil
}

func (rsc *OptimizeConfig) clientID(srv *Server) string {
	if rsc.ClientIdentity != nil {
		return rsc.ClientIdentity(srv.Authorization.Issuer)
	}
	return ""
}

type updateTokenSource struct {
	src oauth2.TokenSource
	cfg *OptimizeConfig
	az  string
}

func (u *updateTokenSource) Token() (*oauth2.Token, error) {
	t, err := u.src.Token()
	if err != nil {
		return nil, err
	}
	if az, err := u.cfg.Reader().Authorization(u.az); err == nil {
		if az.Credential.TokenCredential != nil && az.Credential.AccessToken != t.AccessToken {
			_ = u.cfg.Update(SaveToken(u.az, t))
			_ = u.cfg.Write()
		}
	}
	return t, nil
}
