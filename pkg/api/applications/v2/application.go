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
	"time"

	"github.com/thestormforge/optimize-go/pkg/api"
)

type Application struct {
	api.Metadata `json:"-"`
	Name         ApplicationName `json:"name"`
	DisplayName  string          `json:"title,omitempty"` // TODO This doesn't seem to get set
	Resources    []interface{}   `json:"resources,omitempty"`
	Cluster      string          `json:"cluster,omitempty"` // TODO This is write only?
	CreatedAt    time.Time       `json:"createdAt"`
}

// NOTE: Use `DisplayName` as the field since `Title()` is a function on the embedded `Metadata`.
var _ = Application{}.Title()

type ApplicationListQuery struct{ api.IndexQuery }

type ApplicationItem struct {
	Application
	// The number of scenarios associated with this application.
	ScenarioCount   int                 `json:"scenarioCount,omitempty"`
	LastDeployedAt  time.Time           `json:"lastDeployedAt,omitempty"`
	Recommendations RecommendationsMode `json:"recommendations,omitempty"`
}

type RecommendationsMode string

const (
	RecommendationsDisabled RecommendationsMode = "disabled"
	RecommendationsManual   RecommendationsMode = "manual"
	RecommendationsAuto     RecommendationsMode = "auto"
)

func (ai *ApplicationItem) UnmarshalJSON(b []byte) error {
	type t ApplicationItem
	return api.UnmarshalJSON(b, (*t)(ai))
}

type ApplicationList struct {
	// The application list metadata.
	api.Metadata `json:"-"`
	// The total number of items in the collection.
	TotalCount int `json:"totalCount,omitempty"`
	// The list of applications.
	Applications []ApplicationItem `json:"applications"`
}
