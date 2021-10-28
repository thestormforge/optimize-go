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

package v2

import (
	"context"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// Lister is a helper to individually visit all items in a list (even across page boundaries).
type Lister struct {
	// API is the Application API used to fetch objects.
	API API
	// BatchSize overrides the default batch size for fetching lists.
	BatchSize int
}

// ForEachApplication iterates over all the applications matching the supplied query.
func (l *Lister) ForEachApplication(ctx context.Context, q ApplicationListQuery, f func(*ApplicationItem) error) error {
	// Define a helper to iteratively (NOT recursively) visit applications
	forEach := func(lst ApplicationList, err error) (string, error) {
		if err != nil {
			return "", err
		}

		for i := range lst.Applications {
			if err := f(&lst.Applications[i]); err != nil {
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

	// Iterate over all applications, starting with first page
	u, err := forEach(l.API.ListApplications(ctx, q))
	for u != "" && err == nil {
		u, err = forEach(l.API.ListApplicationsByPage(ctx, u))
	}
	return err
}

// ForEachScenario iterates over all scenarios for an application matching the supplied query.
func (l *Lister) ForEachScenario(ctx context.Context, app *Application, q ScenarioListQuery, f func(*ScenarioItem) error) (err error) {
	// Define a helper to iteratively (NOT recursively) list and visit scenarios
	forEach := func(u string) (string, error) {
		lst, err := l.API.ListScenarios(ctx, u, q)
		if err != nil {
			return "", err
		}

		for i := range lst.Scenarios {
			if err := f(&lst.Scenarios[i]); err != nil {
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

	// Iterate over all scenario pages, starting with the application's "rel=scenarios"
	u := app.Link(api.RelationScenarios)
	for u != "" && err == nil {
		u, err = forEach(u)
	}
	return
}
