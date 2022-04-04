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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
)

func newTrialsCommand(cfg Config) *cobra.Command {
	return &cobra.Command{
		Use:     "trials [NAME ...]",
		Aliases: []string{"trial"},

		// Trial names start with experiment names, so we can reuse the completion code
		ValidArgsFunction: validExperimentArgs(cfg, "-"),
	}
}

// NewGetTrialsCommand returns a command for getting trials.
func NewGetTrialsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		selector string
		all      bool
	)

	cmd := newTrialsCommand(cfg)
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

		if err := forExperimentTrials(ctx, &l, q, parseTrialArgs(args), result.Add); err != nil {
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
	cmd := newTrialsCommand(cfg)
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
		return forExperimentTrials(ctx, &l, q, parseTrialArgs(args), func(item *experiments.TrialItem) error {
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

// NewLabelTrialsCommand returns a command for labeling trials.
func NewLabelTrialsCommand(cfg Config, p Printer) *cobra.Command {
	cmd := newTrialsCommand(cfg)
	// TODO Should we extend validargsfn with suggestions like `baseline=true` ?
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
		names, labels := argsToNamesAndLabels(args)
		return forExperimentTrials(ctx, &l, q, parseTrialArgs(names), func(item *experiments.TrialItem) error {
			labelsURL := item.Link(api.RelationLabels)
			if labelsURL == "" {
				return fmt.Errorf("malformed response, missing labels link")
			}

			err = l.API.LabelTrial(ctx, labelsURL, experiments.TrialLabels{Labels: labels})
			if err != nil {
				return err
			}

			return p.Fprint(out, item)
		})
	}

	return cmd
}

// experimentTrials is an experiment name and a list of trial numbers. Because trials cannot
// currently be fetched individually, we only want to fetch the full trial list once and then
// iterate over that for all the trials that were selected.
type experimentTrials struct {
	name    experiments.ExperimentName
	numbers []int64
}

// filter decorates the supplied function to ensure it is only invoked on trials
// whose number is included. The function is unchanged if the current number list is empty.
func (et *experimentTrials) filter(f func(item *experiments.TrialItem) error) func(item *experiments.TrialItem) error {
	if len(et.numbers) == 0 {
		return f
	}

	return func(item *experiments.TrialItem) error {
		for _, num := range et.numbers {
			if item.Number == num {
				return f(item)
			}
		}
		return nil
	}
}

// parseTrialArgs aggregates the trial numbers for each experiment to prevent us
// from fetching the trial lists multiple times.
func parseTrialArgs(args []string) []experimentTrials {
	trials := make([]experimentTrials, 0, len(args))
	for _, arg := range args {
		name, number := experiments.SplitTrialName(arg)

		var inv *experimentTrials
		for i := range trials {
			if trials[i].name == name {
				inv = &trials[i]
			}
		}
		if inv == nil {
			trials = append(trials, experimentTrials{name: name})
			inv = &trials[len(trials)-1]
		}

		if number >= 0 {
			inv.numbers = append(inv.numbers, number)
		}
	}
	return trials
}

// forExperimentTrials iterates over the experiment trials, fetching the distinct experiments and trial lists once.
func forExperimentTrials(ctx context.Context, l *experiments.Lister, q experiments.TrialListQuery, trials []experimentTrials, f func(item *experiments.TrialItem) error) error {
	for _, et := range trials {
		exp, err := l.API.GetExperimentByName(ctx, et.name)
		if err != nil {
			return err
		}

		if err := l.ForEachTrial(ctx, &exp, q, et.filter(f)); err != nil {
			return err
		}
	}
	return nil
}
