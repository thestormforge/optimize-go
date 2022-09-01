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
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	"sigs.k8s.io/yaml"
)

// NewCreateScenarioCommand returns a command for creating scenarios.
func NewCreateScenarioCommand(cfg Config, p Printer) *cobra.Command {
	var (
		title                     string
		clusters                  []string
		containerResourceSelector string
		replicaSelector           string
		goals                     []string
		perftestScenario          struct {
			testCase string
		}
		locustScenario struct {
			locustfile string
			users      int
			spawnRate  int
			runTime    time.Duration
		}
		customScenario struct {
			usePushGateway     bool
			podTemplateFile    string
			initialDelay       time.Duration
			approximateRuntime time.Duration
			image              string
		}
	)

	cmd := &cobra.Command{
		Use:     "scenario APP_NAME[/NAME]",
		Aliases: []string{"scn"},
		Args:    cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&title, "title", "", "human readable `name` for the scenario")
	cmd.Flags().StringArrayVar(&clusters, "cluster", nil, "cluster `name` used for experimentation")
	cmd.Flags().StringVar(&containerResourceSelector, "container-resource-selector", "", "`selector` for application resources which should have container resource optimization applied")
	cmd.Flags().StringVar(&replicaSelector, "replica-selector", "", "`selector` for application resources which should have replica optimization applied")
	cmd.Flags().StringSliceVar(&goals, "goals", nil, "specify the application optimization `objectives`")
	cmd.Flags().StringVar(&perftestScenario.testCase, "test-case", "", "`name` of the StormForge Performance test case to use")
	cmd.Flags().StringVar(&locustScenario.locustfile, "locustfile", "", "`file` containing the Python module to run")
	cmd.Flags().IntVar(&locustScenario.users, "locust-users", 0, "`num`ber of concurrent Locust users")
	cmd.Flags().IntVar(&locustScenario.spawnRate, "locust-spawn-rate", 0, "`rate` per second in which users are spawned")
	cmd.Flags().DurationVar(&locustScenario.runTime, "locust-run-time", 0, "stop after the specified amount of `time`")
	cmd.Flags().BoolVar(&customScenario.usePushGateway, "custom-use-push-gateway", false, "enables the Prometheus Push Gateway")
	cmd.Flags().StringVar(&customScenario.podTemplateFile, "custom-pod-template", "", "`file` containing the custom trial job pod template")
	cmd.Flags().DurationVar(&customScenario.initialDelay, "custom-initial-delay", 0, "additional `delay` before starting the trial job pod")
	cmd.Flags().DurationVar(&customScenario.approximateRuntime, "custom-approximate-runtime", 0, "the estimated amount of `time` the trial should last")
	cmd.Flags().StringVar(&customScenario.image, "custom-image", "", "override the image `name` of the first container in the trial job pod")

	// TODO The application service will not persist these values
	cmd.Flag("locustfile").Hidden = true
	cmd.Flag("locust-users").Hidden = true
	cmd.Flag("locust-spawn-rate").Hidden = true
	cmd.Flag("locust-run-time").Hidden = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		appAPI := applications.NewAPI(client)

		appName, scnName := applications.SplitScenarioName(args[0])
		app, err := appAPI.GetApplicationByName(ctx, appName)
		if err != nil {
			return err
		}

		scenariosURL := app.Link(api.RelationScenarios)
		if scenariosURL == "" {
			return fmt.Errorf("malformed response, missing scenarios link")
		}

		scn := applications.Scenario{
			DisplayName:   title,
			Configuration: []interface{}{},
			Objective:     []interface{}{},
			Clusters:      clusters,
		}

		// Only support setting the label selector on parameter configurations
		if containerResourceSelector != "" {
			scn.Configuration = append(scn.Configuration, map[string]interface{}{
				"containerResources": map[string]interface{}{
					// TODO This should be "labelSelector" but then the UI wouldn't recognize it
					"selector": containerResourceSelector,
				},
			})
		}
		if replicaSelector != "" {
			scn.Configuration = append(scn.Configuration, map[string]interface{}{
				"replicas": map[string]interface{}{
					// TODO This should be "labelSelector" but then the UI wouldn't recognize it
					"selector": replicaSelector,
				},
			})
		}

		// Only support generating named based goals
		var namedGoals []interface{}
		for _, goal := range goals {
			namedGoals = append(namedGoals, map[string]interface{}{"name": goal})
		}
		if len(namedGoals) > 0 {
			scn.Objective = append(scn.Objective, map[string]interface{}{"goals": namedGoals})
		}

		// Scenario settings
		settings := make(map[string]interface{})
		switch {
		case perftestScenario.testCase != "":
			settings["testCase"] = perftestScenario.testCase
			scn.StormForgePerformance = settings

		case locustScenario.locustfile != "":
			switch strings.ToLower(strings.SplitN(locustScenario.locustfile, ":", 2)[0]) {
			case "http", "https":
				//	The file name field can be a URL
				settings["locustfile"] = locustScenario.locustfile
			default:
				// The file contents can be inlined in the file name field
				data, err := os.ReadFile(locustScenario.locustfile)
				if err != nil {
					return err
				}
				settings["locustfile"] = string(data)
			}
			if locustScenario.users > 0 {
				settings["users"] = locustScenario.users
			}
			if locustScenario.spawnRate > 0 {
				settings["spawnRate"] = locustScenario.spawnRate
			}
			if locustScenario.runTime > 0 {
				settings["runTime"] = locustScenario.runTime.String()
			}
			scn.Locust = settings

		default:
			if customScenario.podTemplateFile != "" {
				data, err := os.ReadFile(customScenario.podTemplateFile)
				if err != nil {
					return err
				}

				var podTemplate map[string]interface{}
				if err := yaml.Unmarshal(data, &podTemplate); err != nil {
					return err
				}

				if _, ok := podTemplate["spec"]; !ok {
					return fmt.Errorf("pod template should have a 'spec' field")
				}

				settings["podTemplate"] = podTemplate
			}
			if customScenario.usePushGateway {
				settings["pushGateway"] = customScenario.usePushGateway
			}
			if customScenario.initialDelay > 0 {
				settings["initialDelaySeconds"] = int(customScenario.initialDelay.Round(time.Second).Seconds())
			}
			if customScenario.approximateRuntime > 0 {
				settings["approximateRuntimeSeconds"] = int(customScenario.approximateRuntime.Round(time.Second).Seconds())
			}
			if customScenario.image != "" {
				settings["image"] = customScenario.image
			}
			if len(settings) > 0 {
				scn.Custom = settings
			}
		}

		var selfURL string
		if scnName != "" {
			md, err := appAPI.CreateScenarioByName(ctx, scenariosURL, scnName, scn)
			if err != nil {
				return err
			}
			selfURL = md.Link(api.RelationSelf)
		} else {
			md, err := appAPI.CreateScenario(ctx, scenariosURL, scn)
			if err != nil {
				return err
			}
			selfURL = md.Location()
		}

		// Fetch the scenario back for display
		if selfURL != "" {
			if s, err := appAPI.GetScenario(ctx, selfURL); err == nil {
				scn = s
			}
		}

		return p.Fprint(out, NewScenarioRow(&applications.ScenarioItem{Scenario: scn}))
	}
	return cmd
}

// NewEditScenarioCommand returns a command for editing a scenario.
func NewEditScenarioCommand(cfg Config, p Printer) *cobra.Command {
	var (
		title    string
		clusters []string
	)

	cmd := &cobra.Command{
		Use:     "scenario APP_NAME/NAME",
		Aliases: []string{"scn"},
		Args:    cobra.ExactArgs(1),
	}

	cmd.Flags().StringVar(&title, "title", "", "human readable `name` for the scenario")
	cmd.Flags().StringArrayVar(&clusters, "cluster", nil, "cluster `name` used for experimentation")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := applications.Lister{
			API: applications.NewAPI(client),
		}

		return l.ForEachNamedScenario(ctx, args, false, func(item *applications.ScenarioItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			scn := applications.Scenario{
				DisplayName: title,
				Clusters:    nil,
			}

			if scn.DisplayName == "" {
				return nil
			}

			if err := l.API.PatchScenario(ctx, selfURL, scn); err != nil {
				return err
			}
			return p.Fprint(out, NewScenarioRow(item))
		})
	}
	return cmd
}

// NewGetScenariosCommand returns a command for getting scenarios.
func NewGetScenariosCommand(cfg Config, p Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scenarios APP_NAME | APP_NAME/NAME ...",
		Aliases: []string{"scenario", "scn"},
		Args:    cobra.MinimumNArgs(1),
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

		result := &ScenarioOutput{Items: make([]ScenarioRow, 0, len(args))}
		if err := l.ForEachNamedScenario(ctx, args, false, result.Add); err != nil {
			return err
		}

		return p.Fprint(out, result)
	}
	return cmd
}

// NewDeleteScenariosCommand returns a command for deleting scenarios.
func NewDeleteScenariosCommand(cfg Config, p Printer) *cobra.Command {
	var (
		ignoreNotFound bool
	)

	cmd := &cobra.Command{
		Use:     "scenarios APP_NAME | APP_NAME/NAME ...",
		Aliases: []string{"scenario", "scn"},
		Args:    cobra.MinimumNArgs(1),
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

		return l.ForEachNamedScenario(ctx, args, ignoreNotFound, func(item *applications.ScenarioItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			if err := l.API.DeleteScenario(ctx, selfURL); err != nil {
				return err
			}

			return p.Fprint(out, NewScenarioRow(item))
		})
	}
	return cmd
}
