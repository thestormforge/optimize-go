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
		Use:     "clusters [NAME ...]",
		Aliases: []string{"cluster"},

		ValidArgsFunction: validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
			directive |= cobra.ShellCompDirectiveNoFileComp
			l.forAllClusters(func(item *applications.ClusterItem) {
				if strings.HasPrefix(item.Name.String(), toComplete) {
					completions = append(completions, item.Name.String())
				}
			})
			return
		}),
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
