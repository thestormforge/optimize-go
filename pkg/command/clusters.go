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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

// NewEditClusterCommand returns a command for editing a cluster.
func NewEditClusterCommand(cfg Config, p Printer) *cobra.Command {
	var (
		title string
	)

	cmd := &cobra.Command{
		Use:               "cluster NAME",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: validClusterArgs(cfg),
	}

	cmd.Flags().StringVar(&title, "title", "", "update the `title` value")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		return l.ForEachNamedCluster(ctx, args, false, func(item *applications.ClusterItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			// Update the title
			if title != "" {
				if err := l.API.PatchCluster(ctx, selfURL, applications.ClusterTitle{Title: title}); err != nil {
					return err
				}
			}

			return p.Fprint(out, NewClusterRow(item))
		})
	}
	return cmd
}

// NewGetClustersCommand returns a command for getting clusters.
func NewGetClustersCommand(cfg Config, p Printer) *cobra.Command {
	var (
		product string
	)

	cmd := &cobra.Command{
		Use:               "clusters [NAME ...]",
		Aliases:           []string{"cluster"},
		ValidArgsFunction: validClusterArgs(cfg),
	}

	cmd.Flags().StringVar(&product, "for", product, "show only clusters for a specific `product`; one of: optimize-pro|optimize-live")

	_ = cmd.RegisterFlagCompletionFunc("for", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"optimize-pro", "optimize-live"}, cobra.ShellCompDirectiveDefault
	})

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		result := &ClusterOutput{Items: make([]ClusterRow, 0, len(args))}
		if len(args) > 0 {
			if err := l.ForEachNamedCluster(ctx, args, false, result.Add); err != nil {
				return err
			}
		} else {
			q := applications.ClusterListQuery{}
			switch product {
			case "optimize-pro", "pro":
				q.SetModules(applications.ClusterScenarios)
			case "optimize-live", "live":
				q.SetModules(applications.ClusterRecommendations)
			}
			if err := l.ForEachCluster(ctx, q, result.Add); err != nil {
				return err
			}
		}

		return p.Fprint(out, result)
	}
	return cmd
}

// NewDeleteClustersCommand returns a command for deleting clusters.
func NewDeleteClustersCommand(cfg Config, p Printer) *cobra.Command {
	var (
		ignoreNotFound bool
	)

	cmd := &cobra.Command{
		Use:               "clusters [NAME ...]",
		Aliases:           []string{"cluster"},
		ValidArgsFunction: validClusterArgs(cfg),
	}

	cmd.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", ignoreNotFound, "treat not found errors as successful deletes")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		return l.ForEachNamedCluster(ctx, args, ignoreNotFound, func(item *applications.ClusterItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			if err := l.API.DeleteCluster(ctx, selfURL); err != nil {
				return err
			}

			return p.Fprint(out, NewClusterRow(item))
		})
	}
	return cmd
}

func validClusterArgs(cfg Config, modules ...applications.ClusterModule) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
		directive |= cobra.ShellCompDirectiveNoFileComp
		l.forAllClusters(func(item *applications.ClusterItem) {
			if strings.HasPrefix(item.Name.String(), toComplete) {
				completions = append(completions, item.Name.String()+"\t"+item.Title())
			}
		}, modules...)
		return
	})
}
