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
	"encoding/json"

	"github.com/thestormforge/optimize-go/pkg/api"
)

type Scenario struct {
	api.Metadata  `json:"-"`
	Name          ScenarioName  `json:"name,omitempty"`
	DisplayName   string        `json:"title,omitempty"`
	Configuration []interface{} `json:"configuration,omitempty"`
	Objective     []interface{} `json:"objective,omitempty"`
	Clusters      []string      `json:"clusters,omitempty"`

	StormForgePerformance interface{} `json:"stormforgePerf,omitempty"`
	Locust                interface{} `json:"locust,omitempty"`
	Custom                interface{} `json:"custom,omitempty"`
}

// NOTE: Use `DisplayName` as the field since `Title()` is a function on the embedded `Metadata`
var _ = Scenario{}.Title()

type ScenarioListQuery struct{ api.IndexQuery }

type ScenarioItem struct {
	Scenario
}

func (l *ScenarioItem) UnmarshalJSON(b []byte) error {
	type t ScenarioItem
	return api.UnmarshalJSON(b, (*t)(l))
}

type ScenarioList struct {
	// The scenario list metadata.
	api.Metadata `json:"-"`
	// The total number of items in the collection.
	TotalCount int `json:"totalCount,omitempty"`
	// The list of scenarios.
	Scenarios []ScenarioItem `json:"scenarios,omitempty"`
}

type TemplateParameterBounds struct {
	// The minimum value for a numeric parameter.
	Min json.Number `json:"min,omitempty"`
	// The maximum value for a numeric parameter.
	Max json.Number `json:"max,omitempty"`
}

type TemplateParameter struct {
	// The name of the parameter.
	Name string `json:"name"`
	// The type of the parameter.
	Type string `json:"type"`
	// The optional baseline value of the parameter, either numeric or categorical.
	Baseline *api.NumberOrString `json:"baseline,omitempty"`
	// The domain of the parameter.
	Bounds *TemplateParameterBounds `json:"bounds,omitempty"`
	// The list of allowed categorical values for the parameter.
	Values []string `json:"values,omitempty"`
}

type TemplateMetricBounds struct {
	// The minimum value for a metric.
	Min float64 `json:"min,omitempty"`
	// The maximum value for a metric.
	Max float64 `json:"max,omitempty"`
}

type TemplateMetric struct {
	// The name of the metric.
	Name string `json:"name"`
	// The flag indicating this metric should be minimized.
	Minimize bool `json:"minimize,omitempty"`
	// The flag indicating this metric is optimized (nil defaults to true).
	Optimize *bool `json:"optimize,omitempty"`
	// The domain of the metric
	Bounds *TemplateMetricBounds `json:"bounds,omitempty"`
}

type Template struct {
	// The list of parameters for this template.
	Parameters []TemplateParameter `json:"parameters,omitempty"`
	// The list of metrics for this template.
	Metrics []TemplateMetric `json:"metrics,omitempty"`
}
