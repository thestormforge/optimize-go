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

// NewGetClustersCommand returns a command for getting clusters.
func NewGetClustersCommand(cfg Config, p Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "clusters [NAME ...]",
		Aliases:           []string{"cluster"},
		ValidArgsFunction: validClusterArgs(cfg),
	}

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
			return fmt.Errorf("get cluster by name is not supported")
		} else {
			if err := l.ForEachCluster(ctx, result.Add); err != nil {
				return err
			}
		}

		return p.Fprint(out, result)
	}

	return cmd
}

func validClusterArgs(cfg Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
		directive |= cobra.ShellCompDirectiveNoFileComp
		l.forAllClusters(func(item *applications.ClusterItem) {
			if strings.HasPrefix(item.Name.String(), toComplete) {
				completions = append(completions, item.Name.String())
			}
		})
		return
	})
}
