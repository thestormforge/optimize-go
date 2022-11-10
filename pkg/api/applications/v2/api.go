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
	"context"

	"github.com/thestormforge/optimize-go/pkg/api"
)

const (
	ErrApplicationInvalid     api.ErrorType = "application-invalid"
	ErrApplicationNotFound    api.ErrorType = "application-not-found"
	ErrApplicationExists      api.ErrorType = "application-exists"
	ErrScenarioInvalid        api.ErrorType = "scenario-invalid"
	ErrScenarioNotFound       api.ErrorType = "scenario-not-found"
	ErrScenarioExists         api.ErrorType = "scenario-exists"
	ErrScanInvalid            api.ErrorType = "scan-invalid"
	ErrActivityInvalid        api.ErrorType = "activity-invalid"
	ErrActivityRateLimited    api.ErrorType = "activity-rate-limited"
	ErrRecommendationInvalid  api.ErrorType = "recommendation-invalid"
	ErrRecommendationNotFound api.ErrorType = "recommendation-not-found"
	ErrClusterNotFound        api.ErrorType = "cluster-not-found"
	ErrRemoteWriteInvalid     api.ErrorType = "remote-write-invalid"
	ErrRateLimited            api.ErrorType = "rate-limit-exceeded"
)

// Subscriber describes a strategy for subscribing to feed notifications.
type Subscriber interface {
	// Subscribe initiates a subscription that continues for the lifetime of the context.
	Subscribe(ctx context.Context, ch chan<- ActivityItem) error
}

type API interface {
	// CheckEndpoint verifies we can talk to the backend.
	CheckEndpoint(ctx context.Context) (api.Metadata, error)

	// ListApplications gets a list of existing applications for an authorized request.
	ListApplications(ctx context.Context, q ApplicationListQuery) (ApplicationList, error)
	// ListApplicationsByPage returns single page of applications identified by the supplied URL.
	ListApplicationsByPage(ctx context.Context, u string) (ApplicationList, error)
	// CreateApplication creates a new application.
	CreateApplication(ctx context.Context, app Application) (api.Metadata, error)
	// CreateApplicationByName creates a new application.
	CreateApplicationByName(ctx context.Context, n ApplicationName, app Application) (api.Metadata, error)
	// GetApplication retrieves an application.
	GetApplication(ctx context.Context, u string) (Application, error)
	// GetApplicationByName retrieves an application.
	GetApplicationByName(ctx context.Context, n ApplicationName) (Application, error)
	// UpdateApplication updates an application.
	UpdateApplication(ctx context.Context, u string, app Application) (api.Metadata, error)
	// UpdateApplicationByName updates or creates an application.
	UpdateApplicationByName(ctx context.Context, n ApplicationName, app Application) (api.Metadata, error)
	// DeleteApplication deletes an application.
	DeleteApplication(ctx context.Context, u string) error

	// ListScenarios lists configured scenarios for an application.
	ListScenarios(ctx context.Context, u string, q ScenarioListQuery) (ScenarioList, error)
	// CreateScenario creates a scenario.
	CreateScenario(ctx context.Context, u string, scn Scenario) (api.Metadata, error)
	// CreateScenarioByName creates a scenario.
	CreateScenarioByName(ctx context.Context, u string, n ScenarioName, scn Scenario) (Scenario, error)
	// GetScenario retrieves a scenario.
	GetScenario(ctx context.Context, u string) (Scenario, error)
	// GetScenarioByName retrieves a scenario by name.
	GetScenarioByName(ctx context.Context, u string, n ScenarioName) (Scenario, error)
	// UpdateScenario updates or creates a scenario.
	UpdateScenario(ctx context.Context, u string, scn Scenario) (Scenario, error)
	// UpdateScenarioByName updates or creates a scenario.
	UpdateScenarioByName(ctx context.Context, u string, n ScenarioName, scn Scenario) (Scenario, error)
	// DeleteScenario deletes a scenario.
	DeleteScenario(ctx context.Context, u string) error
	// PatchScenario updates attributes on a scenario.
	PatchScenario(ctx context.Context, u string, scn Scenario) error

	// GetTemplate gets the application scenario template.
	GetTemplate(ctx context.Context, u string) (Template, error)
	// UpdateTemplate records or updates scenario template.
	UpdateTemplate(ctx context.Context, u string, s Template) error
	// PatchTemplate updates a partial scenario template.
	PatchTemplate(ctx context.Context, u string, s Template) error

	// ListActivity gets activity feed for an application.
	ListActivity(ctx context.Context, u string, q ActivityFeedQuery) (ActivityFeed, error)
	// CreateActivity creates application activity.
	CreateActivity(ctx context.Context, u string, a Activity) error
	// DeleteActivity resolves application activity.
	DeleteActivity(ctx context.Context, u string) error
	// PatchApplicationActivity updates application activity.
	PatchApplicationActivity(ctx context.Context, u string, a ActivityFailure) error

	// SubscribeActivity returns a subscriber for the activity feed.
	SubscribeActivity(ctx context.Context, q ActivityFeedQuery) (Subscriber, error)

	// CreateRecommendation creates an application recommendation using the most recently published values.
	CreateRecommendation(ctx context.Context, u string) (api.Metadata, error)
	// GetRecommendation retrieves a recommendation.
	GetRecommendation(ctx context.Context, u string) (Recommendation, error)
	// ListRecommendations lists recommendations and recommendation configuration for an application.
	ListRecommendations(ctx context.Context, u string) (RecommendationList, error)
	// PatchRecommendations updates recommendation configuration.
	PatchRecommendations(ctx context.Context, u string, details RecommendationList) error

	// GetCluster retrieves a cluster.
	GetCluster(ctx context.Context, u string) (Cluster, error)
	// GetClusterByName retrieves a cluster.
	GetClusterByName(ctx context.Context, n ClusterName) (Cluster, error)
	// ListClusters lists clusters.
	ListClusters(ctx context.Context, q ClusterListQuery) (ClusterList, error)
	// PatchCluster updates a cluster title.
	PatchCluster(ctx context.Context, u string, c ClusterTitle) error
	// DeleteCluster deletes a cluster.
	DeleteCluster(ctx context.Context, u string) error

	// RemoteWrite allows raw data to be compressed and written to the remote write endpoint.
	// TODO Instead of JSON or Protobuf bytes, this should take...something else
	RemoteWrite(ctx context.Context, body []byte) error
}
