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

package api

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// Client is used to handle interactions with the API Server.
type Client interface {
	// URL returns the location of the specified endpoint
	URL(endpoint string) *url.URL
	// Do performs the interaction specified by the HTTP request
	Do(context.Context, *http.Request) (*http.Response, []byte, error)
}

// NewClient returns a new client for accessing API server.
func NewClient(address string, transport http.RoundTripper) (Client, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	return &httpClient{
		endpoint: u,
		client: http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}, nil
}

type httpClient struct {
	endpoint *url.URL
	client   http.Client
}

// URL resolves an endpoint to a fully qualified URL.
func (c *httpClient) URL(ep string) *url.URL {
	u, err := c.endpoint.Parse(ep)
	if err != nil {
		// If code panics here, the caller needs to verify it's input before
		// passing it on to the `Client.URL(endpoint string)` function.
		panic(err)
	}
	return u
}

// Do executes an HTTP request using this client and the supplied context.
func (c *httpClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var body []byte
	done := make(chan struct{})
	go func() {
		body, err = ioutil.ReadAll(resp.Body)
		close(done)
	}()

	select {
	case <-ctx.Done():
		<-done
		err = resp.Body.Close()
		if err == nil {
			err = ctx.Err()
		}
	case <-done:
	}

	return resp, body, err
}
