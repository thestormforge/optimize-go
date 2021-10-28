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

package v1alpha1

import (
	"context"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// Lister is a helper to individually visit all items in a list (even across page boundaries).
type Lister struct {
	// API is the Experiment API used to fetch objects.
	API API
	// BatchSize overrides the default batch size for fetching lists.
	BatchSize int
}

// ForEachExperiment iterates over all the experiments matching the supplied query.
func (l *Lister) ForEachExperiment(ctx context.Context, q ExperimentListQuery, f func(*ExperimentItem) error) error {
	// Define a helper to iteratively (NOT recursively) visit experiments
	forEach := func(lst ExperimentList, err error) (string, error) {
		if err != nil {
			return "", err
		}

		for i := range lst.Experiments {
			if err := f(&lst.Experiments[i]); err != nil {
				return "", err
			}
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}

		return lst.Link(api.RelationNext), nil
	}

	// Overwrite the limit
	if l.BatchSize > 0 {
		q.SetLimit(l.BatchSize)
	}

	// Iterate over all experiments, starting with first page
	u, err := forEach(l.API.GetAllExperiments(ctx, q))
	for u != "" && err == nil {
		u, err = forEach(l.API.GetAllExperimentsByPage(ctx, u))
	}
	return err
}

// ForEachTrial iterates over all trials for an experiment matching the supplied query.
func (l *Lister) ForEachTrial(ctx context.Context, exp *Experiment, q TrialListQuery, f func(*TrialItem) error) (err error) {
	// Define a helper to iteratively (NOT recursively) list and visit scenarios
	forEach := func(u string) (string, error) {
		lst, err := l.API.GetAllTrials(ctx, u, q)
		if err != nil {
			return "", err
		}

		for i := range lst.Trials {
			if err := f(&lst.Trials[i]); err != nil {
				return "", err
			}
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}

		return lst.Link(api.RelationNext), nil
	}

	// Overwrite the limit
	if l.BatchSize > 0 {
		q.SetLimit(l.BatchSize)
	}

	// Iterate over all trial pages, starting with the experiment's "rel=trials"
	u := exp.Link(api.RelationTrials)
	for u != "" && err == nil {
		u, err = forEach(u)
	}
	return
}
