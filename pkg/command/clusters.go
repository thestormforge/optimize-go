package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

func newClustersCommand(cfg Config) *cobra.Command {
	return &cobra.Command{
		Use:               "clusters [NAME ...]",
		Aliases:           []string{"cluster"},
		ValidArgsFunction: validClusterArgs(cfg),
	}
}

// NewGetClustersCommand returns a command for getting clusters.
func NewGetClustersCommand(cfg Config, p Printer) *cobra.Command {
	cmd := newClustersCommand(cfg)
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

// validClusterArgs returns shell completion logic for cluster names.
func validClusterArgs(cfg Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		names := make([]string, 0, 16)
		_ = l.ForEachCluster(ctx, func(item *applications.ClusterItem) error {
			if name := item.Name.String(); strings.HasPrefix(name, toComplete) {
				names = append(names, name)
			}
			return nil
		})

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
