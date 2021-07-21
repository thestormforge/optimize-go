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

package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivityFeed_SetBaseURL(t *testing.T) {
	cases := []struct {
		desc     string
		base     string
		feed     ActivityFeed
		expected ActivityFeed
	}{
		{
			desc: "empty",
		},
		{
			desc: "relative",
			// Normally the base should be equal to the FeedURL
			base: "https://test.example.com/feed",
			feed: ActivityFeed{
				// Make sure we preserve blank URLs
				HomePageURL: "",
				FeedURL:     "/feed",
				NextURL:     "/feed?next",
				Hubs: []ActivityHub{
					{
						URL: "/subscribe",
					},
				},
				Items: []ActivityItem{
					{
						URL: "/items/1",
						// Make sure we leave absolute URLs
						ExternalURL: "https://other.example.com/items/100",
					},
				},
			},
			expected: ActivityFeed{
				HomePageURL: "",
				FeedURL:     "https://test.example.com/feed",
				NextURL:     "https://test.example.com/feed?next",
				Hubs: []ActivityHub{
					{
						URL: "https://test.example.com/subscribe",
					},
				},
				Items: []ActivityItem{
					{
						URL:         "https://test.example.com/items/1",
						ExternalURL: "https://other.example.com/items/100",
					},
				},
			},
		},
		{
			desc: "item ID is not a URL",
			base: "https://test.example.com/feed",
			feed: ActivityFeed{
				Items: []ActivityItem{
					{
						ID: "/items/1",
					},
				},
			},
			expected: ActivityFeed{
				Items: []ActivityItem{
					{
						ID: "/items/1",
					},
				},
			},
		},
		{
			desc: "actual URI resolution",
			base: "https://test.example.com/feed",
			feed: ActivityFeed{
				FeedURL: "foobar",
			},
			expected: ActivityFeed{
				// We are using the RFC 3986 Section 5.2 definition of path resolution
				FeedURL: "https://test.example.com/foobar",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			f := c.feed
			f.SetBaseURL(c.base)
			assert.Equal(t, c.expected, f)
		})
	}
}
