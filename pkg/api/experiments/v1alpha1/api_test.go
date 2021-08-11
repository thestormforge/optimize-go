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

package v1alpha1_test

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thestormforge/optimize-go/pkg/api"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
	"github.com/thestormforge/optimize-go/pkg/api/internal/apitest"
)

var (
	client api.Client
	cases  []apitest.ExperimentTestDefinition
)

func TestMain(m *testing.M) {
	var err error
	path := "testdata"
	flag.Parse()

	// Create a client
	client, err = apitest.NewClient(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Load the test data
	cases, err = apitest.ReadExperimentsTestData(path)
	if err != nil {
		log.Fatal(err)
	}

	// Execute the tests
	os.Exit(m.Run())
}

func TestAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping API test in short mode.")
	}

	expAPI := experiments.NewAPI(client)

	for i := range cases {
		t.Run(string(cases[i].ExperimentName), func(t *testing.T) {
			runTest(t, &cases[i], expAPI)
		})
	}
}

func runTest(t *testing.T, td *apitest.ExperimentTestDefinition, expAPI experiments.API) {
	ctx := context.Background()
	var exp experiments.Experiment

	ok := t.Run("Create Experiment", func(t *testing.T) {
		var err error
		exp, err = expAPI.CreateExperimentByName(ctx, td.ExperimentName, td.Experiment)
		require.NoError(t, err, "failed to create experiment by name")

		// We need the URLs for creating and obtaining trials
		assert.NotEmpty(t, exp.Link(api.RelationTrials), "missing trials link")
		assert.NotEmpty(t, exp.Link(api.RelationNextTrial), "missing next trial link")

		// Since this was a PUT instead of a POST we are expecting a self link instead of a location
		assert.NotEmpty(t, exp.Link(api.RelationSelf), "missing self link")

		// Ensure we have the required application and scenario labels
		assert.NotEmpty(t, exp.Labels["application"], "missing application label")
		assert.NotEmpty(t, exp.Labels["scenario"], "missing scenario label")
	})

	t.Run("Send Baseline", func(t *testing.T) {
		if !ok || td.Baseline == nil {
			t.Skip("skipping baseline.")
		}

		suggestion := experiments.TrialAssignments{
			Labels:      map[string]string{"baseline": "true"},
			Assignments: td.Baseline,
		}

		bl, err := expAPI.CreateTrial(ctx, exp.Link(api.RelationTrials), suggestion)
		require.NoError(t, err, "failed to create baseline trial")

		ta, err := expAPI.NextTrial(ctx, exp.Link(api.RelationNextTrial))
		require.NoError(t, err, "failed to fetch baseline trial assignments")
		require.NotEmpty(t, ta.Location(), "missing baseline location")
		assert.Equal(t, indexAssignments(&bl), indexAssignments(&ta), "first trial is not the baseline")

		err = expAPI.ReportTrial(ctx, ta.Location(), td.TrialResults(&ta))
		require.NoError(t, err, "failed to report baseline trial")
	})

	t.Run("Trial Loop", func(t *testing.T) {
		if !ok || exp.Link(api.RelationNextTrial) == "" {
			t.Skip("skipping trial loop.")
		}

		for {
			ta, err := expAPI.NextTrial(ctx, exp.Link(api.RelationNextTrial))
			var aerr *api.Error
			if errors.As(err, &aerr) && aerr.Type == experiments.ErrExperimentStopped {
				break
			}
			require.NoError(t, err, "failed to fetch trial assignments")
			assert.NotEmpty(t, ta.Location(), "missing location")

			err = expAPI.ReportTrial(ctx, ta.Location(), td.TrialResults(&ta))
			require.NoError(t, err, "failed to report trial")
		}
	})

	t.Run("Delete Experiment", func(t *testing.T) {
		if exp.Link(api.RelationSelf) == "" {
			t.Skip("skipping delete experiment.")
		}

		err := expAPI.DeleteExperiment(ctx, exp.Link(api.RelationSelf))
		require.NoError(t, err, "failed to delete experiment")
	})
}

func indexAssignments(ta *experiments.TrialAssignments) map[string]api.NumberOrString {
	result := make(map[string]api.NumberOrString, len(ta.Assignments))
	for _, a := range ta.Assignments {
		result[a.ParameterName] = a.Value
	}
	return result
}
