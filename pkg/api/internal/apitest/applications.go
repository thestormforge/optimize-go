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
	"os"
	"path/filepath"

	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

// ApplicationTestDefinition is used to define tests to run against an application API implementation.
type ApplicationTestDefinition struct {
	Application applications.Application
	Scenario    applications.Scenario

	// In addition to the application and scenario, we need an experiment.
	ExperimentTestDefinition
}

// GenerateTemplate returns details of experiment for this test definition.
func (td *ApplicationTestDefinition) GenerateTemplate() applications.Template {
	result := applications.Template{
		Parameters: make([]applications.TemplateParameter, 0, len(td.Experiment.Parameters)),
		Metrics:    make([]applications.TemplateMetric, 0, len(td.Experiment.Metrics)),
	}

	for _, p := range td.Experiment.Parameters {
		sp := applications.TemplateParameter{
			Name:   p.Name,
			Type:   string(p.Type),
			Values: p.Values,
		}
		if p.Bounds != nil {
			sp.Bounds = &applications.TemplateParameterBounds{
				Min: p.Bounds.Min,
				Max: p.Bounds.Max,
			}
		}
		for _, b := range td.Baseline {
			if b.ParameterName == p.Name {
				sp.Baseline = &b.Value
			}
		}
		result.Parameters = append(result.Parameters, sp)
	}

	for _, m := range td.Experiment.Metrics {
		sm := applications.TemplateMetric{
			Name:     m.Name,
			Minimize: m.Minimize,
			Optimize: m.Optimize,
		}
		// TODO Where do we get the metric bounds from?
		result.Metrics = append(result.Metrics, sm)
	}

	return result
}

// ReadApplicationTestData reads all of the JSON files in the supplied test data directory.
func ReadApplicationTestData(path string) ([]ApplicationTestDefinition, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data directory %q: %w", path, err)
	}

	var result []ApplicationTestDefinition
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read test data %q: %w", entry.Name(), err)
		}

		td := ApplicationTestDefinition{}
		if err := json.Unmarshal(data, &td); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test definition: %w", err)
		}

		result = append(result, td)
	}
	return result, nil
}
