/*
Copyright 2022 GramLabs, Inc.

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

package command

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// NewGetActivityCommand returns a command for getting activity feed items.
func NewGetActivityCommand(cfg Config, p Printer) *cobra.Command {
	var (
		tags []string
	)

	cmd := &cobra.Command{
		Use:     "activity-feed",
		Aliases: []string{"activity", "feed"},
	}

	cmd.Flags().StringSliceVar(&tags, "tags", nil, "limit activity items to the specified `tag`s")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		appAPI := applications.NewAPI(client)

		q := applications.ActivityFeedQuery{}
		if len(tags) > 0 {
			q.SetType(tags...)
		}

		md, err := appAPI.CheckEndpoint(ctx)
		if err != nil {
			return err
		}

		u := md.Link(api.RelationAlternate)
		if u == "" {
			return fmt.Errorf("missing activity feed URL")
		}

		feed, err := appAPI.ListActivity(ctx, u, q)
		if err != nil {
			return err
		}

		result := &ActivityOutput{
			Items:        make([]ActivityRow, 0, len(feed.Items)),
			ActivityFeed: feed,
		}
		for i := range feed.Items {
			result.Add(&feed.Items[i])
		}
		return p.Fprint(out, result)
	}
	return cmd
}

// NewWatchActivityCommand returns a command for watching the activity feed.
func NewWatchActivityCommand(cfg Config) *cobra.Command {
	var (
		pollInterval         time.Duration
		jitterFactor         float64
		hideFailedActivities bool
		tags                 []string
		deleteItems          bool

		feedTemplateText string
		itemTemplateText string
		userAgent        string
	)

	cmd := &cobra.Command{
		Use: "activity-feed",
	}

	cmd.Flags().DurationVar(&pollInterval, "poll", 30*time.Second, "polling `interval` to refresh the feed")
	cmd.Flags().Float64Var(&jitterFactor, "jitter", 1.0, "polling jitter `factor` to refresh the feed")
	cmd.Flags().BoolVar(&hideFailedActivities, "no-failed", false, "do not show items with a failure reason")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "limit activity items to the specified `tag`s")
	cmd.Flags().BoolVar(&deleteItems, "delete", false, "delete new items")
	cmd.Flags().StringVar(&feedTemplateText, "feed-template", `{{ template "ActivityFeed" . }}`, "the feed `template` used to render the activity feed")
	cmd.Flags().StringVar(&itemTemplateText, "item-template", `{{ template "ActivityItem" . }}`, "the item `template` used to render the items")
	cmd.Flags().StringVar(&userAgent, "user-agent", "", "override the User-Agent `header` used when making requests")
	cmd.Flag("feed-template").Hidden = true
	cmd.Flag("item-template").Hidden = true
	cmd.Flag("user-agent").Hidden = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		// Create the templates for rendering activities
		tmpl := template.New("activity").Funcs(map[string]interface{}{
			"subject": subject(ctx, cfg),
			"join":    strings.Join,
		})
		template.Must(tmpl.New("ActivityFeed").Parse(`Feed:
  Feed URL: {{ .FeedURL }}
  Subject: {{ subject }}
`))
		template.Must(tmpl.New("ActivityItem").Parse(`Item:
  ID: {{ .ID }}
  External URL: {{ .ExternalURL }}
  Tags: {{ join .Tags ", " }}
  Time: {{ .DatePublished.Format "` + time.RFC3339 + `" }}
`))

		feedTemplate, err := tmpl.New("feed").Parse(feedTemplateText)
		if err != nil {
			return err
		}
		itemTemplate, err := tmpl.New("item").Parse(itemTemplateText)
		if err != nil {
			return err
		}

		// Override the user agent for debugging
		if userAgent != "" {
			ctx = context.WithValue(ctx, "User-Agent", userAgent)
		}

		s := &applications.PollingSubscriber{
			API:                    applications.NewAPI(client),
			PollInterval:           pollInterval,
			JitterFactor:           jitterFactor,
			ReportFailedActivities: !hideFailedActivities,
		}

		q := applications.ActivityFeedQuery{}
		if len(tags) > 0 {
			q.SetType(tags...)
		}

		// Normally you would just call API.Subscribe; but we want to display additional information
		md, err := s.API.CheckEndpoint(ctx)
		if err != nil {
			return err
		}

		u := md.Link(api.RelationAlternate)
		if u == "" {
			return fmt.Errorf("missing activity feed URL")
		}

		feed, err := s.API.ListActivity(ctx, u, q)
		if err != nil {
			return err
		}

		if err := feedTemplate.Execute(out, feed); err != nil {
			return err
		}

		// Create a channel to watch for new items
		activity := make(chan applications.ActivityItem)
		go func() {
			for item := range activity {
				// Render each item
				if err := itemTemplate.Execute(out, item); err != nil {
					_, _ = fmt.Fprintf(out, "Error: failed to render activity %q: %v", item.URL, err)
				}

				// If requested, delete the item to prevent it from being processed again
				if deleteItems {
					if err := s.API.DeleteActivity(ctx, item.URL); err != nil {
						_, _ = fmt.Fprintf(out, "Error: failed to delete activity %q: %v\n", item.URL, err)
					}
				}
			}
		}()

		// Set the feed URL and start polling
		s.FeedURL = feed.FeedURL
		return s.Subscribe(ctx, activity)
	}
	return cmd
}

// subject is a function we can use in templates to extract the subject claim from
// an authorization token.
func subject(ctx context.Context, cfg Config) func() (string, error) {
	tcfg, ok := cfg.(interface {
		TokenSource(context.Context) oauth2.TokenSource
	})
	if !ok {
		return func() (string, error) { return "<unavailable>", nil }
	}

	return func() (string, error) {
		tok, err := tcfg.TokenSource(ctx).Token()
		if err != nil {
			return "", err
		}
		accessToken, err := jwt.ParseSigned(tok.AccessToken)
		if err != nil {
			return "", err
		}
		c := jwt.Claims{}
		if err := accessToken.UnsafeClaimsWithoutVerification(&c); err != nil {
			return "", err
		}
		return c.Subject, nil
	}
}
