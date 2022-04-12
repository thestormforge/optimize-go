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

package command

import (
	"io"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Printer is the interface required to render results.
type Printer interface {
	// Fprint renders an object (which may represent a list) to the supplied writer.
	Fprint(out io.Writer, obj interface{}) error
}

// formatTime is a helper that returns empty strings for zero times and adds
// support for a humanized format (if the layout is empty).
func formatTime(t time.Time, layout string) string {
	switch {
	case t.IsZero():
		return ""
	case layout == "":
		return humanize.Time(t)
	default:
		return t.Format(layout)
	}
}

// NOTE: All the "*Row" structs have `json:"-"` for everything EXCEPT their
// inline "*Item" field so when the row is marshalled as JSON it appears the
// same as what the item would have been.

// ApplicationRow is a table row representation of an application.
type ApplicationRow struct {
	Name                string `table:"name" csv:"name" json:"-"`
	DisplayName         string `table:"Name,custom" json:"-"`
	ScenarioCount       int    `table:"scenarios,wide" csv:"scenario_count" json:"-"`
	RecommendationMode  string `table:"recommendations" csv:"recommendations" json:"-"`
	DeployInterval      string `table:"deploy_interval" csv:"deploy_interval" json:"-"`
	LastDeployedMachine string `table:"-" csv:"last_deployed" json:"-"`
	LastDeployedHuman   string `table:"last_deployed" csv:"-" json:"-"`
	Age                 string `table:"age,wide" csv:"-" json:"-"`

	applications.ApplicationItem `table:"-" csv:"-"`
}

// ApplicationOutput wraps an application list for output.
type ApplicationOutput struct {
	Items []ApplicationRow `json:"items"`
}

// Add an application item to the output.
func (o *ApplicationOutput) Add(item *applications.ApplicationItem) error {
	o.Items = append(o.Items, ApplicationRow{
		Name:                item.Name.String(),
		DisplayName:         item.Title(),
		ScenarioCount:       item.ScenarioCount,
		RecommendationMode:  cases.Title(language.AmericanEnglish).String(string(item.Recommendations)),
		LastDeployedMachine: formatTime(item.LastDeployedAt, time.RFC3339),
		LastDeployedHuman:   formatTime(item.LastDeployedAt, ""),
		Age:                 formatTime(item.CreatedAt, ""),

		ApplicationItem: *item,
	})
	return nil
}

// RecommendationRow is a table row representation of a recommendation.
type RecommendationRow struct {
	applications.RecommendationItem `table:"-" csv:"-"`
}

// RecommendationOutput wraps a recommendation list for output.
type RecommendationOutput struct {
	Items []RecommendationRow `json:"items"`
}

// Add a recommendation item to the output.
func (o *RecommendationOutput) Add(item *applications.RecommendationItem) error {
	o.Items = append(o.Items, RecommendationRow{
		RecommendationItem: *item,
	})
	return nil
}

// ExperimentRow is a table row representation of an experiment.
type ExperimentRow struct {
	Name         string            `table:"name" csv:"name" json:"-"`
	DisplayName  string            `table:"Name,custom" json:"-"`
	Observations int64             `table:"observations,wide" csv:"observations" json:"-"`
	Labels       map[string]string `table:"labels,labels" csv:"label_,labels,flatten" json:"-"`

	experiments.ExperimentItem `table:"-" csv:"-"`
}

// ExperimentOutput wraps an experiment list for output.
type ExperimentOutput struct {
	Items []ExperimentRow `json:"items"`
}

// Add an experiment item to the output.
func (o *ExperimentOutput) Add(item *experiments.ExperimentItem) error {
	o.Items = append(o.Items, ExperimentRow{
		Name:         item.Name.String(),
		DisplayName:  item.DisplayName,
		Observations: item.Observations,
		Labels:       item.Labels,

		ExperimentItem: *item,
	})
	return nil
}

// TrialRow is a table row representation of a trial.
type TrialRow struct {
	Experiment     string            `table:"experiment,custom" csv:"experiment" json:"-"`
	Name           string            `table:"name" json:"-"`
	Number         int64             `table:"number,custom" csv:"number" json:"-"`
	Status         string            `table:"status" csv:"status" json:"-"`
	Assignments    map[string]string `csv:"parameter_,flatten" json:"-"`
	Values         map[string]string `csv:"metric_,flatten" json:"-"`
	FailureReason  string            `table:"failure_reason,wide" csv:"failure_reason" json:"-"`
	FailureMessage string            `table:"failure_message,wide" csv:"failure_message" json:"-"`
	Labels         map[string]string `table:"labels,labels" csv:"label_,labels,flatten" json:"-"`

	experiments.TrialItem `table:"-" csv:"-"`
}

// TrialOutput wraps a trial list for output.
type TrialOutput struct {
	Items []TrialRow `json:"items"`
}

// Add a trial item to the output.
func (o *TrialOutput) Add(item *experiments.TrialItem) error {
	var experiment string
	if item.Experiment != nil {
		experiment = item.Experiment.DisplayName
	}

	assignments := make(map[string]string, len(item.Assignments))
	for i := range item.Assignments {
		assignments[item.Assignments[i].ParameterName] = item.Assignments[i].Value.String()
	}

	values := make(map[string]string, len(item.Values))
	for i := range item.Values {
		values[item.Values[i].MetricName] = strconv.FormatFloat(item.Values[i].Value, 'f', -1, 64)
	}

	o.Items = append(o.Items, TrialRow{
		Experiment:     experiment,
		Name:           experiments.JoinTrialName(item.Experiment, item.Number),
		Number:         item.Number,
		Status:         cases.Title(language.English).String(string(item.Status)),
		FailureReason:  item.FailureReason,
		FailureMessage: item.FailureMessage,
		Assignments:    assignments,
		Values:         values,
		Labels:         item.Labels,

		TrialItem: *item,
	})

	return nil
}

// ClusterRow is a table row representation of a cluster.
type ClusterRow struct {
	Name                   string `table:"name" csv:"name" json:"-"`
	DisplayName            string `table:"Name,custom" json:"-"`
	OptimizeProVersion     string `table:"optimize_pro" csv:"optimize_pro_version" json:"-"`
	OptimizeLiveVersion    string `table:"optimize_live" csv:"optimize_live_version" json:"-"`
	PerformanceTestVersion string `table:"performance_test,wide" csv:"performance_test_version" json:"-"`
	KubernetesVersion      string `table:"kubernetes,wide" csv:"kubernetes_version" json:"-"`
	LastSeenMachine        string `table:"-" csv:"last_seen" json:"-"`
	LastSeenHuman          string `table:"last_seen" csv:"-" json:"-"`
	Age                    string `table:"age,wide" csv:"-" json:"-"`

	applications.ClusterItem `table:"-" csv:"-"`
}

// ClusterOutput wraps a cluster list for output.
type ClusterOutput struct {
	Items []ClusterRow `json:"items"`
}

// Add a cluster item to the output.
func (o *ClusterOutput) Add(item *applications.ClusterItem) error {
	o.Items = append(o.Items, ClusterRow{
		Name:                   item.Name.String(),
		DisplayName:            item.Title(),
		OptimizeProVersion:     item.OptimizeProVersion,
		OptimizeLiveVersion:    item.OptimizeLiveVersion,
		PerformanceTestVersion: item.PerformanceTestVersion,
		KubernetesVersion:      item.KubernetesVersion,
		LastSeenMachine:        formatTime(item.LastSeen, time.RFC3339),
		LastSeenHuman:          formatTime(item.LastSeen, ""),
		Age:                    formatTime(item.CreatedAt, ""),

		ClusterItem: *item,
	})
	return nil
}
