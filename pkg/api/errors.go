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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ErrorType is an identifying token for errors.
type ErrorType string

const (
	ErrUnauthorized ErrorType = "unauthorized"
	ErrUnexpected   ErrorType = "unexpected"
)

// Error represents the API specific error messages and may be used in response to HTTP status codes
type Error struct {
	Type       ErrorType     `json:"-"`
	Message    string        `json:"error"`
	RetryAfter time.Duration `json:"-"`
	Location   string        `json:"-"`
}

// Error returns the message associated with this API error.
func (e *Error) Error() string {
	return e.Message
}

// NewUnexpectedError returns an error in situations where the API returned an
// undocumented status for the requested resource.
func NewUnexpectedError(resp *http.Response, body []byte) *Error {
	t := ErrUnexpected
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired:
		t = ErrUnauthorized
	}
	return NewError(t, resp, body)
}

// NewError returns a new error with an API specific error condition, it also captures the details of the response
func NewError(t ErrorType, resp *http.Response, body []byte) *Error {
	err := &Error{Type: t}

	// Unmarshal the response body into the error to get the server supplied error message
	// TODO We should be comparing compatible media types here (e.g. charset)
	if resp.Header.Get("Content-Type") == "application/json" {
		_ = json.Unmarshal(body, err)
	}

	// Capture the URL of the request
	if resp.Request != nil && resp.Request.URL != nil {
		err.Location = resp.Request.URL.String()
	}

	// Capture the Retry-After header for "service unavailable"
	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
		if ra, _ := strconv.Atoi(resp.Header.Get("Retry-After")); ra > 0 {
			err.RetryAfter = time.Duration(ra) * time.Second
		}
	}

	// Make sure we have a message
	if err.Message == "" {
		switch resp.StatusCode {
		case http.StatusNotFound:
			err.Message = fmt.Sprintf("not found: %s", err.Location)
		case http.StatusUnauthorized:
			err.Message = "unauthorized"
		case http.StatusPaymentRequired:
			err.Message = "account is not activated"
		default:
			switch err.Type {
			case ErrUnexpected:
				err.Message = fmt.Sprintf("unexpected server response (%s)", http.StatusText(resp.StatusCode))
			default:
				err.Message = strings.ReplaceAll(string(err.Type), "-", " ")
			}
		}
	}

	return err
}

// IsUnauthorized checks to see if the error is an "unauthorized" error.
func IsUnauthorized(err error) bool {
	// OAuth errors (e.g. fetching tokens) will have a full HTTP response
	var oauthErr *oauth2.RetrieveError
	if errors.As(err, &oauthErr) && oauthErr.Response.StatusCode == http.StatusUnauthorized {
		return true
	}

	// Our API errors have an identifiable type code
	var apiErr *Error
	if errors.As(err, &apiErr) && apiErr.Type == ErrUnauthorized {
		return true
	}

	// Handle a specific gateway generated error message
	if err != nil && err.Error() == "no Bearer token" {
		return true
	}

	return false
}
