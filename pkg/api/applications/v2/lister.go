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
	"errors"
	"fmt"
	"sort"
	"strings"

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

// ForEachNamedApplication iterates over all the named applications, optionally ignoring those that do not exist.
func (l *Lister) ForEachNamedApplication(ctx context.Context, names []string, ignoreNotFound bool, f func(item *ApplicationItem) error) error {
	for _, name := range names {
		app, err := l.API.GetApplicationByName(ctx, ApplicationName(name))
		if err != nil {
			var notFoundErr *api.Error
			if errors.As(err, &notFoundErr) && notFoundErr.Type == ErrApplicationNotFound && ignoreNotFound {
				continue
			}
			return err
		}

		if err := f(&ApplicationItem{Application: app}); err != nil {
			return err
		}
	}
	return nil
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

		// Reset the query so it is only used once
		q = ScenarioListQuery{}
	}
	return
}

// ForEachRecommendation iterates over all the recommendations for an application.
func (l *Lister) ForEachRecommendation(ctx context.Context, app *Application, f func(item *RecommendationItem) error) (err error) {
	// Define a helper to iteratively (NOT recursively) list and visit recommendations
	forEach := func(u string) (string, error) {
		lst, err := l.API.ListRecommendations(ctx, u)
		if err != nil {
			return "", err
		}

		for i := range lst.Recommendations {
			if err := f(&lst.Recommendations[i]); err != nil {
				return "", err
			}
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}

		return lst.Link(api.RelationNext), nil
	}

	// Iterate over all scenario pages, starting with the application's "rel=scenarios"
	u := app.Link(api.RelationRecommendations)
	for u != "" && err == nil {
		u, err = forEach(u)
	}
	return
}

// ForEachNamedRecommendation iterates over all the named recommendations, optionally ignoring those that do not exist.
func (l *Lister) ForEachNamedRecommendation(ctx context.Context, names []string, ignoreNotFound bool, f func(item *RecommendationItem) error) error {
	cache := make(map[ApplicationName]map[string]string)
	for _, name := range names {
		appName, recName := SplitRecommendationName(name)

		// Unlike trials, the recommendation index is incomplete: there is more
		// information available on the individual resources. However: the only
		// way to safely traverse to the recommendation is via the index. Here
		// we are just building up the list of URLs to the recommendations.
		if _, ok := cache[appName]; !ok {
			app, err := l.API.GetApplicationByName(ctx, appName)
			if err != nil {
				return err
			}

			cache[appName] = make(map[string]string)
			if err := l.ForEachRecommendation(ctx, &app, func(item *RecommendationItem) error {
				cache[appName][item.Name] = item.Link(api.RelationSelf)
				return nil
			}); err != nil {
				return err
			}

			// HACK: Because the index contains so few entries, if we have a
			// recommendation name, we are going to take a guess at the URL
			// while we have the application in scope. Typically, this is strictly verboten.
			if _, ok := cache[appName][recName]; !ok && recName != "" {
				cache[appName][recName] = strings.TrimRight(app.Link(api.RelationRecommendations), "/") + "/" + recName
			}
		}

		// If there is no recommendation name, emit all recommendations in
		// descending order. Note that "all" here mean "everything from the
		// index"; as of writing, that is a partial "most recent" list with no
		// query support for full inclusion.
		if recName == "" {
			result := make([]*RecommendationItem, 0, len(cache[appName]))
			for _, u := range cache[appName] {
				rec, err := l.API.GetRecommendation(ctx, u)
				if err != nil {
					var notFoundErr *api.Error
					if errors.As(err, &notFoundErr) && notFoundErr.Type == ErrRecommendationNotFound && ignoreNotFound {
						continue
					}
					return err
				}
				result = append(result, &RecommendationItem{Recommendation: rec})
			}
			sort.Slice(result, func(i, j int) bool { return result[i].LastModified().After(result[j].LastModified()) })
			for _, r := range result {
				if err := f(r); err != nil {
					return err
				}
			}
			return nil
		}

		// Get the recommendation out of the recommendation cache
		if u, ok := cache[appName][recName]; ok {
			rec, err := l.API.GetRecommendation(ctx, u)
			if err != nil {
				return err
			}
			if err := f(&RecommendationItem{Recommendation: rec}); err != nil {
				var notFoundErr *api.Error
				if errors.As(err, &notFoundErr) && notFoundErr.Type == ErrRecommendationNotFound && ignoreNotFound {
					continue
				}
				return err
			}
		} else if !ignoreNotFound {
			return &api.Error{Type: ErrRecommendationNotFound, Message: fmt.Sprintf("recommendation not found: %q", recName)}
		}
	}
	return nil
}

// GetApplicationByNameOrTitle tries to get an application by name and falls back to a
// linear search over all the applications by title.
func (l *Lister) GetApplicationByNameOrTitle(ctx context.Context, name string) (*Application, error) {
	// First try to get the application by name
	app, err := l.API.GetApplicationByName(ctx, ApplicationName(name))
	if err == nil {
		return &app, nil
	}

	// Unless it's an "app not found" error, there is nothing we can do
	var notFoundErr *api.Error
	if !errors.As(err, &notFoundErr) || notFoundErr.Type != ErrApplicationNotFound {
		return nil, err
	}

	// Try to find the application by title
	found := false
	err = l.ForEachApplication(ctx, ApplicationListQuery{}, func(item *ApplicationItem) error {
		if item.Title() == name {
			app = item.Application
			found = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Not found, return the original "app not found" error
	if !found {
		return nil, notFoundErr
	}

	return &app, nil
}

// GetScenarioByNameOrTitle tries to get a scenario by name and falls back to a
// linear search over all the scenarios by title.
func (l *Lister) GetScenarioByNameOrTitle(ctx context.Context, app *Application, name string) (*Scenario, error) {
	var scnByName, scnByTitle *Scenario
	err := l.ForEachScenario(ctx, app, ScenarioListQuery{}, func(scn *ScenarioItem) error {
		// This should be unique
		if scn.Name == name {
			scnByName = &scn.Scenario
		}

		// This might be ambiguous
		if scn.Title() == name {
			scnByTitle = &scn.Scenario
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// If we didn't find anything, report it as an error
	if scnByName == nil && scnByTitle == nil {
		return nil, &api.Error{
			Type:    ErrScenarioNotFound,
			Message: fmt.Sprintf("scenario %q not found", name),
		}
	}

	// Prefer the scenario with the matching name
	if scnByName != nil {
		return scnByName, nil
	}
	return scnByTitle, nil
}

// ForEachCluster iterates over all the clusters.
func (l *Lister) ForEachCluster(ctx context.Context, f func(item *ClusterItem) error) error {
	// Define a helper to iteratively (NOT recursively) visit clusters
	forEach := func(lst ClusterList, err error) (string, error) {
		if err != nil {
			return "", err
		}

		for i := range lst.Items {
			if err := f(&lst.Items[i]); err != nil {
				return "", err
			}
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}

		return lst.Link(api.RelationNext), nil
	}

	// Iterate over all clusters, starting with first page
	u, err := forEach(l.API.ListClusters(ctx))
	for u != "" && err == nil {
		// At the time of writing, the clusters list API did not support paging
		panic("not implemented")
	}
	return err
}

// ForEachNamedCluster iterates over all the named clusters, optionally ignoring those that do not exist.
func (l *Lister) ForEachNamedCluster(ctx context.Context, names []string, ignoreNotFound bool, f func(item *ClusterItem) error) error {
	for _, name := range names {
		c, err := l.API.GetClusterByName(ctx, ClusterName(name))
		if err != nil {
			var notFoundErr *api.Error
			if errors.As(err, &notFoundErr) && notFoundErr.Type == ErrClusterNotFound && ignoreNotFound {
				continue
			}
			return err
		}

		if err := f(&ClusterItem{Cluster: c}); err != nil {
			return err
		}
	}
	return nil
}
