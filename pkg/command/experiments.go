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

func newExperimentsCommand(cfg Config) *cobra.Command {
	return &cobra.Command{
		Use:     "experiments [NAME ...]",
		Aliases: []string{"experiment", "exps", "exp"},

		ValidArgsFunction: validArgs(cfg, func(l *completionLister, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
			directive |= cobra.ShellCompDirectiveNoFileComp
			l.forEachExperiment(func(item *experiments.ExperimentItem) {
				if strings.HasPrefix(item.Name.String(), toComplete) {
					completions = append(completions, item.Name.String())
				}
			})
			return
		}),
	}
}

// NewGetExperimentsCommand returns a command for getting experiments.
func NewGetExperimentsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		batchSize int
		selector  string
	)

	cmd := newExperimentsCommand(cfg)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := experiments.Lister{
			API:       experiments.NewAPI(client),
			BatchSize: batchSize,
		}

		result := &ExperimentOutput{Items: make([]ExperimentRow, 0, len(args))}
		if len(args) > 0 {
			if err := l.ForEachNamedExperiment(ctx, args, false, result.Add); err != nil {
				return err
			}
		} else {
			q := experiments.ExperimentListQuery{}
			q.SetLabelSelector(parseLabelSelector(selector))
			if err := l.ForEachExperiment(ctx, q, result.Add); err != nil {
				return err
			}
		}

		return p.Fprint(out, result)
	}

	cmd.Flags().IntVar(&batchSize, "batch-size", batchSize, "fetch large lists in chu`n`ks rather then all at once")
	cmd.Flags().StringVarP(&selector, "selector", "l", selector, "selector (label `query`) to filter on")

	return cmd
}

// NewDeleteExperimentsCommand returns a command for deleting experiments.
func NewDeleteExperimentsCommand(cfg Config, p Printer) *cobra.Command {
	var (
		ignoreNotFound bool
	)

	cmd := newExperimentsCommand(cfg)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := experiments.Lister{
			API: experiments.NewAPI(client),
		}

		return l.ForEachNamedExperiment(ctx, args, ignoreNotFound, func(item *experiments.ExperimentItem) error {
			selfURL := item.Link(api.RelationSelf)
			if selfURL == "" {
				return fmt.Errorf("malformed response, missing self link")
			}

			if err := l.API.DeleteExperiment(ctx, selfURL); err != nil {
				return err
			}

			return p.Fprint(out, item)
		})
	}

	cmd.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", ignoreNotFound, "treat not found errors as successful deletes")

	return cmd
}

// NewLabelExperimentsCommand returns a command for labeling experiments.
func NewLabelExperimentsCommand(cfg Config, p Printer) *cobra.Command {
	cmd := newExperimentsCommand(cfg)
	cmd.Use += " KEY=VAL ..."
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return err
		}

		l := experiments.Lister{
			API: experiments.NewAPI(client),
		}

		names, labels := argsToNamesAndLabels(args)
		return l.ForEachNamedExperiment(ctx, names, false, func(item *experiments.ExperimentItem) error {
			labelsURL := item.Link(api.RelationLabels)
			if labelsURL == "" {
				return fmt.Errorf("malformed response, missing labels link")
			}

			if err := l.API.LabelExperiment(ctx, labelsURL, experiments.ExperimentLabels{Labels: labels}); err != nil {
				return err
			}

			return p.Fprint(out, item)
		})
	}

	return cmd
}
