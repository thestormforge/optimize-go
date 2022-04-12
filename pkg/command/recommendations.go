package command

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

func newRecommendationsCommand(cfg Config) *cobra.Command {
	return &cobra.Command{
		Use:     "recommendations [APPNAME/NAME ...]",
		Aliases: []string{"recommendation", "recs", "rec"},

		ValidArgsFunction: validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
			directive |= cobra.ShellCompDirectiveNoFileComp
			l.forAllApplications(func(item *applications.ApplicationItem) {
				if strings.HasPrefix(item.Name.String(), toComplete) {
					completions = append(completions, item.Name.String())
				}
			})
			return
		}),
	}
}

// NewGetRecommendationsCommand returns a command for getting recommendations.
func NewGetRecommendationsCommand(cfg Config, p Printer) *cobra.Command {
	cmd := newRecommendationsCommand(cfg)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		result := &RecommendationOutput{Items: make([]RecommendationRow, 0, len(args))}

		if err := l.ForEachNamedRecommendation(ctx, args, false, result.Add); err != nil {
			return err
		}

		return p.Fprint(out, result)
	}

	return cmd
}
