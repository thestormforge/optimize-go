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
	"net/url"
	"strings"
	"time"

	"github.com/thestormforge/optimize-go/pkg/api"
)

type ActivityFeed struct {
	HomePageURL string         `json:"home_page_url,omitempty"`
	FeedURL     string         `json:"feed_url,omitempty"`
	NextURL     string         `json:"next_url,omitempty"`
	Hubs        []ActivityHub  `json:"hubs,omitempty"`
	Items       []ActivityItem `json:"items"`
}

type ActivityHub struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type ActivityItem struct {
	ID            string             `json:"id"`
	URL           string             `json:"url,omitempty"`
	ExternalURL   string             `json:"external_url,omitempty"`
	Title         string             `json:"title,omitempty"`
	DatePublished time.Time          `json:"date_published,omitempty"`
	DateModified  time.Time          `json:"date_modified,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	StormForge    *ActivityExtension `json:"_stormforge,omitempty"`
}

func (ai *ActivityItem) HasTag(tag string) bool {
	for _, t := range ai.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

const (
	TagRun     string = "run"
	TagScan    string = "scan"
	TagApprove string = "approve"
	TagRefresh string = "refresh"
)

type ActivityExtension struct {
	ActivityFailure
}

type ActivityFeedQuery struct {
	Query map[string][]string
}

func (q *ActivityFeedQuery) SetType(t ...string) {
	if q.Query == nil {
		q.Query = make(map[string][]string)
	}
	url.Values(q.Query).Set("type", strings.Join(t, ","))
}

type Activity struct {
	api.Metadata `json:"-"`
	Run          *RunActivity     `json:"run,omitempty"`
	Scan         *ScanActivity    `json:"scan,omitempty"`
	Approve      *ApproveActivity `json:"approve,omitempty"`
	Refresh      *RefreshActivity `json:"refresh,omitempty"`
}

type RunActivity struct {
	Scenario string `json:"scenario"`
	ActivityFailure
}

type ScanActivity struct {
	Scenario string `json:"scenario"`
	ActivityFailure
}

type ApproveActivity struct {
	Recommendation string `json:"recommendation"`
	ActivityFailure
}

type RefreshActivity struct {
	Application string `json:"application"`
	ActivityFailure
}

type ActivityPatchRequest struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Data  any      `json:"_stormforge,omitempty"`
}

type ActivityFailure struct {
	FailureReason  string `json:"failure_reason,omitempty"`
	FailureMessage string `json:"failure_message,omitempty"`
}

// SetBaseURL resolves the URLs on the activity feed against a supplied base.
// Typically, the URL used to fetch the feed, the feed's `feed_url` field, and
// the base URL should all match; however, it may be the case that the `feed_url`
// field is returned as a relative URL (the JSON Feed spec does not specify a
// behavior in this regard).
func (af *ActivityFeed) SetBaseURL(u string) {
	// Create a function to resolve references against the base URL
	base, err := url.Parse(u)
	if err != nil {
		return
	}
	res := func(u string) string {
		if u != "" {
			if uu, err := base.Parse(u); err == nil {
				return uu.String()
			}
		}
		return u
	}

	// Resolve all known URLs on the feed
	af.HomePageURL = res(af.HomePageURL)
	af.FeedURL = res(af.FeedURL)
	af.NextURL = res(af.NextURL)
	for i := range af.Hubs {
		af.Hubs[i].URL = res(af.Hubs[i].URL)
	}
	for i := range af.Items {
		af.Items[i].URL = res(af.Items[i].URL)
		af.Items[i].ExternalURL = res(af.Items[i].ExternalURL)
	}
}
