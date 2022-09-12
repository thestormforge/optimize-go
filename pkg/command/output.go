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
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
	"golang.org/x/text/cases"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Printer is the interface required to render results.
type Printer interface {
	// Fprint renders an object (which may represent a list) to the supplied writer.
	Fprint(out io.Writer, obj interface{}) error
}

// formatTime is a helper that returns empty strings for zero times and adds
// support for a humanized format (if the layout is empty).
func formatTime(t *time.Time, layout string) string {
	switch {
	case t == nil || t.IsZero():
		return ""
	case layout == "ago":
		return humanize.Time(*t)
	case layout == "":
		return strings.TrimSpace(humanize.RelTime(*t, time.Now(), "", ""))
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
	Title               string `table:"title" csv:"title" json:"title,omitempty"`
	ScenarioCount       int    `table:"scenarios" csv:"scenario_count" json:"-"`
	RecommendationMode  string `table:"recommendations" csv:"recommendations" json:"-"`
	DeployInterval      string `table:"deploy_interval,wide" csv:"deploy_interval" json:"-"`
	LastDeployedMachine string `table:"-" csv:"last_deployed" json:"-"`
	LastDeployedHuman   string `table:"last_deployed,wide" csv:"-" json:"-"`
	Age                 string `table:"age,wide" csv:"-" json:"-"`

	applications.ApplicationItem `table:"-" csv:"-"`

	// Special case: the recommendation configuration fields are sub-resources of the actual application

	RecommendationsDeployConfig  *applications.DeployConfiguration `table:"-" csv:"-" json:"recommendationsDeployConfig,omitempty"`
	RecommendationsConfiguration []applications.Configuration      `table:"-" csv:"-" json:"recommendationsConfiguration,omitempty"`
}

func NewApplicationRow(item *applications.ApplicationItem) *ApplicationRow {
	return &ApplicationRow{
		Name:                item.Name.String(),
		Title:               item.Title(),
		ScenarioCount:       item.ScenarioCount,
		RecommendationMode:  "Disabled",
		LastDeployedMachine: formatTime(item.LastDeployedAt, time.RFC3339),
		LastDeployedHuman:   formatTime(item.LastDeployedAt, "ago"),
		Age:                 formatTime(item.CreatedAt, ""),

		ApplicationItem: *item,
	}
}

func (r *ApplicationRow) Lookup(key string) (interface{}, bool) {
	switch SortByKey(key) {
	case "name":
		return r.Name, true
	case "title":
		return r.Title, true
	case "scenarios":
		return r.ScenarioCount, true
	case "recommendations":
		return r.RecommendationMode, true
	case "deploy_interval":
		return r.DeployInterval, true
	case "last_deployed":
		return r.ApplicationItem.LastDeployedAt, true
	case "age":
		return r.ApplicationItem.CreatedAt, true
	default:
		return nil, false
	}
}

func (r *ApplicationRow) SetRecommendationsDeployConfig(deploy *applications.DeployConfiguration) {
	if deploy == nil {
		return
	}

	// Hack to account for missing mode information
	if deploy.Mode == "" {
		deploy.Mode = r.ApplicationItem.Recommendations
	}
	if deploy.Mode != "" {
		r.RecommendationMode = cases.Title(language.AmericanEnglish).String(string(deploy.Mode))
	}

	if deploy.Interval > 0 {
		r.DeployInterval = deploy.Interval.String()
	}

	r.RecommendationsDeployConfig = deploy
}

func (r *ApplicationRow) SetRecommendationsConfiguration(config []applications.Configuration) {
	for i := range config {
		r.RecommendationsConfiguration = append(r.RecommendationsConfiguration, config[i])
	}
}

// ApplicationOutput wraps an application list for output.
type ApplicationOutput struct {
	Items []ApplicationRow `json:"items"`
}

// Add an application item to the output.
func (o *ApplicationOutput) Add(item *applications.ApplicationItem) error {
	o.Items = append(o.Items, *NewApplicationRow(item))
	return nil
}

// Len returns the number of items being output.
func (o *ApplicationOutput) Len() int { return len(o.Items) }

// Swap exchanges the order of the two specified items.
func (o *ApplicationOutput) Swap(i, j int) { o.Items[i], o.Items[j] = o.Items[j], o.Items[i] }

// Item returns the specified row value.
func (o *ApplicationOutput) Item(i int) Row { return &o.Items[i] }

// SortBy sorts the output by the named value.
func (o *ApplicationOutput) SortBy(key string) error { return SortBy(o, key) }

// ScenarioRow is a table row representation of a scenario.
type ScenarioRow struct {
	Name string `table:"name" csv:"name" json:"-"`

	applications.ScenarioItem `table:"-" csv:"-"`
}

func NewScenarioRow(item *applications.ScenarioItem) *ScenarioRow {
	return &ScenarioRow{
		Name: item.Name.String(),

		ScenarioItem: *item,
	}
}

func (r *ScenarioRow) Lookup(key string) (interface{}, bool) {
	switch SortByKey(key) {
	case "name":
		return r.Name, true
	default:
		return nil, false
	}
}

// ScenarioOutput wraps a scenario list for output.
type ScenarioOutput struct {
	Items []ScenarioRow `json:"items"`
}

// Add a scenario item to the output.
func (o *ScenarioOutput) Add(item *applications.ScenarioItem) error {
	o.Items = append(o.Items, *NewScenarioRow(item))
	return nil
}

// Len returns the number of items being output.
func (o *ScenarioOutput) Len() int { return len(o.Items) }

// Swap exchanges the order of the two specified items.
func (o *ScenarioOutput) Swap(i, j int) { o.Items[i], o.Items[j] = o.Items[j], o.Items[i] }

// Item returns the specified row value.
func (o *ScenarioOutput) Item(i int) Row { return &o.Items[i] }

// SortBy sorts the output by the named value.
func (o *ScenarioOutput) SortBy(key string) error { return SortBy(o, key) }

// RecommendationRow is a table row representation of a recommendation.
type RecommendationRow struct {
	Name              string `table:"name" csv:"name" json:"-"`
	DeployedAtMachine string `table:"-" csv:"last_deployed" json:"-"`
	DeployedAtHuman   string `table:"last_deployed" csv:"-" json:"-"`

	applications.RecommendationItem `table:"-" csv:"-"`
}

func NewRecommendationRow(item *applications.RecommendationItem) *RecommendationRow {
	return &RecommendationRow{
		Name:              item.Name,
		DeployedAtMachine: formatTime(item.DeployedAt, time.RFC3339),
		DeployedAtHuman:   formatTime(item.DeployedAt, "ago"),

		RecommendationItem: *item,
	}
}

func (r *RecommendationRow) Lookup(key string) (interface{}, bool) {
	switch SortByKey(key) {
	case "name":
		return r.Name, true
	case "last_deployed":
		return r.RecommendationItem.DeployedAt, true
	default:
		return nil, false
	}
}

// RecommendationOutput wraps a recommendation list for output.
type RecommendationOutput struct {
	Items []RecommendationRow `json:"items"`
}

// Add a recommendation item to the output.
func (o *RecommendationOutput) Add(item *applications.RecommendationItem) error {
	o.Items = append(o.Items, *NewRecommendationRow(item))
	return nil
}

// Len returns the number of items being output.
func (o *RecommendationOutput) Len() int { return len(o.Items) }

// Swap exchanges the order of the two specified items.
func (o *RecommendationOutput) Swap(i, j int) { o.Items[i], o.Items[j] = o.Items[j], o.Items[i] }

// Item returns the specified row value.
func (o *RecommendationOutput) Item(i int) Row { return &o.Items[i] }

// SortBy sorts the output by the named value.
func (o *RecommendationOutput) SortBy(key string) error { return SortBy(o, key) }

// ExperimentRow is a table row representation of an experiment.
type ExperimentRow struct {
	Name         string            `table:"name" csv:"name" json:"-"`
	DisplayName  string            `table:"Name,custom" json:"-"`
	Observations int64             `table:"observations,wide" csv:"observations" json:"-"`
	Labels       map[string]string `table:"labels,labels" csv:"label_,labels,flatten" json:"-"`

	experiments.ExperimentItem `table:"-" csv:"-"`
}

func NewExperimentRow(item *experiments.ExperimentItem) *ExperimentRow {
	return &ExperimentRow{
		Name:         item.Name.String(),
		DisplayName:  item.DisplayName,
		Observations: item.Observations,
		Labels:       item.Labels,

		ExperimentItem: *item,
	}
}

func (r *ExperimentRow) Lookup(key string) (interface{}, bool) {
	switch SortByKey(key) {
	case "name":
		return r.Name, true
	case "observations":
		return r.Observations, true
	default:
		return nil, false
	}
}

// ExperimentOutput wraps an experiment list for output.
type ExperimentOutput struct {
	Items []ExperimentRow `json:"items"`
}

// Add an experiment item to the output.
func (o *ExperimentOutput) Add(item *experiments.ExperimentItem) error {
	o.Items = append(o.Items, *NewExperimentRow(item))
	return nil
}

// Len returns the number of items being output.
func (o *ExperimentOutput) Len() int { return len(o.Items) }

// Swap exchanges the order of the two specified items.
func (o *ExperimentOutput) Swap(i, j int) { o.Items[i], o.Items[j] = o.Items[j], o.Items[i] }

// Item returns the specified row value.
func (o *ExperimentOutput) Item(i int) Row { return &o.Items[i] }

// SortBy sorts the output by the named value.
func (o *ExperimentOutput) SortBy(key string) error { return SortBy(o, key) }

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

func NewTrialRow(item *experiments.TrialItem) *TrialRow {
	var experiment string
	if item.Experiment != nil {
		experiment = item.Experiment.DisplayName
	}

	var name string
	if item.Experiment != nil && item.Experiment.Name != "" {
		name = fmt.Sprintf("%s/%03d", item.Experiment.Name, item.Number)
	} else {
		name = fmt.Sprintf("%03d", item.Number)
	}

	assignments := make(map[string]string, len(item.Assignments))
	for i := range item.Assignments {
		assignments[item.Assignments[i].ParameterName] = item.Assignments[i].Value.String()
	}

	values := make(map[string]string, len(item.Values))
	for i := range item.Values {
		values[item.Values[i].MetricName] = strconv.FormatFloat(item.Values[i].Value, 'f', -1, 64)
	}

	return &TrialRow{
		Experiment:     experiment,
		Name:           name,
		Number:         item.Number,
		Status:         cases.Title(language.English).String(string(item.Status)),
		FailureReason:  item.FailureReason,
		FailureMessage: item.FailureMessage,
		Assignments:    assignments,
		Values:         values,
		Labels:         item.Labels,

		TrialItem: *item,
	}
}

func (r *TrialRow) Lookup(key string) (interface{}, bool) {
	switch SortByKey(key) {
	case "name":
		return r.Name, true
	case "status":
		return r.Status, true
	case "failure_reason":
		return r.FailureReason, true
	default:
		return nil, false
	}
}

// TrialOutput wraps a trial list for output.
type TrialOutput struct {
	Items []TrialRow `json:"items"`
}

// Add a trial item to the output.
func (o *TrialOutput) Add(item *experiments.TrialItem) error {
	o.Items = append(o.Items, *NewTrialRow(item))
	return nil
}

// Len returns the number of items being output.
func (o *TrialOutput) Len() int { return len(o.Items) }

// Swap exchanges the order of the two specified items.
func (o *TrialOutput) Swap(i, j int) { o.Items[i], o.Items[j] = o.Items[j], o.Items[i] }

// Item returns the specified row value.
func (o *TrialOutput) Item(i int) Row { return &o.Items[i] }

// SortBy sorts the output by the named value.
func (o *TrialOutput) SortBy(key string) error { return SortBy(o, key) }

// ClusterRow is a table row representation of a cluster.
type ClusterRow struct {
	Name                   string `table:"name" csv:"name" json:"-"`
	DisplayName            string `table:"title" csv:"title" json:"title,omitempty"`
	OptimizeProVersion     string `table:"optimize_pro" csv:"optimize_pro_version" json:"-"`
	OptimizeLiveVersion    string `table:"optimize_live" csv:"optimize_live_version" json:"-"`
	PerformanceTestVersion string `table:"performance_test,wide" csv:"performance_test_version" json:"-"`
	KubernetesVersion      string `table:"kubernetes,wide" csv:"kubernetes_version" json:"-"`
	LastSeenMachine        string `table:"-" csv:"last_seen" json:"-"`
	LastSeenHuman          string `table:"last_seen" csv:"-" json:"-"`
	Age                    string `table:"age,wide" csv:"-" json:"-"`

	applications.ClusterItem `table:"-" csv:"-"`
}

func NewClusterRow(item *applications.ClusterItem) *ClusterRow {
	return &ClusterRow{
		Name:                   item.Name.String(),
		DisplayName:            item.Title(),
		OptimizeProVersion:     item.OptimizeProVersion,
		OptimizeLiveVersion:    item.OptimizeLiveVersion,
		PerformanceTestVersion: item.PerformanceTestVersion,
		KubernetesVersion:      item.KubernetesVersion,
		LastSeenMachine:        formatTime(item.LastSeen, time.RFC3339),
		LastSeenHuman:          formatTime(item.LastSeen, "ago"),
		Age:                    formatTime(item.CreatedAt, ""),

		ClusterItem: *item,
	}
}

func (r *ClusterRow) Lookup(key string) (interface{}, bool) {
	switch SortByKey(key) {
	case "name":
		return r.Name, true
	case "title":
		return r.DisplayName, true
	case "optimize_pro", "pro":
		return r.OptimizeProVersion, true
	case "optimize_live", "live":
		return r.OptimizeLiveVersion, true
	case "kubernetes":
		return r.KubernetesVersion, true
	case "last_seen":
		return r.ClusterItem.LastSeen, true
	case "age":
		return r.ClusterItem.CreatedAt, true
	default:
		return nil, false
	}
}

// ClusterOutput wraps a cluster list for output.
type ClusterOutput struct {
	Items []ClusterRow `json:"items"`
}

// Add a cluster item to the output.
func (o *ClusterOutput) Add(item *applications.ClusterItem) error {
	o.Items = append(o.Items, *NewClusterRow(item))
	return nil
}

// Len returns the number of items being output.
func (o *ClusterOutput) Len() int { return len(o.Items) }

// Swap exchanges the order of the two specified items.
func (o *ClusterOutput) Swap(i, j int) { o.Items[i], o.Items[j] = o.Items[j], o.Items[i] }

// Item returns the specified row value.
func (o *ClusterOutput) Item(i int) Row { return &o.Items[i] }

// SortBy sorts the output by the named value.
func (o *ClusterOutput) SortBy(key string) error { return SortBy(o, key) }

type ActivityRow struct {
	ID               string `table:"id" csv:"id" json:"-"`
	Title            string `table:"title" csv:"title" json:"-"`
	Tags             string `table:"tags" csv:"tags" json:"-"`
	ExternalURL      string `table:"reference" csv:"external_url" json:"-"`
	URL              string `table:"url,wide" csv:"url" json:"-"`
	FailureReason    string `table:"reason,wide" csv:"failure_reason" json:"-"`
	PublishedHuman   string `table:"published" csv:"-" json:"-"`
	PublishedMachine string `table:"-" csv:"published" json:"-"`

	applications.ActivityItem `table:"-" csv:"-"`
}

func NewActivityRow(item *applications.ActivityItem) *ActivityRow {
	var fr string
	if item.StormForge != nil {
		fr = item.StormForge.FailureReason
	}

	return &ActivityRow{
		ID:               item.ID,
		Title:            item.Title,
		Tags:             strings.Join(item.Tags, ", "),
		ExternalURL:      item.ExternalURL,
		URL:              item.URL,
		FailureReason:    fr,
		PublishedMachine: formatTime(&item.DatePublished, time.RFC3339),
		PublishedHuman:   formatTime(&item.DatePublished, "ago"),

		ActivityItem: *item,
	}
}

type ActivityOutput struct {
	Items []ActivityRow `json:"-"`

	applications.ActivityFeed
}

func (o *ActivityOutput) Add(item *applications.ActivityItem) {
	o.Items = append(o.Items, *NewActivityRow(item))
}

// Row represents a single row in the output.
type Row interface {
	// Lookup returns a named value on the row.
	Lookup(string) (interface{}, bool)
}

// Output represents an output list.
type Output interface {
	// Len returns the number of items in the output.
	Len() int
	// Swap exchanges the items at the specified indices.
	Swap(int, int)
	// Item returns the complete row.
	Item(int) Row

	// NOTE: there should also be an `Add(*item) error`-ish function
}

// SortBy sorts the supplied output using the named value on each row.
func SortBy(o Output, name string) error {
	if name == "" {
		return nil
	}

	n := o.Len()
	s := &sorter{
		Output: o,
		keys:   make([][]byte, n),
	}

	c := collate.New(language.AmericanEnglish, collate.Loose, collate.Numeric)
	buf := &collate.Buffer{}
	for i := 0; i < n; i++ {
		value, ok := s.Item(i).Lookup(name)
		if !ok {
			return fmt.Errorf("unknown sort-by key: %q", name)
		}

		if value == nil {
			s.keys[i] = c.KeyFromString(buf, "")
			continue
		}

		switch value := value.(type) {
		case string:
			s.keys[i] = c.KeyFromString(buf, value)
		case int:
			s.keys[i] = c.KeyFromString(buf, strconv.Itoa(value))
		case *time.Time:
			s.keys[i] = c.KeyFromString(buf, strconv.FormatInt(value.Unix(), 10))
		default:
			// If you get this panic, add support for the missing type!
			panic(fmt.Sprintf("unknown sort type %T on %T for %s", value, o, name))
		}
	}

	sort.Sort(s)
	return nil
}

// SortByKey normalizes the user supplied sort-by key.
func SortByKey(key string) string {
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ToLower(key)
	return key
}

type sorter struct {
	Output
	keys [][]byte
}

func (s *sorter) Less(i, j int) bool { return bytes.Compare(s.keys[i], s.keys[j]) == -1 }
func (s *sorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.Output.Swap(i, j)
}
