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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpClient_URL(t *testing.T) {
	cases := []struct {
		desc     string
		address  string
		endpoint string
		url      string
	}{
		{
			desc:     "standard",
			address:  "https://example.com/v1",
			endpoint: "/experiments/",
			url:      "https://example.com/v1/experiments/",
		},
		{
			desc:     "named resource endpoint",
			address:  "https://example.com/v1",
			endpoint: "/experiments/foobar",
			url:      "https://example.com/v1/experiments/foobar",
		},
		{
			desc:     "trailing address slash",
			address:  "https://example.com/v1/",
			endpoint: "/experiments/",
			url:      "https://example.com/v1/experiments/",
		},
		{
			desc:     "no base path",
			address:  "https://example.com",
			endpoint: "/experiments/",
			url:      "https://example.com/experiments/",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if client, err := NewClient(c.address, nil); assert.NoError(t, err) {
				assert.Equal(t, c.url, client.URL(c.endpoint).String())
			}
		})
	}
}
