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
