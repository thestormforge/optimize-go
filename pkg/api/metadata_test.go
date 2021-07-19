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
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetadata(t *testing.T) {
	md := Metadata{
		"Title":         []string{`Testing`},
		"Last-Modified": []string{`fail`},
		"location":      []string{`https://invalid.example.com/testing`},
		"Link": []string{
			`</foo>;rel="abc"`,
			`</bar>; rel=xyz`,
			`</list?offset=0>;rel="previous",</list?offset=10>;rel="next"`,
		},
	}

	// Simple case
	assert.Equal(t, "Testing", md.Title())

	// Invalid time returns zero value
	assert.Equal(t, time.Time{}, md.LastModified())

	// Metadata keys must be in canonical form (use `http.CanonicalMIMEHeaderKey`)
	assert.Equal(t, "", md.Location())

	// Simple case
	assert.Equal(t, "/foo", md.Link("abc"))

	// Quoting, white-space
	assert.Equal(t, "/bar", md.Link("xyz"))

	// Combined links, canonical relations
	assert.Equal(t, "/list?offset=0", md.Link(RelationPrev))
	assert.Equal(t, "/list?offset=10", md.Link(RelationNext))
}

func TestJsonMetadata_UnmarshalJSON(t *testing.T) {
	// Verify last-entry-wins
	data := []byte(`
{
  "Link": "</foo>; rel=foo",
  "Link": "</bar>; rel=bar"
}
`)

	md := jsonMetadata{}
	err := json.Unmarshal(data, &md)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"</bar>; rel=bar"}, md["Link"])
	}
}

func TestUnmarshalMetadata(t *testing.T) {
	cases := []struct {
		desc     string
		resp     http.Response
		expected Metadata
	}{
		{
			desc: "location",
			resp: http.Response{
				Header: http.Header{
					"Location": {`/foo`},
				},
				Request: &http.Request{
					URL: &url.URL{Scheme: "https", Host: "invalid.example.com", Path: "/"},
				},
			},
			expected: Metadata{
				"Location": {`https://invalid.example.com/foo`},
			},
		},
		{
			desc: "simple link",
			resp: http.Response{
				Header: http.Header{
					"Link": {`</foo>;rel="test"`},
				},
				Request: &http.Request{
					URL: &url.URL{Scheme: "https", Host: "invalid.example.com", Path: "/"},
				},
			},
			expected: Metadata{
				"Link": {`<https://invalid.example.com/foo>;rel="test"`},
			},
		},
		{
			desc: "comma links",
			resp: http.Response{
				Header: http.Header{
					"Link": {`</foo>;rel="test",</bar>;rel="test2"`},
				},
				Request: &http.Request{
					URL: &url.URL{Scheme: "https", Host: "invalid.example.com", Path: "/"},
				},
			},
			expected: Metadata{
				"Link": {`<https://invalid.example.com/foo>;rel="test",<https://invalid.example.com/bar>;rel="test2"`},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			md := Metadata{}
			UnmarshalMetadata(&c.resp, &md)
			assert.Equal(t, c.expected, md)
		})
	}
}
