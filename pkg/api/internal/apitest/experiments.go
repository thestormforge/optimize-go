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

package apitest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
)

// ExperimentTestDefinition is used to define tests to run against an experiments API implementation.
type ExperimentTestDefinition struct {
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
func (d *ExperimentTestDefinition) TrialResults(ta *experiments.TrialAssignments) experiments.TrialValues {
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

// ReadExperimentsTestData reads all of the JSON files in the supplied test data directory.
func ReadExperimentsTestData(path string) ([]ExperimentTestDefinition, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data directory %q: %w", path, err)
	}

	var result []ExperimentTestDefinition
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read test data %q: %w", entry.Name(), err)
		}

		td := ExperimentTestDefinition{}
		if err := json.Unmarshal(data, &td); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test definition: %w", err)
		}
		if td.ExperimentName == "" {
			td.ExperimentName = experiments.ExperimentName(strings.TrimSuffix(filepath.Base(entry.Name()), ".json"))
		}

		result = append(result, td)
	}
	return result, nil
}
