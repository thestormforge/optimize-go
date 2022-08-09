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
	"strings"
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
	Configuration       []Configuration      `json:"configuration,omitempty"`
	Recommendations     []RecommendationItem `json:"recommendations,omitempty"`
}

type DeployConfiguration struct {
	Mode                   RecommendationsMode `json:"mode,omitempty"`
	Interval               api.Duration        `json:"interval,omitempty"`
	Limits                 []LimitRangeItem    `json:"limits,omitempty"`
	MaxRecommendationRatio *ResourceList       `json:"maxRecommendationRatio,omitempty"`
	Clusters               []string            `json:"clusters,omitempty"`
}

func (dc *DeployConfiguration) GetLimits(t string) *LimitRangeItem {
	for i := range dc.Limits {
		if t == dc.Limits[i].Type {
			return &dc.Limits[i]
		}
	}
	return nil
}

type Configuration struct {
	ContainerResources *ContainerResources `json:"containerResources,omitempty"`
}

type ContainerResources struct {
	Selector          string        `json:"selector,omitempty"`
	Interval          api.Duration  `json:"interval,omitempty"`
	TargetUtilization *ResourceList `json:"targetUtilization,omitempty"`
	Tolerance         *ResourceList `json:"tolerance,omitempty"`
}

type LimitRangeItem struct {
	Type string        `json:"type,omitempty"`
	Max  *ResourceList `json:"max,omitempty"`
	Min  *ResourceList `json:"min,omitempty"`
}

type ResourceList struct {
	CPU    *api.NumberOrString `json:"cpu,omitempty"`
	Memory *api.NumberOrString `json:"memory,omitempty"`
}

func (rl *ResourceList) Get(name string) *api.NumberOrString {
	if rl != nil {
		switch name {
		case "cpu":
			return rl.CPU
		case "memory":
			return rl.Memory
		}
	}
	return nil
}

func (rl *ResourceList) Set(name string, value api.NumberOrString) {
	switch name {
	case "cpu":
		rl.CPU = &value
	case "memory":
		rl.Memory = &value
	}
}

// NOTE: tolerance is a number or string type to allow it in a resource list

type Tolerance api.NumberOrString

func ToleranceFrom(s string) *Tolerance {
	switch strings.ToLower(s) {
	case "low", "l":
		return &Tolerance{StrVal: "low", IsString: true}
	case "medium", "med", "m":
		return &Tolerance{StrVal: "medium", IsString: true}
	case "high", "h":
		return &Tolerance{StrVal: "high", IsString: true}
	}
	return nil
}
