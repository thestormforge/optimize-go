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
	"strings"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
)

// Config represents the configuration necessary to run a command.
type Config interface {
	// Address returns the base address for the API endpoints.
	Address() string
}

// parseLabelSelector returns a map of simple equality based label selectors.
func parseLabelSelector(s string) map[string]string {
	if s == "" {
		return nil
	}

	// Note: we used to use the Kubernetes code to implement matching on the client side,
	// this code implements a significantly reduced set of simple AND'd equals expressions
	selector := make(map[string]string)
	for _, sel := range strings.Split(s, ",") {
		pair := strings.SplitN(sel, "=", 2)
		var value string
		if len(pair) > 1 {
			value = strings.TrimSpace(pair[1])
		}
		selector[strings.TrimSpace(pair[0])] = value
	}

	return selector
}

// argsToNamesAndLabels returns a list of names and label mappings from an argument stream.
func argsToNamesAndLabels(args []string) ([]string, map[string]string) {
	names := make([]string, 0, len(args))
	labels := make(map[string]string, len(args))

	for _, arg := range args {
		if p := strings.SplitN(arg, "=", 2); len(p) == 2 {
			labels[p[0]] = p[1]
		} else if p := strings.TrimSuffix(arg, "-"); p != arg {
			labels[p] = ""
		} else {
			names = append(names, arg)
		}
	}

	return names, labels
}

func validArgs(cfg Config, f func(*completionLister, string) ([]string, cobra.ShellCompDirective)) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := api.NewClient(cfg.Address(), nil)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return f(&completionLister{ctx: cmd.Context(), client: client}, toComplete)
	}
}

// completionLister is a helper for creating lists used for completions.
type completionLister struct {
	ctx    context.Context
	client api.Client
}

// forEachApplication lists all applications, ignoring errors.
func (c *completionLister) forAllApplications(f func(item *applications.ApplicationItem)) {
	l := applications.Lister{API: applications.NewAPI(c.client)}
	_ = l.ForEachApplication(c.ctx, applications.ApplicationListQuery{}, func(item *applications.ApplicationItem) error {
		f(item)
		return nil
	})
}

// forEachExperiment lists all experiments, ignoring errors.
func (c *completionLister) forAllExperiments(f func(item *experiments.ExperimentItem)) {
	l := experiments.Lister{API: experiments.NewAPI(c.client)}
	_ = l.ForEachExperiment(c.ctx, experiments.ExperimentListQuery{}, func(item *experiments.ExperimentItem) error {
		f(item)
		return nil
	})
}

// forEachCluster lists all cluster, ignoring errors.
func (c *completionLister) forAllClusters(f func(item *applications.ClusterItem)) {
	l := applications.Lister{API: applications.NewAPI(c.client)}
	_ = l.ForEachCluster(c.ctx, func(item *applications.ClusterItem) error {
		f(item)
		return nil
	})
}
