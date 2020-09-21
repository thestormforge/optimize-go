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

package authorizationcode

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_AuthCodeURLWithPKCE(t *testing.T) {
	cases := []struct {
		desc                        string
		verifier                    []byte
		expectedVerifier            string
		expectedCodeChallengeMethod string
		expectedCodeChallenge       string
	}{
		{
			desc: "Example for the S256 code_challenge_method",
			verifier: []byte{116, 24, 223, 180, 151, 153, 224, 37, 79, 250, 96, 125, 216, 173,
				187, 186, 22, 212, 37, 77, 105, 214, 191, 240, 91, 88, 5, 88, 83, 132, 141, 121},
			expectedVerifier:            "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			expectedCodeChallengeMethod: "S256",
			expectedCodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			acf, err := NewAuthorizationCodeFlowWithPKCE()
			require.NoError(t, err)

			// Override verifier and check the internal encoding
			acf.setVerifier(c.verifier)
			assert.Equal(t, c.expectedVerifier, acf.verifier)

			u, err := url.Parse(acf.AuthCodeURLWithPKCE())
			require.NoError(t, err)

			// Check the URL query parameters
			q := u.Query()
			assert.Equal(t, c.expectedCodeChallengeMethod, q.Get("code_challenge_method"))
			assert.Equal(t, c.expectedCodeChallenge, q.Get("code_challenge"))
		})
	}
}

func TestConfig_ExchangeWithPKCE(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)

		assert.Equal(t, "exchange-code", r.PostForm.Get("code"))
		assert.Equal(t, "authorization_code", r.PostForm.Get("grant_type"))
		assert.Equal(t, base64.RawURLEncoding.EncodeToString([]byte("verifier")), r.PostForm.Get("code_verifier"))

		// Send back an access token to sanity check the response
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = w.Write([]byte("access_token=access-token"))
	}))
	defer ts.Close()

	acf := &Config{}
	acf.Endpoint.TokenURL = ts.URL + "/token"
	acf.setVerifier([]byte("verifier"))

	tok, err := acf.ExchangeWithPKCE(context.Background(), "exchange-code")
	if assert.NoError(t, err) {
		assert.Equal(t, tok.AccessToken, "access-token")
	}
}
