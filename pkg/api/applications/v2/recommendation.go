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
	"encoding/json"
	"time"

	"github.com/thestormforge/optimize-go/pkg/api"
)

type Recommendation struct {
	api.Metadata `json:"-"`
	Name         string      `json:"name"`
	DeployedAt   *time.Time  `json:"deployedAt,omitempty"`
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
	DeployConfiguration *DeployConfiguration `json:"deploy,omitempty"`
	Configuration       []interface{}        `json:"configuration,omitempty"`
	Recommendations     []RecommendationItem `json:"recommendations,omitempty"`
}

type DeployConfiguration struct {
	Mode                   string           `json:"mode,omitempty"` // TODO Is this read-only?
	Interval               api.Duration     `json:"interval,omitempty"`
	Limits                 []LimitRangeItem `json:"limits,omitempty"`
	MaxRecommendationRatio *ResourceList    `json:"maxRecommendationRatio,omitempty"`
	Clusters               []string         `json:"clusters,omitempty"`
}

type LimitRangeItem struct {
	Type                 string        `json:"type,omitempty"`
	Max                  *ResourceList `json:"max,omitempty"`
	Min                  *ResourceList `json:"min,omitempty"`
	Default              *ResourceList `json:"default,omitempty"`
	MaxRequest           *ResourceList `json:"maxRequest,omitempty"`
	MinRequest           *ResourceList `json:"minRequest,omitempty"`
	MaxLimitRequestRatio *ResourceList `json:"maxLimitRequestRatio,omitempty"`
}

func (l *LimitRangeItem) UnmarshalJSON(bytes []byte) error {
	type t LimitRangeItem
	if err := json.Unmarshal(bytes, (*t)(l)); err != nil {
		return err
	}

	// Handle a legacy data migration lazily
	// NOTE: MinRequest/MaxRequest are required, so if they are both missing, swap with Min/Max
	if l.MaxRequest == nil && l.MinRequest == nil && l.Max != nil && l.Min != nil {
		l.MaxRequest, l.Max = l.Max, l.MaxRequest
		l.MinRequest, l.Min = l.Min, l.MinRequest
	}

	return nil
}

type ResourceList struct {
	CPU    *api.NumberOrString `json:"cpu,omitempty"`
	Memory *api.NumberOrString `json:"memory,omitempty"`
}
