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
	"errors"
	"fmt"
	"sort"

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

// ForEachNamedExperiment iterates over all the named experiments, optionally ignoring those that do not exist.
func (l *Lister) ForEachNamedExperiment(ctx context.Context, names []string, ignoreNotFound bool, f func(*ExperimentItem) error) error {
	for _, name := range names {
		exp, err := l.API.GetExperimentByName(ctx, ExperimentName(name))
		if err != nil {
			var notFoundErr *api.Error
			if errors.As(err, &notFoundErr) && notFoundErr.Type == ErrExperimentNotFound && ignoreNotFound {
				continue
			}
			return err
		}

		if err := f(&ExperimentItem{Experiment: exp}); err != nil {
			return err
		}
	}
	return nil
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

		// Reset the query so it is only used once
		q = TrialListQuery{}
	}
	return
}

// ForEachNamedTrial iterates over all the named trials, optionally ignoring those that do not exist.
func (l *Lister) ForEachNamedTrial(ctx context.Context, names []string, q TrialListQuery, ignoreNotFound bool, f func(*TrialItem) error) error {
	// Overwrite the limit
	if l.BatchSize > 0 {
		q.SetLimit(l.BatchSize)
	}

	cache := make(map[ExperimentName]map[int64]*TrialItem)
	for _, n := range names {
		expName, trialNum := SplitTrialName(n)

		// There is no reliable way to get the per-trial addresses, just load
		// all the trials into memory the first time we see the experiment
		if _, ok := cache[expName]; !ok {
			exp, err := l.API.GetExperimentByName(ctx, expName)
			if err != nil {
				return err
			}

			cache[expName] = make(map[int64]*TrialItem)
			if err := l.ForEachTrial(ctx, &exp, q, func(item *TrialItem) error {
				cache[expName][item.Number] = item
				return nil
			}); err != nil {
				return err
			}
		}

		// If there was no trial number, emit all trials in descending order
		if trialNum < 0 {
			result := make([]*TrialItem, 0, len(cache[expName]))
			for _, t := range cache[expName] {
				result = append(result, t)
			}
			sort.Slice(result, func(i, j int) bool { return result[i].Number > result[j].Number })
			for _, r := range result {
				if err := f(r); err != nil {
					return err
				}
			}
			return nil
		}

		// Get the trial out of the trial cache
		if t, ok := cache[expName][trialNum]; ok {
			if err := f(t); err != nil {
				return err
			}
		} else if !ignoreNotFound {
			return &api.Error{Type: ErrTrialNotFound, Message: fmt.Sprintf("trial not found: %q", n)}
		}
	}
	return nil
}
