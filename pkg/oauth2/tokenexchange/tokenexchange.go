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

// Package tokenexchange provides an RFC 8693 OAuth 2.0 token exchange for
// obtaining tokens used for impersonation or delegation.
// https://datatracker.ietf.org/doc/html/rfc8693
package tokenexchange

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// TokenType represents the allowed issued token types.
type TokenType string

const (
	TokenTypeAccessToken     = TokenType("urn:ietf:params:oauth:token-type:access_token")
	TokenTypeRefreshToken    = TokenType("urn:ietf:params:oauth:token-type:refresh_token")
	TokenTypeIdentifierToken = TokenType("urn:ietf:params:oauth:token-type:id_token")
	TokenTypeSAML1           = TokenType("urn:ietf:params:oauth:token-type:saml1")
	TokenTypeSAML2           = TokenType("urn:ietf:params:oauth:token-type:saml2")
	TokenTypeJWT             = TokenType("urn:ietf:params:oauth:token-type:jwt")
)

// ExchangeToken represents an exchanged token: either as input (as in a subject
// or actor) or as output (as in an issued token).
type ExchangeToken struct {
	oauth2.Token
	IssuedTokenType TokenType `json:"issued_token_type"`
}

// Valid tests to see if this token meets the minimum requirements for being valid.
func (t *ExchangeToken) Valid() bool {
	// TODO Should we perform additional validation based on the token type?
	return t != nil && t.Token.Valid() && t.IssuedTokenType != ""
}

// Assertion returns the SAML 1 or 2 assertion represented by this token.
func (t *ExchangeToken) Assertion() ([]byte, error) {
	if t.IssuedTokenType != TokenTypeSAML1 && t.IssuedTokenType != TokenTypeSAML2 {
		return nil, fmt.Errorf("oauth2: exchanged token is not a SAML token")
	}
	return base64.URLEncoding.DecodeString(t.AccessToken)
}

// ExchangeTokenSource is either something that can provide a token as input for
// an exchange, or something that can provide a token as output from an exchange.
type ExchangeTokenSource interface {
	// Token returns an exchange token.
	Token() (*ExchangeToken, error)
}

// Config is used to hold the basic configuration information for facilitating token exchanges.
type Config struct {
	// TokenURL is the URL of the authorization server's token endpoint.
	TokenURL string
	// Resource is the optional URI indicating where the issued tokens will be used.
	Resource string
	// Audience is the optional URI identifying the logical service where the issued tokens will be used.
	Audience string
	// Scopes are optionally used to determine how the issued token can be used.
	Scopes []string
	// RequestedTokenType optionally specifies the type of token that should be issued.
	RequestedTokenType TokenType
	// Actor is used to identify the software acting on behalf of the subject.
	Actor ExchangeTokenSource
}

// Exchange a subject token a new token.
func (c *Config) Exchange(ctx context.Context, subject *ExchangeToken) (*ExchangeToken, error) {
	// Make sure we have a valid subject token
	if !subject.Valid() {
		return nil, fmt.Errorf("oauth2: invalid subject token")
	}

	// We are going to reuse the Client Credentials logic from the stock OAuth2
	// library to perform this exchange. Client Credentials is somewhat similar
	// in that makes a single request to the token endpoint.
	cc := clientcredentials.Config{
		AuthStyle: oauth2.AuthStyleInParams,
		TokenURL:  c.TokenURL,
		Scopes:    c.Scopes,
		EndpointParams: url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {subject.AccessToken},
			"subject_token_type": {string(subject.IssuedTokenType)},
		},
	}

	// Set the optional request parameters
	if c.Resource != "" {
		cc.EndpointParams.Set("resource", c.Resource)
	}
	if c.Audience != "" {
		cc.EndpointParams.Set("audience", c.Audience)
	}
	if c.RequestedTokenType != "" {
		cc.EndpointParams.Set("requested_token_type", string(c.RequestedTokenType))
	}
	if c.Actor != nil {
		actor, err := c.Actor.Token()
		if err != nil {
			return nil, err
		}
		if !actor.Valid() {
			return nil, fmt.Errorf("oauth2: invalid actor token")
		}
		cc.EndpointParams.Set("actor_token", actor.AccessToken)
		cc.EndpointParams.Set("actor_token_type", string(actor.IssuedTokenType))
	}

	// Request the token and extract the issued token type
	tk, err := cc.Token(ctx)
	if err != nil {
		return nil, err
	}
	issuedTokenType, ok := tk.Extra("issued_token_type").(string)
	if !ok || issuedTokenType == "" {
		return nil, fmt.Errorf("oauth2: missing issued token type")
	}
	t := &ExchangeToken{
		Token:           *tk,
		IssuedTokenType: TokenType(issuedTokenType),
	}
	return t, nil
}

// TokenSource returns a new token source for exchanging a source of subject tokens.
func (c *Config) TokenSource(ctx context.Context, subject ExchangeTokenSource) ExchangeTokenSource {
	return ReuseExchangeTokenSource(nil, &exchangeTokenSource{
		ctx:  ctx,
		conf: c,
		sub:  subject,
	})
}

// StaticExchangeTokenSource returns a token source for a static token.
func StaticExchangeTokenSource(t *ExchangeToken) ExchangeTokenSource {
	return staticExchangeTokenSource{t: t}
}

// ReuseExchangeTokenSource returns a token source that uses the supplied token
// until it is no longer valid; then it sources a new token.
func ReuseExchangeTokenSource(t *ExchangeToken, src ExchangeTokenSource) ExchangeTokenSource {
	if rt, ok := src.(*reuseExchangeTokenSource); ok {
		src = rt.new
	}
	return &reuseExchangeTokenSource{t: t, new: src}
}

// OAuth2ExchangeTokenSource returns a token source based on the assumption that
// the supplied OAuth2 token source is providing either access or refresh tokens.
func OAuth2ExchangeTokenSource(src oauth2.TokenSource) ExchangeTokenSource {
	return &oauth2ExchangeTokenSource{src: src}
}

type exchangeTokenSource struct {
	ctx  context.Context
	conf *Config
	sub  ExchangeTokenSource
}

func (s *exchangeTokenSource) Token() (*ExchangeToken, error) {
	if s.sub == nil {
		return nil, fmt.Errorf("missing subject token source")
	}

	subject, err := s.sub.Token()
	if err != nil {
		return nil, err
	}

	return s.conf.Exchange(s.ctx, subject)
}

type staticExchangeTokenSource struct {
	t *ExchangeToken
}

func (s staticExchangeTokenSource) Token() (*ExchangeToken, error) {
	return s.t, nil
}

type reuseExchangeTokenSource struct {
	new ExchangeTokenSource
	mu  sync.Mutex
	t   *ExchangeToken
}

func (s *reuseExchangeTokenSource) Token() (*ExchangeToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t.Valid() {
		return s.t, nil
	}
	t, err := s.new.Token()
	if err != nil {
		return nil, err
	}
	s.t = t
	return t, nil
}

type oauth2ExchangeTokenSource struct {
	src oauth2.TokenSource
}

func (s *oauth2ExchangeTokenSource) Token() (*ExchangeToken, error) {
	t, err := s.src.Token()
	if err != nil {
		return nil, err
	}

	itt := TokenTypeAccessToken
	if t.AccessToken == "" && t.RefreshToken != "" {
		itt = TokenTypeRefreshToken
	}

	return &ExchangeToken{Token: *t, IssuedTokenType: itt}, nil
}
