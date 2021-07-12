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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexQuery_SetLimit(t *testing.T) {
	q := IndexQuery{}

	q.SetLimit(11)
	q.SetLimit(97)
	assert.Equal(t, []string{"97"}, q[ParamLimit])

	q.SetLimit(0)
	assert.NotContains(t, q, ParamLimit)
}

func TestIndexQuery_SetLabelSelector(t *testing.T) {
	q := IndexQuery{}

	q.SetLabelSelector(map[string]string{
		"application": "my-app",
		"scenario":    "cyber-monday",
	})
	assert.Equal(t, []string{"application=my-app,scenario=cyber-monday"}, q[ParamLabelSelector])

	q.SetLabelSelector(map[string]string{
		"best": "true",
	})
	assert.Equal(t, []string{"application=my-app,scenario=cyber-monday", "best=true"}, q[ParamLabelSelector])
}

func TestIndexQuery_nil(t *testing.T) {
	// Ensure the setter on a nil value allocates a map, otherwise embedding the
	// IndexQuery will have unexpected results
	q := IndexQuery(nil)
	assert.Nil(t, q)
	q.SetLimit(97)
	assert.NotNil(t, q)
	assert.Equal(t, []string{"97"}, q[ParamLimit])
}

func TestIndexQuery_AppendToURL(t *testing.T) {
	cases := []struct {
		desc     string
		q        *IndexQuery
		u        string
		expected string
	}{
		{
			desc: "empty",
		},
		{
			desc:     "empty query",
			q:        &IndexQuery{},
			u:        "foobar",
			expected: "foobar",
		},
		{
			desc: "relative URL",
			q: &IndexQuery{
				"offset": []string{"10"},
			},
			u:        "foobar",
			expected: "foobar?offset=10",
		},
		{
			desc: "qualified URL",
			q: &IndexQuery{
				"offset": []string{"10"},
			},
			u:        "https://example.com/foobar/",
			expected: "https://example.com/foobar/?offset=10",
		},
		{
			desc: "query merges",
			q: &IndexQuery{
				"offset": []string{"10"},
			},
			u:        "https://example.com/foobar/?limit=20",
			expected: "https://example.com/foobar/?limit=20&offset=10",
		},
		{
			desc: "query appends",
			q: &IndexQuery{
				"limit": []string{"10"},
			},
			u:        "https://example.com/foobar/?limit=20",
			expected: "https://example.com/foobar/?limit=20&limit=10",
		},
		{
			desc: "multiple parameters",
			q: &IndexQuery{
				"limit":  []string{"10"},
				"offset": []string{"30"},
			},
			u:        "https://example.com/foobar",
			expected: "https://example.com/foobar?limit=10&offset=30",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			u, err := c.q.AppendToURL(c.u)
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, u)
			}
		})
	}
}
