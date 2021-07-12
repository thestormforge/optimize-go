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
	ErrApplicationInvalid  api.ErrorType = "application-invalid"
	ErrApplicationNotFound api.ErrorType = "application-not-found"
	ErrScenarioInvalid     api.ErrorType = "scenario-invalid"
	ErrScenarioNotFound    api.ErrorType = "scenario-not-found"
	ErrScanInvalid         api.ErrorType = "scan-invalid"
	ErrActivityInvalid     api.ErrorType = "activity-invalid"
	ErrActivityRateLimited api.ErrorType = "activity-rate-limited"
)

// Subscriber describes a strategy for subscribing to feed notifications.
type Subscriber interface {
	// Subscribe initiates a subscription that continues for the lifetime of the context.
	Subscribe(ctx context.Context, ch chan<- ActivityItem)
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
	// GetApplication retrieves an application.
	GetApplication(ctx context.Context, u string) (Application, error)
	// GetApplicationByName retrieves an application.
	GetApplicationByName(ctx context.Context, n ApplicationName) (Application, error)
	// UpsertApplication updates or creates an application.
	UpsertApplication(ctx context.Context, u string, app Application) (api.Metadata, error)
	// UpsertApplicationByName updates or creates an application.
	UpsertApplicationByName(ctx context.Context, n ApplicationName, app Application) (api.Metadata, error)
	// DeleteApplication deletes an application.
	DeleteApplication(ctx context.Context, u string) error

	// ListScenarios lists configured scenarios for an application.
	ListScenarios(ctx context.Context, u string, q ScenarioListQuery) (ScenarioList, error)
	// CreateScenario creates a scenario.
	CreateScenario(ctx context.Context, u string, scn Scenario) (api.Metadata, error)
	// GetScenario retrieves a scenario.
	GetScenario(ctx context.Context, u string) (Scenario, error)
	// UpsertScenario updates or creates a scenario with the URL scenario name.
	UpsertScenario(ctx context.Context, u string, scn Scenario) (Scenario, error)
	// DeleteScenario deletes a scenario.
	DeleteScenario(ctx context.Context, u string) error
	// PatchScenario updates attributes on a scenario.
	PatchScenario(ctx context.Context, u string, scn Scenario) error

	// GetScan gets application scenario scan.
	GetScan(ctx context.Context, u string) (Scan, error)
	// UpdateScan records or updates cluster scan results.
	UpdateScan(ctx context.Context, u string, s Scan) error
	// PatchScan updates partial cluster scan results.
	PatchScan(ctx context.Context, u string, s Scan) error

	// ListActivity gets activity feed for an application.
	ListActivity(ctx context.Context, u string, q ActivityFeedQuery) (ActivityFeed, error)
	// CreateActivity creates application activity.
	CreateActivity(ctx context.Context, u string, a Activity) error
	// DeleteActivity resolves application activity.
	DeleteActivity(ctx context.Context, u string) error
	// GetApplicationActivity retrieves an application activity item by ID.
	GetApplicationActivity(ctx context.Context, u string) (Activity, error)
	// UpdateApplicationActivity updates application activity.
	UpdateApplicationActivity(ctx context.Context, u string, a Activity) error

	// SubscribeActivity returns a subscriber for the activity feed.
	SubscribeActivity(ctx context.Context, q ActivityFeedQuery) (Subscriber, error)
}
