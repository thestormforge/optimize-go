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
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
)

// NewCreateTrialCommand returns a command for creating a trial.
func NewCreateTrialCommand(cfg Config, p Printer) *cobra.Command {
	var (
		assignments     map[string]string
		defaultBehavior string
	)

	cmd := &cobra.Command{
		Use:  "trial EXP_NAME",
		Args: cobra.ExactArgs(1),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		expAPI := experiments.NewAPI(client)

		exp, err := expAPI.GetExperimentByName(ctx, experiments.ExperimentName(args[0]))
		if err != nil {
			return err
		}

		trialsURL := exp.Link(api.RelationTrials)
		if trialsURL == "" {
			return fmt.Errorf("malformed response, missing trials link")
		}

		t := experiments.TrialItem{}
		for _, p := range exp.Parameters {
			v, err := parameterValue(&p, assignments, defaultBehavior)
			if err != nil {
				return err
			}
			if v == nil {
				return fmt.Errorf("no assignment for parameter %q", p.Name)
			}
			if err := experiments.CheckParameterValue(&p, v); err != nil {
				return err
			}
			t.Assignments = append(t.Assignments, experiments.Assignment{ParameterName: p.Name, Value: *v})
		}

		if err := experiments.CheckParameterConstraints(t.Assignments, exp.Constraints); err != nil {
			return err
		}

		if _, err := expAPI.CreateTrial(ctx, trialsURL, t.TrialAssignments); err != nil {
			return err
		}

		// Abuse TrialOutput to help with formatting
		// NOTE: The trial number will not exist until the assignments have been pull from the queue
		o := TrialOutput{}
		_ = o.Add(&t)
		return p.Fprint(out, o.Items[0])
	}

	cmd.Flags().StringToStringVarP(&assignments, "assign", "A", nil, "assign an explicit `key=value` to a parameter")
	cmd.Flags().StringVar(&defaultBehavior, "default", "", "select the `behavior` for default values; one of: none|min|max|rand")

	return cmd
}

// NewEditTrialCommand returns a command for editing a trial.
func NewEditTrialCommand(cfg Config, p Printer) *cobra.Command {
	var (
		labels map[string]string
	)

	cmd := &cobra.Command{
		Use:               "trial EXP_NAME/TRIAL_NUM",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: validTrialArgs(cfg),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := experiments.Lister{
			API: experiments.NewAPI(client),
		}

		q := experiments.TrialListQuery{}
		q.SetStatus(experiments.TrialCompleted)
		return l.ForEachNamedTrial(ctx, args, q, false, func(item *experiments.TrialItem) error {
			// Apply label changes
			if len(labels) > 0 {
				labelsURL := item.Link(api.RelationLabels)
				if labelsURL == "" {
					return fmt.Errorf("malformed response, missing labels link")
				}

				err = l.API.LabelTrial(ctx, labelsURL, experiments.TrialLabels{Labels: labels})
				if err != nil {
					return err
				}
			}

			return p.Fprint(out, item)
		})
	}

	cmd.Flags().StringToStringVar(&labels, "set-label", nil, "label `key=value` pairs to assign")

	return cmd
}

// NewGetTrialsCommand returns a command for getting trials.
func NewGetTrialsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		selector string
		all      bool
	)

	cmd := &cobra.Command{
		Use:               "trials [EXP_NAME/TRIAL_NUM ...]",
		Aliases:           []string{"trial"},
		ValidArgsFunction: validTrialArgs(cfg),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := experiments.Lister{
			API: experiments.NewAPI(client),
		}

		result := &TrialOutput{Items: make([]TrialRow, 0, len(args))}

		q := experiments.TrialListQuery{}
		q.SetLabelSelector(parseLabelSelector(selector))
		q.SetStatus(experiments.TrialActive, experiments.TrialCompleted, experiments.TrialFailed)
		if all {
			q.AddStatus(experiments.TrialStaged)
		}

		if err := l.ForEachNamedTrial(ctx, args, q, false, result.Add); err != nil {
			return err
		}

		return p.Fprint(out, result)
	}

	cmd.Flags().StringVarP(&selector, "selector", "l", selector, "selector (label `query`) to filter on")
	cmd.Flags().BoolVarP(&all, "all", "A", all, "include all resources")

	return cmd
}

// NewDeleteTrialsCommand returns a command for deleting ("abandoning") trials.
func NewDeleteTrialsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		ignoreNotFound bool
	)

	cmd := &cobra.Command{
		Use:               "trials [EXP_NAME/TRIAL_NUM ...]",
		Aliases:           []string{"trial"},
		ValidArgsFunction: validTrialArgs(cfg),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := experiments.Lister{
			API: experiments.NewAPI(client),
		}

		q := experiments.TrialListQuery{}
		q.SetStatus(experiments.TrialActive)
		return l.ForEachNamedTrial(ctx, args, q, ignoreNotFound, func(item *experiments.TrialItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			err = l.API.AbandonRunningTrial(ctx, selfURL)
			if err != nil {
				return err
			}

			return p.Fprint(out, item)
		})
	}

	return cmd
}

func validTrialArgs(cfg Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
		directive |= cobra.ShellCompDirectiveNoFileComp
		l.forAllExperiments(func(item *experiments.ExperimentItem) {
			if strings.HasPrefix(item.Name.String(), toComplete) {
				completions = append(completions, item.Name.String())
			}
		})

		if len(completions) == 1 && completions[0] == toComplete {
			completions[0] += "-"
			directive |= cobra.ShellCompDirectiveNoSpace
		}

		return
	})
}

func parameterValue(p *experiments.Parameter, assignments map[string]string, defaultBehavior string) (*api.NumberOrString, error) {
	if a, ok := assignments[p.Name]; ok {
		return p.ParseValue(a)
	}

	switch defaultBehavior {
	case "none", "":
		return nil, nil
	case "min", "minimum":
		return p.LowerBound()
	case "max", "maximum":
		return p.UpperBound()
	case "rand", "random":
		return p.RandomValue()
	default:
		return nil, fmt.Errorf("unknown default behavior %q", defaultBehavior)
	}
}
