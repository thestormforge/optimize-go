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

type Cluster struct {
	api.Metadata           `json:"-"`
	Name                   ClusterName `json:"name"`
	CreatedAt              time.Time   `json:"created"`
	OptimizeProVersion     string      `json:"optimizeProVersion,omitempty"`
	OptimizeLiveVersion    string      `json:"optimizeLiveVersion,omitempty"`
	PerformanceTestVersion string      `json:"performanceTestVersion,omitempty"`
	KubernetesVersion      string      `json:"kubernetesVersion,omitempty"`
	LastSeen               time.Time   `json:"lastSeen"`
}

type ClusterItem struct {
	Cluster
}

func (ci *ClusterItem) UnmarshalJSON(b []byte) error {
	type t ClusterItem
	return api.UnmarshalJSON(b, (*t)(ci))
}

type ClusterList struct {
	// The cluster list metadata.
	api.Metadata `json:"-"`
	// The total number of items in the collection.
	TotalCount int `json:"totalCount,omitempty"`
	// The list of clusters.
	Items []ClusterItem `json:"items"`
}

type ClusterTitle struct {
	Title string `json:"title"`
}
