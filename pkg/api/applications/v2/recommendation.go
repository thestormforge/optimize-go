/*
Copyright 2022 GramLabs, Inc.

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

type Recommendation struct {
	api.Metadata `json:"-"`
	DeployedAt   time.Time   `json:"deployedAt,omitempty"`
	Parameters   []Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	Target             TargetRef     `json:"target"`
	ContainerResources []interface{} `json:"containerResources"`
}

type TargetRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Workload  string `json:"workload,omitempty"`
}

type RecommendationItem struct {
	Recommendation
}

func (l *RecommendationItem) UnmarshalJSON(b []byte) error {
	type t RecommendationItem
	return api.UnmarshalJSON(b, (*t)(l))
}

type RecommendationList struct {
	api.Metadata        `json:"-"`
	DeployConfiguration DeployConfiguration  `json:"deploy,omitempty"`
	Configuration       []interface{}        `json:"configuration,omitempty"`
	Recommendations     []RecommendationItem `json:"recommendations,omitempty"`
}

type DeployConfiguration struct {
	Interval               string      `json:"interval,omitempty"`
	Limits                 interface{} `json:"limits,omitempty"`
	MaxRecommendationRatio interface{} `json:"maxRecommendationRatio,omitempty"`
}
