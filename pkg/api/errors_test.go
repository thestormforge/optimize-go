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

package api

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestNewError(t *testing.T) {
	cases := []struct {
		desc      string
		errorType ErrorType
		response  http.Response
		body      []byte
		expected  Error
	}{
		{
			desc: "empty",
		},
		{
			desc:      "api message",
			errorType: ErrorType("test-error"),
			response: http.Response{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
			body: []byte(`{"error":"test message"}`),
			expected: Error{
				Type:    "test-error",
				Message: "test message",
			},
		},
		{
			desc:      "not found",
			errorType: ErrorType("test-error"),
			response: http.Response{
				StatusCode: http.StatusNotFound,
				Request: &http.Request{
					URL: &url.URL{
						Scheme: "https",
						Host:   "invalid.example.com",
						Path:   "/testing",
					},
				},
			},
			expected: Error{
				Type:    "test-error",
				Message: "not found: https://invalid.example.com/testing",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Error(t, &c.expected, NewError(c.errorType, &c.response, c.body))
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	cases := []struct {
		desc     string
		err      error
		expected bool
	}{
		{
			desc: "empty",
		},
		{
			desc: "format error",
			err:  fmt.Errorf("test"),
		},
		{
			desc:     "hard coded error text",
			err:      fmt.Errorf("no Bearer token"),
			expected: true,
		},
		{
			desc:     "api error",
			err:      &Error{Type: ErrUnauthorized},
			expected: true,
		},
		{
			desc:     "wrapped api error",
			err:      fmt.Errorf("test: %w", &Error{Type: ErrUnauthorized}),
			expected: true,
		},
		{
			desc: "oauth2 error",
			err: &url.Error{ // http.Client.Do wraps errors in *url.Error
				Err: &oauth2.RetrieveError{
					Response: &http.Response{
						StatusCode: http.StatusUnauthorized,
					},
				},
			},
			expected: true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, IsUnauthorized(c.err))
		})
	}
}
