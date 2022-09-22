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
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// NewTrialAssignments constructs a trial assignments instance using the supplied string values.
// The default behavior can be "none", "baseline", "minimum", "maximum", or "random".
func NewTrialAssignments(e *Experiment, assignments map[string]string, baselines map[string]*api.NumberOrString, defaultBehavior string) (*TrialAssignments, error) {
	ta := &TrialAssignments{}
	for _, p := range e.Parameters {
		v, err := parameterValue(&p, assignments, baselines, defaultBehavior)
		if err != nil {
			return nil, err
		}
		if err := CheckParameterValue(&p, v); err != nil {
			return nil, err
		}
		ta.Assignments = append(ta.Assignments, Assignment{ParameterName: p.Name, Value: *v})
	}
	if err := CheckParameterConstraints(ta.Assignments, e.Constraints); err != nil {
		return nil, err
	}
	return ta, nil
}

func parameterValue(p *Parameter, assignments map[string]string, baselines map[string]*api.NumberOrString, defaultBehavior string) (*api.NumberOrString, error) {
	if a, ok := assignments[p.Name]; ok {
		return p.ParseValue(a)
	}

	switch defaultBehavior {
	case "none", "":
		return nil, nil
	case "base", "baseline":
		return baselines[p.Name], nil
	case "min", "minimum":
		return p.LowerBound()
	case "max", "maximum":
		return p.UpperBound()
	case "rand", "random":
		return p.RandomValue()
	default:
		return nil, fmt.Errorf("unknown default behavior %q", defaultBehavior)
	}
}

// LowerBound attempts to return the lower bound for this parameter.
func (p *Parameter) LowerBound() (*api.NumberOrString, error) {
	if p.Type == ParameterTypeCategorical {
		if len(p.Values) == 0 {
			return nil, fmt.Errorf("unable to determine categorical minimum bound")
		}
		return &api.NumberOrString{StrVal: p.Values[0], IsString: true}, nil
	}

	if p.Bounds == nil {
		return nil, fmt.Errorf("unable to determine numeric minimum bound")
	}

	return &api.NumberOrString{NumVal: p.Bounds.Min}, nil
}

// UpperBound attempts to return the upper bound for this parameter.
func (p *Parameter) UpperBound() (*api.NumberOrString, error) {
	if p.Type == ParameterTypeCategorical {
		if len(p.Values) == 0 {
			return nil, fmt.Errorf("unable to determine categorical maximum bound")
		}
		return &api.NumberOrString{StrVal: p.Values[len(p.Values)-1], IsString: true}, nil
	}

	if p.Bounds == nil {
		return nil, fmt.Errorf("unable to determine numeric maximum bound")
	}

	return &api.NumberOrString{NumVal: p.Bounds.Max}, nil
}

// ParseValue attempts to parse the supplied value into a NumberOrString based on the type of this parameter.
func (p *Parameter) ParseValue(s string) (*api.NumberOrString, error) {
	var v api.NumberOrString
	switch p.Type {
	case ParameterTypeInteger:
		if _, err := strconv.ParseInt(s, 10, 64); err != nil {
			return nil, err
		}
		v = api.FromNumber(json.Number(s))
	case ParameterTypeDouble:
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return nil, err
		}
		v = api.FromNumber(json.Number(s))
	case ParameterTypeCategorical:
		v = api.FromString(s)
	}
	return &v, nil
}

// RandomValue returns a random value for a parameter.
func (p *Parameter) RandomValue() (*api.NumberOrString, error) {
	var v api.NumberOrString
	switch p.Type {
	case ParameterTypeInteger:
		min, err := p.Bounds.Min.Int64()
		if err != nil {
			return nil, fmt.Errorf("unable to determine minimum integer bound: %w", err)
		}
		max, err := p.Bounds.Max.Int64()
		if err != nil {
			return nil, fmt.Errorf("unable to determine maximum integer bound: %w", err)
		}
		v = api.FromInt64(rand.Int63n(max-min) + min)
	case ParameterTypeDouble:
		min, err := p.Bounds.Min.Float64()
		if err != nil {
			return nil, fmt.Errorf("unable to determine minimum double bound: %w", err)
		}
		max, err := p.Bounds.Max.Float64()
		if err != nil {
			return nil, fmt.Errorf("unable to determine maximum double bound: %w", err)
		}
		v = api.FromFloat64(rand.Float64()*max + min)
	case ParameterTypeCategorical:
		v = api.FromString(p.Values[rand.Intn(len(p.Values))])
	}
	return &v, nil
}

// CheckParameterValue validates that the supplied value can be used for a parameter.
func CheckParameterValue(p *Parameter, v *api.NumberOrString) error {
	if v == nil {
		return fmt.Errorf("no assignment for parameter %q", p.Name)
	}

	if p.Type == ParameterTypeCategorical {
		if !v.IsString {
			return fmt.Errorf("categorical value must be a string: %s", v.String())
		}
		for _, allowed := range p.Values {
			if v.StrVal == allowed {
				return nil
			}
		}
		return fmt.Errorf("categorical value is out of range: %s [%s]", v.String(), strings.Join(p.Values, ", "))
	}

	if v.IsString {
		return fmt.Errorf("numeric value must not be a string: %s", v.String())
	}

	lower, err := p.LowerBound()
	if err != nil {
		return err
	}
	upper, err := p.UpperBound()
	if err != nil {
		return err
	}

	switch p.Type {
	case ParameterTypeInteger:
		val := v.Int64Value()
		min, max := lower.Int64Value(), upper.Int64Value()
		if val < min || val > max {
			return fmt.Errorf("integer value is out of range [%d-%d]: %d", min, max, val)
		}
	case ParameterTypeDouble:
		val := v.Float64Value()
		min, max := lower.Float64Value(), upper.Float64Value()
		if val < min || val > max {
			return fmt.Errorf("double value is out of range [%f-%f]: %f", min, max, val)
		}
	default:
		return fmt.Errorf("unknown parameter type: %s", p.Type)
	}
	return nil
}

// CheckParameterConstraints validates that the supplied assignments do not validate the constraints.
func CheckParameterConstraints(assignments []Assignment, constraints []Constraint) error {
	if len(constraints) == 0 || len(assignments) == 0 {
		return nil
	}

	// Index numeric assignments and expose a helper for validating them
	values := make(map[string]float64, len(assignments))
	for _, a := range assignments {
		if a.Value.IsString {
			values[a.ParameterName] = math.NaN()
		} else {
			values[a.ParameterName] = a.Value.Float64Value()
		}
	}
	getValue := func(constraintName, parameterName string) (float64, error) {
		value, ok := values[parameterName]
		switch {
		case !ok:
			return 0, fmt.Errorf("constraint %q references missing parameter %q", constraintName, parameterName)
		case math.IsNaN(value):
			return 0, fmt.Errorf("non-numeric assignment for parameter %q cannot be used to satisfy constraint %q", parameterName, constraintName)
		default:
			return value, nil
		}
	}

	for _, c := range constraints {
		switch c.ConstraintType {
		case ConstraintOrder:
			lower, err := getValue(c.Name, c.OrderConstraint.LowerParameter)
			if err != nil {
				return err
			}

			upper, err := getValue(c.Name, c.OrderConstraint.UpperParameter)
			if err != nil {
				return err
			}

			if lower > upper {
				return fmt.Errorf("assignment does not satisfy constraint %q", c.Name)
			}

		case ConstraintSum:
			var sum float64
			for _, p := range c.SumConstraint.Parameters {
				value, err := getValue(c.Name, p.ParameterName)
				if err != nil {
					return err
				}

				sum += value * p.Weight
			}

			if (c.IsUpperBound && sum > c.Bound) || (!c.IsUpperBound && sum < c.Bound) {
				return fmt.Errorf("assignment does not satisfy constraint %q", c.Name)
			}
		}
	}

	return nil
}
