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
	Target             TargetRef     `json:"target,omitempty"`
	ContainerResources []interface{} `json:"containerResources,omitempty"`
}

type TargetRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Workload  string `json:"workload,omitempty"`
}

type RecommendationDetails struct {
	api.Metadata        `json:"-"`
	DeployConfiguration DeployConfiguration `json:"deploy,omitempty"`
	Configuration       []interface{}       `jsonn:"configuration,omitempty"`
	Recommendations     api.Metadata        `json:"-"`
}

type DeployConfiguration struct {
	Interval               string      `json:"interval,omitempty"`
	Limits                 interface{} `json:"limits,omitempty"`
	MaxRecommendationRatio interface{} `json:"maxRecommendationRatio,omitempty"`
}
