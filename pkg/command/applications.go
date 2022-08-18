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
	"github.com/thestormforge/optimize-go/pkg/command/recommendation"
)

// NewCreateApplicationCommand returns a command for creating applications.
func NewCreateApplicationCommand(cfg Config, p Printer) *cobra.Command {
	var (
		title    string
		resource applications.Resource
	)

	cmd := &cobra.Command{
		Use:     "application [NAME]",
		Aliases: []string{"app"},
		Args:    cobra.MaximumNArgs(1),
	}

	cmd.Flags().StringVar(&title, "title", "", "human readable `name` for the application")
	cmd.Flags().StringArrayVar(&resource.Kubernetes.Namespaces, "namespace", nil, "select application resources from a specific `namespace`")
	cmd.Flags().StringVar(&resource.Kubernetes.NamespaceSelector, "ns-selector", "", "`sel`ect application resources from labeled namespaces")
	cmd.Flags().StringVarP(&resource.Kubernetes.Selector, "selector", "l", "", "`sel`ect only labeled application resources")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		appAPI := applications.NewAPI(client)

		// Construct the application we want to create
		app := applications.Application{
			DisplayName: title,
		}

		if r, ok := normalizeResource(resource); ok {
			app.Resources = append(app.Resources, r)
		}

		// Upsert the application if we have a name, otherwise create it with a generated name
		var selfURL string
		if len(args) > 0 && args[0] != "" {
			name := applications.ApplicationName(args[0])
			md, err := appAPI.UpsertApplicationByName(ctx, name, app)
			if err != nil {
				return err
			}
			selfURL = md.Link(api.RelationSelf)
		} else {
			md, err := appAPI.CreateApplication(ctx, app)
			if err != nil {
				return err
			}
			selfURL = md.Location()
		}

		// Fetch the application back for display
		if selfURL != "" {
			if a, err := appAPI.GetApplication(ctx, selfURL); err == nil {
				app = a
			}
		}

		return p.Fprint(out, &app)
	}
	return cmd
}

// NewEditApplicationCommand returns a command for editing an application.
func NewEditApplicationCommand(cfg Config, p Printer) *cobra.Command {
	var (
		title    string
		resource applications.Resource
	)

	cmd := &cobra.Command{
		Use:               "application NAME",
		Aliases:           []string{"app"},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: validApplicationArgs(cfg),
	}

	cmd.Flags().StringVar(&title, "title", "", "human readable `name` for the application")
	cmd.Flags().StringArrayVar(&resource.Kubernetes.Namespaces, "namespace", nil, "select application resources from a specific `namespace`")
	cmd.Flags().StringVar(&resource.Kubernetes.NamespaceSelector, "ns-selector", "", "`sel`ect application resources from labeled namespaces")
	cmd.Flags().StringVarP(&resource.Kubernetes.Selector, "selector", "l", "", "`sel`ect only labeled application resources")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		return l.ForEachNamedApplication(ctx, args, false, func(item *applications.ApplicationItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			var needsUpdate bool

			// Update the title
			if title != "" {
				item.Application.DisplayName = title
				needsUpdate = true
			}

			// Update the resource
			if r, ok := normalizeResource(resource); ok {
				if len(item.Application.Resources) > 0 {
					item.Application.Resources[0] = r
				} else {
					item.Application.Resources = append(item.Application.Resources, r)
				}
				needsUpdate = true
			}

			if !needsUpdate {
				return nil
			}

			if _, err := l.API.UpsertApplication(ctx, selfURL, item.Application); err != nil {
				return err
			}
			return p.Fprint(out, item)
		})
	}
	return cmd
}

// NewEnableApplicationRecommendationsCommand returns a new command for enabling recommendations.
func NewEnableApplicationRecommendationsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		deployConfiguration recommendation.DeployConfigurationOptions
		containerResources  recommendation.ContainerResourcesOptions
	)

	cmd := &cobra.Command{
		Use:               "application-recommendations APP_NAME",
		Aliases:           []string{"app-recs", "recs"},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: validApplicationArgs(cfg),
	}

	deployConfiguration.AddFlags(cmd)
	containerResources.AddFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("cluster", validClusterArgs(cfg, applications.ClusterRecommendations))

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

		recs, err := appAPI.ListRecommendations(ctx, recommendationsURL)
		if err != nil {
			return err
		}

		patch := applications.RecommendationList{}
		deployConfiguration.Apply(&patch.DeployConfiguration)
		containerResources.Apply(&patch.Configuration)
		if err := recommendation.Finish(cmd, appAPI, app, recs, &patch); err != nil {
			return err
		}

		if err := appAPI.PatchRecommendations(ctx, recommendationsURL, patch); err != nil {
			return err
		}

		if rl, err := appAPI.ListRecommendations(ctx, recommendationsURL); err == nil {
			recs = rl
		}

		// Re-use the application list output to generate an ApplicationRow
		result := &ApplicationOutput{}
		if err := result.Add(&applications.ApplicationItem{Application: app}); err != nil {
			return err
		}
		result.Items[0].SetRecommendationsDeployConfig(recs.DeployConfiguration)
		result.Items[0].SetRecommendationsConfiguration(recs.Configuration)
		return p.Fprint(out, &result.Items[0])
	}
	return cmd
}

// NewDisableApplicationRecommendationsCommand returns a new command for disabling recommendations.
func NewDisableApplicationRecommendationsCommand(cfg Config, p Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "application-recommendations APP_NAME",
		Aliases:           []string{"app-recs", "recs"},
		Args:              cobra.ExactArgs(1),
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

		// Construct a patch to disable recommendations
		patch := applications.RecommendationList{
			DeployConfiguration: &applications.DeployConfiguration{
				Mode: applications.RecommendationsDisabled,
			},
		}
		if err := appAPI.PatchRecommendations(ctx, recommendationsURL, patch); err != nil {
			return err
		}

		// Re-use the application list output to generate an ApplicationRow
		result := &ApplicationOutput{}
		if err := result.Add(&applications.ApplicationItem{Application: app}); err != nil {
			return err
		}
		return p.Fprint(out, &result.Items[0])
	}
	return cmd
}

// NewGetApplicationsCommand returns a command for getting applications.
func NewGetApplicationsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		product   string
		batchSize int
	)

	cmd := &cobra.Command{
		Use:               "applications [NAME ...]",
		Aliases:           []string{"application", "apps", "app"},
		ValidArgsFunction: validApplicationArgs(cfg),
	}

	cmd.Flags().StringVar(&product, "for", product, "show only clusters for a specific `product`; one of: optimize-pro|optimize-live")
	cmd.Flags().IntVar(&batchSize, "batch-size", batchSize, "fetch large lists in chu`n`ks rather then all at once")

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

			result.Items[i].SetRecommendationsDeployConfig(rl.DeployConfiguration)
			result.Items[i].SetRecommendationsConfiguration(rl.Configuration)
		}

		// Filter applications by product
		if product != "" {
			items := make([]ApplicationRow, 0, len(result.Items))
			for i := range result.Items {
				isPro := result.Items[i].ScenarioCount > 0
				isLive := result.Items[i].Recommendations != applications.RecommendationsDisabled

				// Only skip applications if we know it is one or the other
				if isPro || isLive {
					switch product {
					case "optimize-pro", "pro":
						if !isPro {
							continue
						}
					case "optimize-live", "live":
						if !isLive {
							continue
						}
					}
				}

				items = append(items, result.Items[i])
			}
			result.Items = items
		}

		return p.Fprint(out, result)
	}
	return cmd
}

// NewDeleteApplicationsCommand returns a command for deleting applications.
func NewDeleteApplicationsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		ignoreNotFound bool
	)

	cmd := &cobra.Command{
		Use:               "applications [NAME ...]",
		Aliases:           []string{"application", "apps", "app"},
		ValidArgsFunction: validApplicationArgs(cfg),
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
	return cmd
}

func validApplicationArgs(cfg Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
		directive |= cobra.ShellCompDirectiveNoFileComp
		l.forAllApplications(func(item *applications.ApplicationItem) {
			if strings.HasPrefix(item.Name.String(), toComplete) {
				completions = append(completions, item.Name.String())
			}
		})
		return
	})
}

func normalizeResource(r applications.Resource) (applications.Resource, bool) {
	if r.Kubernetes.Namespace == "" && len(r.Kubernetes.Namespaces) == 0 && r.Kubernetes.NamespaceSelector == "" {
		return r, false
	}

	if r.Kubernetes.Namespace == "" && len(r.Kubernetes.Namespaces) == 1 {
		r.Kubernetes.Namespace = r.Kubernetes.Namespaces[0]
		r.Kubernetes.Namespaces = nil
	}

	if r.Kubernetes.Namespace != "" && len(r.Kubernetes.Namespaces) > 0 {
		r.Kubernetes.Namespaces = append(r.Kubernetes.Namespaces, r.Kubernetes.Namespace)
		r.Kubernetes.Namespace = ""
	}

	return r, true
}
