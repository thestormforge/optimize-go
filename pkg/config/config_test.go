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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoints_Resolve(t *testing.T) {
	cfg := &RedSkyConfig{}
	require.NoError(t, defaultLoader(cfg))
	experimentsEndpoint := &cfg.data.Servers[0].Server.API.ExperimentsEndpoint

	cases := []struct {
		desc                string
		experimentsEndpoint string
		endpoint            string
		expected            string
	}{
		{
			desc:                "default experiments",
			experimentsEndpoint: *experimentsEndpoint, // Default value hack
			endpoint:            "/experiments/",
			expected:            "https://api.stormforge.io/v1/experiments/",
		},
		{
			desc:                "default experiment",
			experimentsEndpoint: *experimentsEndpoint, // Default value hack
			endpoint:            "/experiments/foo_bar",
			expected:            "https://api.stormforge.io/v1/experiments/foo_bar",
		},
		{
			desc:                "default trials",
			experimentsEndpoint: *experimentsEndpoint, // Default value hack
			endpoint:            "/experiments/foo_bar/trials/",
			expected:            "https://api.stormforge.io/v1/experiments/foo_bar/trials/",
		},
		{
			desc:                "explicit endpoint",
			experimentsEndpoint: "http://example.com/api/experiments/",
			endpoint:            "/experiments/",
			expected:            "http://example.com/api/experiments/",
		},
		{
			desc:                "missing trailing slash",
			experimentsEndpoint: "http://example.com/api/experiments",
			endpoint:            "/experiments/",
			expected:            "http://example.com/api/experiments/",
		},
		{
			desc:                "query string",
			experimentsEndpoint: "http://example.com/api/experiments?foo=bar",
			endpoint:            "/experiments/",
			expected:            "http://example.com/api/experiments/?foo=bar",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			*experimentsEndpoint = c.experimentsEndpoint
			ep, err := cfg.Endpoints()
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, ep.Resolve(c.endpoint).String())
			}
		})
	}
}
