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
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thestormforge/optimize-go/pkg/api"
	"github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1/numstr"
)

type Assignment struct {
	// The name of the parameter in the experiment the assignment corresponds to.
	ParameterName string `json:"parameterName"`
	// The assigned value of the parameter.
	Value numstr.NumberOrString `json:"value"`
}

type TrialAssignments struct {
	// The trial metadata.
	api.Metadata `json:"-"`
	// The list of parameter names and their assigned values.
	Assignments []Assignment `json:"assignments"`
	// Labels for this trial.
	Labels map[string]string `json:"labels,omitempty"`
}

type Value struct {
	// The name of the metric in the experiment the value corresponds to.
	MetricName string `json:"metricName"`
	// The observed value of the metric.
	Value float64 `json:"value"`
	// The observed error of the metric.
	Error float64 `json:"error,omitempty"`
}

type TrialValues struct {
	// The observed values.
	Values []Value `json:"values,omitempty"`
	// Indicator that the trial failed, Values is ignored when true.
	Failed bool `json:"failed,omitempty"`
	// FailureReason is a the machine-readable reason code for the failure, if Failed is true.
	FailureReason string `json:"failureReason,omitempty"`
	// FailureMessage is a human-readable explanation of the failure, if Failed is true.
	FailureMessage string `json:"failureMessage,omitempty"`
	// StartTime is the time at which the trial was started.
	StartTime *time.Time `json:"startTime,omitempty"`
	// CompletionTime is the time at which the trial was completed.
	CompletionTime *time.Time `json:"completionTime,omitempty"`
}

type TrialStatus string

const (
	TrialStaged    TrialStatus = "staged"
	TrialActive    TrialStatus = "active"
	TrialCompleted TrialStatus = "completed"
	TrialFailed    TrialStatus = "failed"
	TrialAbandoned TrialStatus = "abandoned"
)

type TrialListQuery struct{ api.IndexQuery }

func (q *TrialListQuery) SetStatus(status ...TrialStatus) {
	str := make([]string, len(status))
	for _, s := range status {
		str = append(str, string(s))
	}
	if len(str) > 0 {
		if q.IndexQuery == nil {
			q.IndexQuery = api.IndexQuery{}
		}
		url.Values(q.IndexQuery).Set("status", strings.Join(str, ","))
	}
}

func (q *TrialListQuery) AddStatus(status TrialStatus) {
	value := string(status)
	if v := url.Values(q.IndexQuery).Get("status"); v != "" {
		value = v + "," + value
	}
	if q.IndexQuery == nil {
		q.IndexQuery = api.IndexQuery{}
	}
	url.Values(q.IndexQuery).Set("status", value)
}

type TrialItem struct {
	TrialAssignments
	TrialValues

	// The current trial status.
	Status TrialStatus `json:"status"`
	// Ordinal number indicating when during an experiment the trail was generated.
	Number int64 `json:"number"`

	// Experiment is a reference back to the experiment this trial item is associated with. This field is never
	// populated by the API, but may be useful for consumers to maintain a connection between resources.
	Experiment *Experiment `json:"-"`
}

func (t *TrialItem) UnmarshalJSON(b []byte) error { return api.UnmarshalJSON(b, t) }

// Name returns an effective name for uniquely identifying the trial.
func (t *TrialItem) Name() string {
	if t.Experiment != nil {
		return fmt.Sprintf("%s-%03d", t.Experiment.Name(), t.Number)
	}

	return strconv.FormatInt(t.Number, 10)
}

type TrialList struct {
	// The trial list metadata.
	api.Metadata `json:"-"`
	// The list of trials.
	Trials []TrialItem `json:"trials"`

	// Experiment is a reference back to the experiment this trial item is associated with. This field is never
	// populated by the API, but may be useful for consumers to maintain a connection between resources.
	Experiment *Experiment `json:"-"`
}

type TrialLabels struct {
	// New labels for this trial.
	Labels map[string]string `json:"labels"`
}

// SplitTrialName provides a consistent experience when trying to split a "trial name" into an experiment
// name and a trial number. When the provided name does not contain a number, the resulting number will
// be less then zero.
func SplitTrialName(name string) (ExperimentName, int64) {
	// Names with slashes are always split (since the slash can't be in the name)
	p := strings.LastIndex(name, "/")
	if p >= 0 {
		if num, err := strconv.ParseInt(name[p+1:], 10, 64); err == nil {
			return experimentName(name[0:p]), num
		}
		return experimentName(name[0:p]), -1
	}

	// The only other allowable separator is the hyphen
	p = strings.LastIndex(name, "-")
	if p >= 0 {
		// Strip off a valid number after the "-". If your experiment name has a "-<NUM>" suffix, use a slash
		if num, err := strconv.ParseInt(name[p+1:], 10, 64); err == nil {
			return experimentName(name[0:p]), num
		}
	}

	return experimentName(name), -1
}
