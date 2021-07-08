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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thestormforge/optimize-go/pkg/api"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
	"github.com/thestormforge/optimize-go/pkg/api/internal/apitest"
)

var (
	client api.Client
	cases  []TestDefinition
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
	cases, err = ReadTestData(path)
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
		t.Run(cases[i].ExperimentName.Name(), func(t *testing.T) {
			runTest(t, &cases[i], expAPI)
		})
	}
}

func runTest(t *testing.T, td *TestDefinition, expAPI experiments.API) {
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

// TestDefinition is used to define tests to run against an experiments API implementation.
type TestDefinition struct {
	// The name of the experiment to create during the test.
	ExperimentName experiments.ExperimentName
	// The experiment definition for testing.
	Experiment experiments.Experiment
	// A list of assignments for the baseline trial. May be empty to skip baseline trial tests.
	Baseline []experiments.Assignment
	// A matrix of weights used to map a vector of parameter assignments to a vector of metric values.
	Values [][]float64
	// The list of conditions to report as a failure.
	Failures []struct {
		// The parameter or metric name (assumes they never conflict).
		Name string
		// The optional minimum value to trigger the failure.
		Min *float64
		// The optional maximum value to trigger the failure.
		Max *float64
		// The failure reason to report when the condition is triggered.
		Reason string
		// The failure message to report when the condition is triggered.
		Message string
	}
}

// TrialResults computes dummy trial results for the supplied assignments.
func (d *TestDefinition) TrialResults(ta *experiments.TrialAssignments) experiments.TrialValues {
	// Sanity check the dimensions of the values matrix
	if len(d.Experiment.Parameters) != len(d.Values) || len(d.Experiment.Metrics) != len(d.Values[0]) {
		log.Panicf("values should be %dx%d", len(d.Experiment.Parameters), len(d.Experiment.Metrics))
	}

	tv := experiments.TrialValues{
		StartTime:      new(time.Time),
		CompletionTime: new(time.Time),
	}

	// Set some dummy times for the trial
	*tv.StartTime = time.Now()
	*tv.CompletionTime = tv.StartTime.Add(1 * time.Second)

	// Compute the values
	v := make([]experiments.Value, len(d.Experiment.Metrics))
	for c := range ta.Assignments {
		for r := range d.Experiment.Metrics {
			v[r].MetricName = d.Experiment.Metrics[r].Name
			v[r].Value += ta.Assignments[c].Value.Float64Value() * d.Values[c][r]
		}
	}

	// Index parameter and metric values together for error checking (assume conflict)
	nv := make(map[string]float64, len(d.Experiment.Parameters)+len(d.Experiment.Metrics))
	for _, n := range ta.Assignments {
		nv[n.ParameterName] = n.Value.Float64Value()
	}
	for _, n := range v {
		nv[n.MetricName] = n.Value
	}
	isFailure := func(min, max *float64, v float64) bool {
		return (min != nil && v > *min) || (max != nil && v < *max)
	}

	// Check for failures
	for _, f := range d.Failures {
		if isFailure(f.Min, f.Max, nv[f.Name]) {
			tv.Failed = true
			tv.FailureReason = f.Reason
			tv.FailureMessage = f.Message
			return tv
		}
	}

	// Use the values if there was no failure
	tv.Values = v
	return tv
}

// ReadTestData reads all of the JSON files in the supplied test data directory.
func ReadTestData(path string) ([]TestDefinition, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data directory %q: %w", path, err)
	}

	var result []TestDefinition
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read test data %q: %w", entry.Name(), err)
		}

		td := TestDefinition{}
		if err := json.Unmarshal(data, &td); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test definition: %w", err)
		}
		if td.ExperimentName == nil || td.ExperimentName.Name() == "" {
			td.ExperimentName = experiments.NewExperimentName(strings.TrimSuffix(filepath.Base(entry.Name()), ".json"))
		}

		result = append(result, td)
	}
	return result, nil
}
