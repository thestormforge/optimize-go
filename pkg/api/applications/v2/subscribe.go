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
	"context"
	"errors"
	"math/rand"
	"sort"
	"time"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// NewSubscriber returns a subscriber for the supplied feed.
func NewSubscriber(api API, feed ActivityFeed) Subscriber {
	// Check the feed hubs for any subscription strategies we support
	for _, hub := range feed.Hubs {
		switch hub.Type {
		case "poll":
			// Allow the server to force polling
			return &PollingSubscriber{API: api, FeedURL: hub.URL}
		}
	}

	// By default, return a simple polling subscriber on the feed URL
	return &PollingSubscriber{API: api, FeedURL: feed.FeedURL}
}

// PollingSubscriber is a primitive strategy that simply polls for changes.
type PollingSubscriber struct {
	// The API instance used to fetch the feed.
	API API
	// The URL to poll.
	FeedURL string
	// Time between polling requests. Defaults to 30 seconds.
	PollInterval time.Duration
	// Adjust the poll duration by a random amount. Defaults to 1.0, effectively
	// a random amount up to the full poll interval.
	JitterFactor float64

	// The server may periodically request a longer delay.
	rateLimit time.Duration
}

// PollTimer returns a new timer for the next polling operation.
func (s *PollingSubscriber) PollTimer() *time.Timer {
	// Default to 30 seconds
	interval := s.PollInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	// Default to a factor of 1.0 (i.e. a random value from 0 to a full extra interval)
	jitter := rand.Float64() * float64(interval)
	if s.JitterFactor > 0 {
		jitter *= s.JitterFactor
	}

	// Include the server requested rate limit just for this timer
	// TODO If `s.rateLimit > interval` should we update `s.PollInterval` to prevent excessive 429s?
	interval += s.rateLimit
	s.rateLimit = 0

	return time.NewTimer(interval + time.Duration(jitter))
}

// Subscribe starts fetching the activity feed
func (s *PollingSubscriber) Subscribe(ctx context.Context, ch chan<- ActivityItem) {
	go func() {
		// Close the channel when we are done sending things
		defer close(ch)

		lastID := ""
		for {
			// Wait for the timer
			t := s.PollTimer()
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}

			// Fetch the feed and send new items to the channel
			f, err := s.API.ListActivity(ctx, s.FeedURL, ActivityFeedQuery{})
			if err != nil {
				var apiErr *api.Error
				if errors.As(err, &apiErr) {
					switch apiErr.Type {
					case ErrActivityRateLimited:
						s.rateLimit = apiErr.RetryAfter
						continue
					}
				}

				// TODO What other errors should just be ignored or reported?
				return
			}

			lastID = s.notify(f.Items, lastID, ch)
		}
	}()
}

// notify sends all of the items from the supplied feed to the channel.
// IMPORTANT: this function assumes item identifiers can be compared lexicographically.
func (s *PollingSubscriber) notify(items []ActivityItem, lastID string, ch chan<- ActivityItem) string {
	// Make sure the items are sorted by their identifier
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	for i := range items {
		if lastID == "" || items[i].ID > lastID {
			ch <- items[i]
			lastID = items[i].ID
		}
	}

	return lastID
}
