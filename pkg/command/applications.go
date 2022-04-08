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

package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

func newApplicationsCommand(cfg Config) *cobra.Command {
	return &cobra.Command{
		Use:     "applications [NAME ...]",
		Aliases: []string{"application", "apps", "app"},

		ValidArgsFunction: validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
			directive |= cobra.ShellCompDirectiveNoFileComp
			l.forEachApplication(func(item *applications.ApplicationItem) {
				if strings.HasPrefix(item.Name.String(), toComplete) {
					completions = append(completions, item.Name.String())
				}
			})
			return
		}),
	}
}

// NewGetApplicationsCommand returns a command for getting applications.
func NewGetApplicationsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		batchSize int
	)

	cmd := newApplicationsCommand(cfg)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API:       applications.NewAPI(client),
			BatchSize: batchSize,
		}

		result := &ApplicationOutput{Items: make([]ApplicationRow, 0, len(args))}
		if len(args) > 0 {
			if err := l.ForEachNamedApplication(ctx, args, false, result.Add); err != nil {
				return err
			}
		} else {
			q := applications.ApplicationListQuery{}
			if err := l.ForEachApplication(ctx, q, result.Add); err != nil {
				return err
			}
		}

		for i := range result.Items {
			if result.Items[i].Recommendations == applications.RecommendationsDisabled {
				continue
			}

			u := result.Items[i].ApplicationItem.Link(api.RelationRecommendations)
			if u == "" {
				continue
			}

			rl, err := l.API.ListRecommendations(ctx, u)
			if err != nil {
				return err
			}
			result.Items[i].DeployInterval = rl.DeployConfiguration.Interval
		}

		return p.Fprint(out, result)
	}

	cmd.Flags().IntVar(&batchSize, "batch-size", batchSize, "fetch large lists in chu`n`ks rather then all at once")

	return cmd
}

// NewDeleteApplicationsCommand returns a command for deleting applications.
func NewDeleteApplicationsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		ignoreNotFound bool
	)

	cmd := newApplicationsCommand(cfg)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		return l.ForEachNamedApplication(ctx, args, ignoreNotFound, func(item *applications.ApplicationItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			if err := l.API.DeleteApplication(ctx, selfURL); err != nil {
				return err
			}

			return p.Fprint(out, item)
		})
	}

	cmd.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", ignoreNotFound, "treat not found errors as successful deletes")

	return cmd
}
