/*
Copyright 2020 GramLabs, Inc.

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

package v1alpha1

import (
	"context"

	"github.com/thestormforge/optimize-go/pkg/api"
)

const (
	ErrExperimentNameInvalid  api.ErrorType = "experiment-name-invalid"
	ErrExperimentNameConflict api.ErrorType = "experiment-name-conflict"
	ErrExperimentInvalid      api.ErrorType = "experiment-invalid"
	ErrExperimentNotFound     api.ErrorType = "experiment-not-found"
	ErrExperimentStopped      api.ErrorType = "experiment-stopped"
	ErrTrialInvalid           api.ErrorType = "trial-invalid"
	ErrTrialUnavailable       api.ErrorType = "trial-unavailable"
	ErrTrialNotFound          api.ErrorType = "trial-not-found"
	ErrTrialAlreadyReported   api.ErrorType = "trial-already-reported"
)

type Server struct {
	api.Metadata `json:"-"`
}

// API provides bindings for the supported endpoints
type API interface {
	Options(context.Context) (Server, error)

	GetAllExperiments(context.Context, ExperimentListQuery) (ExperimentList, error)
	GetAllExperimentsByPage(context.Context, string) (ExperimentList, error)
	GetExperimentByName(context.Context, ExperimentName) (Experiment, error)
	GetExperiment(context.Context, string) (Experiment, error)
	CreateExperimentByName(context.Context, ExperimentName, Experiment) (Experiment, error)
	CreateExperiment(context.Context, string, Experiment) (Experiment, error)
	DeleteExperiment(context.Context, string) error
	LabelExperiment(context.Context, string, ExperimentLabels) error

	GetAllTrials(context.Context, string, TrialListQuery) (TrialList, error)
	CreateTrial(context.Context, string, TrialAssignments) (TrialAssignments, error)
	NextTrial(context.Context, string) (TrialAssignments, error)
	ReportTrial(context.Context, string, TrialValues) error
	AbandonRunningTrial(context.Context, string) error
	LabelTrial(context.Context, string, TrialLabels) error
}
