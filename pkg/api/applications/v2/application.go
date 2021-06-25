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
	"github.com/thestormforge/optimize-go/pkg/api"
)

// NewApplicationName returns an application name for a given string.
func NewApplicationName(n string) ApplicationName {
	return applicationName(n)
}

type applicationName string

func (n applicationName) Name() string   { return string(n) }
func (n applicationName) String() string { return string(n) }

type Application struct {
	api.Metadata `json:"-"`
	Name_        string        `json:"name"`
	DisplayName  string        `json:"title,omitempty"`
	Resources    []interface{} `json:"resources,omitempty"`
}

func (a *Application) Name() string {
	return a.Name_
}

type ApplicationListQuery struct{ api.IndexQuery }

type ApplicationItem struct {
	Application
	// The number of scenarios associated with this application.
	ScenarioCount int `json:"scenarioCount,omitempty"`
}

func (l *ApplicationItem) UnmarshalJSON(b []byte) error { return api.UnmarshalJSON(b, l) }

type ApplicationList struct {
	// The application list metadata.
	api.Metadata `json:"-"`
	// The total number of items in the collection.
	TotalCount int `json:"totalCount,omitempty"`
	// The list of applications.
	Applications []ApplicationItem `json:"applications,omitempty"`
}
