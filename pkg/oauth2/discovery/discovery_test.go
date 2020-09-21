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

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWellKnownURI(t *testing.T) {
	cases := []struct {
		desc     string
		id       string
		name     string
		expected string
	}{
		{
			desc:     "empty",
			expected: "/.well-known/",
		},
		{
			desc:     "default",
			id:       "http://example.com",
			expected: "http://example.com/.well-known/",
		},
		{
			desc:     "named",
			id:       "http://example.com",
			name:     "foo",
			expected: "http://example.com/.well-known/foo",
		},
		{
			desc:     "trailing slash",
			id:       "http://example.com/",
			name:     "foo",
			expected: "http://example.com/.well-known/foo",
		},
		{
			desc:     "relative named",
			name:     "foo",
			expected: "/.well-known/foo",
		},
		{
			desc:     "relative trailing slash",
			id:       "/",
			name:     "foo",
			expected: "/.well-known/foo",
		},

		// NOTE: Where is this behavior defined? It doesn't appear to be documented in RFC 8615.
		{
			desc:     "base path",
			id:       "http://example.com/x",
			name:     "foo",
			expected: "http://example.com/.well-known/foo/x",
		},
		{
			desc:     "relative base path",
			id:       "/x",
			name:     "foo",
			expected: "/.well-known/foo/x",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, WellKnownURI(c.id, c.name))
		})
	}
}
