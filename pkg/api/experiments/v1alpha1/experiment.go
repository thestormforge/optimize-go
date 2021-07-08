/*
Copyright 2020 GramLabs, Inc.

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
	"encoding/json"
	"net/url"
	"strings"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// ExperimentName exists to clearly separate cases where an actual name can be used
type ExperimentName interface {
	Name() string
}

// NewExperimentName returns an experiment name for a given string
func NewExperimentName(n string) ExperimentName {
	return experimentName(n)
}

type experimentName string

func (n experimentName) Name() string   { return string(n) }
func (n experimentName) String() string { return string(n) }

type Optimization struct {
	// The name of the optimization parameter.
	Name string `json:"name"`
	// The value of the optimization parameter.
	Value string `json:"value"`
}

type Metric struct {
	// The name of the metric.
	Name string `json:"name"`
	// The flag indicating this metric should be minimized.
	Minimize bool `json:"minimize,omitempty"`
	// The flag indicating this metric is optimized (nil defaults to true).
	Optimize *bool `json:"optimize,omitempty"`
}

type ConstraintType string

const (
	ConstraintSum   ConstraintType = "sum"
	ConstraintOrder ConstraintType = "order"
)

type SumConstraintParameter struct {
	// Name of parameter to be used in constraint.
	ParameterName string `json:"parameterName"`
	// Weight for parameter in constraint.
	Weight float64 `json:"weight"`
}

type SumConstraint struct {
	// Flag indicating if bound is upper or lower bound.
	IsUpperBound bool `json:"isUpperBound,omitempty"`
	// Bound for inequality constraint.
	Bound float64 `json:"bound"`
	// Parameters and weights for constraint.
	Parameters []SumConstraintParameter `json:"parameters"`
}

type OrderConstraint struct {
	// Name of lower parameter.
	LowerParameter string `json:"lowerParameter"`
	// Name of upper parameter.
	UpperParameter string `json:"upperParameter"`
}

type Constraint struct {
	// Optional name for constraint.
	Name string `json:"name,omitempty"`

	ConstraintType   ConstraintType `json:"constraintType"`
	*SumConstraint   `json:",omitempty"`
	*OrderConstraint `json:",omitempty"`
}

type ParameterType string

const (
	ParameterTypeInteger     ParameterType = "int"
	ParameterTypeDouble      ParameterType = "double"
	ParameterTypeCategorical ParameterType = "categorical"
)

type Bounds struct {
	// The minimum value for a numeric parameter.
	Min json.Number `json:"min"`
	// The maximum value for a numeric parameter.
	Max json.Number `json:"max"`
}

// Parameter is a variable that is going to be tuned in an experiment
type Parameter struct {
	// The name of the parameter.
	Name string `json:"name"`
	// The type of the parameter.
	Type ParameterType `json:"type"`
	// The domain of the parameter.
	Bounds *Bounds `json:"bounds,omitempty"`
	// The discrete values for a categorical parameter.
	Values []string `json:"values,omitempty"`
}

// Experiment combines the search space, outcomes and optimization configuration
type Experiment struct {
	// The experiment metadata.
	api.Metadata `json:"-"`
	// The display name of the experiment. Do not use for generating URLs!
	DisplayName string `json:"displayName,omitempty"`
	// The number of observations made for this experiment.
	Observations int64 `json:"observations,omitempty"`
	// The target number of observations for this experiment.
	Budget int64 `json:"budget,omitempty"`
	// Controls how the optimizer will generate trials.
	Optimization []Optimization `json:"optimization,omitempty"`
	// The metrics been optimized in the experiment.
	Metrics []Metric `json:"metrics"`
	// Constraints for the experiment.
	Constraints []Constraint `json:"constraints,omitempty"`
	// The search space of the experiment.
	Parameters []Parameter `json:"parameters"`
	// Labels for this experiment.
	Labels map[string]string `json:"labels,omitempty"`
}

// Name allows an experiment to be used as an ExperimentName
func (e *Experiment) Name() string {
	u, err := url.Parse(e.Link(api.RelationSelf))
	if err != nil {
		return ""
	}
	i := strings.Index(u.Path, endpointExperiments)
	if i < 0 {
		return ""
	}
	return u.Path[len(endpointExperiments)+i:]
}

type ExperimentListQuery struct{ api.IndexQuery }

type ExperimentItem struct{ Experiment }

func (l *ExperimentItem) UnmarshalJSON(b []byte) error { return api.UnmarshalJSON(b, l) }

type ExperimentList struct {
	// The experiment list metadata.
	api.Metadata `json:"-"`
	// The list of experiments.
	Experiments []ExperimentItem `json:"experiments,omitempty"`
}

type ExperimentLabels struct {
	// New labels for this experiment.
	Labels map[string]string `json:"labels"`
}
