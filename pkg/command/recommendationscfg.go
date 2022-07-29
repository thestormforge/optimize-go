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

	"github.com/spf13/cobra"

	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	"github.com/thestormforge/optimize-go/pkg/command/recommendation"
)

// NewCreateRecommendationsConfigCommand returns a new command for creating recommendation configurations.
func NewCreateRecommendationsConfigCommand(cfg Config, p Printer) *cobra.Command {
	var (
		deployConfiguration recommendation.DeployConfigurationOptions
		containerResources  recommendation.ContainerResourcesOptions
	)

	cmd := &cobra.Command{
		Use:     "recommendations-config APP_NAME",
		Aliases: []string{"recommendation-config", "rec-config", "rec-cfg", "recconfig", "reccfg"},
		Args:    cobra.ExactArgs(1),
	}

	deployConfiguration.AddFlags(cmd)
	containerResources.AddFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		appAPI := applications.NewAPI(client)

		appName := applications.ApplicationName(args[0])
		app, err := appAPI.GetApplicationByName(ctx, appName)
		if err != nil {
			return err
		}

		recommendationsURL := app.Link(api.RelationRecommendations)
		if recommendationsURL == "" {
			return fmt.Errorf("malformed response, missing recommendations link")
		}

		recs := applications.RecommendationList{}
		deployConfiguration.Apply(&recs.DeployConfiguration)
		containerResources.Apply(&recs.Configuration)

		if recs.DeployConfiguration == nil && len(recs.Configuration) == 0 {
			return nil
		}

		if err := appAPI.PatchRecommendations(ctx, recommendationsURL, recs); err != nil {
			return err
		}

		if rl, err := appAPI.ListRecommendations(ctx, recommendationsURL); err == nil {
			recs = rl
		}

		return p.Fprint(out, recs)
	}
	return cmd
}

// NewGetRecommendationsConfigCommand returns a command for getting recommendation configuration.
func NewGetRecommendationsConfigCommand(cfg Config, p Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "recommendations-config APP_NAME",
		Aliases:           []string{"recommendation-config", "rec-config", "rec-cfg", "recconfig", "reccfg"},
		ValidArgsFunction: validApplicationArgs(cfg),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		appAPI := applications.NewAPI(client)

		appName := applications.ApplicationName(args[0])
		app, err := appAPI.GetApplicationByName(ctx, appName)
		if err != nil {
			return err
		}

		recommendationsURL := app.Link(api.RelationRecommendations)
		if recommendationsURL == "" {
			return fmt.Errorf("malformed response, missing recommendations link")
		}

		rl, err := appAPI.ListRecommendations(ctx, recommendationsURL)
		if err != nil {
			return err
		}

		result := &RecommendationConfigOutput{
			Name:               app.Name.String(),
			RecommendationList: rl,
		}
		return p.Fprint(out, result)
	}
	return cmd
}
