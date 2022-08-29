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

package recommendation

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

const (
	flagContainerResourcesSelector          = "selector"
	flagContainerResourcesInterval          = "container-resource-interval"
	flagContainerResourcesTargetUtilization = "container-resource-target-utilization"
	flagContainerResourcesTolerance         = "tolerance"
	flagContainerResourcesBoundsLimitsMax   = "max-limit"
	flagContainerResourcesBoundsLimitsMin   = "min-limit"
	flagContainerResourcesRequestsMax       = "max-request"
	flagContainerResourcesRequestsMin       = "min-request"
)

const (
	flagDeployMode                   = "mode"
	flagDeployInterval               = "interval"
	flagDeployMaxRecommendationRatio = "deploy-max-ratio"
	flagDeployCluster                = "cluster"
)

var (
	defaultDeployInterval = api.Duration(1 * time.Hour)
	validDeployModes      = []string{
		string(applications.RecommendationsManual),
		string(applications.RecommendationsAuto),
		string(applications.RecommendationsDisabled),
	}
)

// ContainerResourcesOptions contains options for building the recommender configuration
// for optimizing container resources.
type ContainerResourcesOptions struct {
	Selector          string
	Interval          time.Duration
	TargetUtilization map[string]string
	Tolerance         map[string]string
	BoundsLimitsMax   map[string]string
	BoundsLimitsMin   map[string]string
	BoundsRequestsMax map[string]string
	BoundsRequestsMin map[string]string
}

func (opts *ContainerResourcesOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.Selector, flagContainerResourcesSelector, opts.Selector, "`selector` for application resources which should have container resource optimization applied")
	cmd.Flags().DurationVar(&opts.Interval, flagContainerResourcesInterval, opts.Interval, "amount of `time` between container resource recommendation computations")
	cmd.Flags().StringToStringVar(&opts.TargetUtilization, flagContainerResourcesTargetUtilization, opts.TargetUtilization, "container resource target utilization as `resource=value`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.Tolerance, flagContainerResourcesTolerance, opts.Tolerance, "container resource tolerance as `resource=tolerance`; resource is one of: cpu|memory; tolerance is one of: low|medium|high")
	cmd.Flags().StringToStringVar(&opts.BoundsLimitsMax, flagContainerResourcesBoundsLimitsMax, opts.BoundsLimitsMax, "per-container resource max limits as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.BoundsLimitsMin, flagContainerResourcesBoundsLimitsMin, opts.BoundsLimitsMin, "per-container resource min limits as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.BoundsRequestsMax, flagContainerResourcesRequestsMax, opts.BoundsRequestsMax, "per-container resource max requests as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.BoundsRequestsMin, flagContainerResourcesRequestsMin, opts.BoundsRequestsMin, "per-container resource min requests as `resource=quantity`; resource is one of: cpu|memory")

	cmd.Flag(flagContainerResourcesInterval).Hidden = true
	cmd.Flag(flagContainerResourcesTargetUtilization).Hidden = true
}

func (opts *ContainerResourcesOptions) Apply(configuration *[]applications.Configuration) {
	lazyContainerResources := func() *applications.ContainerResources {
		if len(*configuration) == 0 {
			*configuration = append(*configuration, applications.Configuration{ContainerResources: &applications.ContainerResources{}})
		}
		if (*configuration)[0].ContainerResources == nil {
			(*configuration)[0].ContainerResources = &applications.ContainerResources{}
		}
		return (*configuration)[0].ContainerResources
	}

	if opts.Selector != "" {
		lazyContainerResources().Selector = opts.Selector
	}

	if opts.Interval > 0 {
		lazyContainerResources().Interval = api.Duration(opts.Interval)
	}

	if size := len(opts.TargetUtilization); size > 0 {
		targetUtilization := &applications.ResourceList{}
		for k, v := range opts.TargetUtilization {
			targetUtilization.Set(strings.ToLower(k), api.FromNumber(json.Number(v)))
		}
		lazyContainerResources().TargetUtilization = targetUtilization
	}

	if size := len(opts.Tolerance); size > 0 {
		tolerance := &applications.ResourceList{}
		for k, v := range opts.Tolerance {
			tolerance.Set(strings.ToLower(k), *(*api.NumberOrString)(applications.ToleranceFrom(v)))
		}
		lazyContainerResources().Tolerance = tolerance
	}

	bounds := &applications.Bounds{}
	lazyLimits := func() *applications.BoundsRange {
		if bounds.Limits == nil {
			bounds.Limits = &applications.BoundsRange{}
		}
		return bounds.Limits
	}
	if len(opts.BoundsLimitsMax) > 0 {
		limits := lazyLimits()
		if limits.Max == nil {
			limits.Max = &applications.ResourceList{}
		}
		for k, v := range opts.BoundsLimitsMax {
			limits.Max.Set(strings.ToLower(k), api.FromNumber(json.Number(v)))
		}
	}
	if len(opts.BoundsLimitsMin) > 0 {
		limits := lazyLimits()
		if limits.Min == nil {
			limits.Min = &applications.ResourceList{}
		}
		for k, v := range opts.BoundsLimitsMin {
			limits.Min.Set(strings.ToLower(k), api.FromNumber(json.Number(v)))
		}
	}

	lazyRequests := func() *applications.BoundsRange {
		if bounds.Requests == nil {
			bounds.Requests = &applications.BoundsRange{}
		}
		return bounds.Requests
	}
	if len(opts.BoundsRequestsMax) > 0 {
		requests := lazyRequests()
		if requests.Max == nil {
			requests.Max = &applications.ResourceList{}
		}
		for k, v := range opts.BoundsRequestsMax {
			requests.Max.Set(strings.ToLower(k), api.FromNumber(json.Number(v)))
		}
	}
	if len(opts.BoundsLimitsMin) > 0 {
		requests := lazyRequests()
		if requests.Min == nil {
			requests.Min = &applications.ResourceList{}
		}
		for k, v := range opts.BoundsLimitsMin {
			requests.Min.Set(strings.ToLower(k), api.FromNumber(json.Number(v)))
		}
	}
	if bounds.Limits != nil || bounds.Requests != nil {
		lazyContainerResources().Bounds = bounds
	}
}

type DeployConfigurationOptions struct {
	Mode                   string
	Interval               time.Duration
	MaxRecommendationRatio map[string]string
	Clusters               []string
}

func (opts *DeployConfigurationOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.Mode, flagDeployMode, opts.Mode, "deployment `mode`; one of: manual|auto|disabled")
	cmd.Flags().DurationVar(&opts.Interval, flagDeployInterval, opts.Interval, "desired amount of `time` between deployments")
	cmd.Flags().StringToStringVar(&opts.MaxRecommendationRatio, flagDeployMaxRecommendationRatio, opts.MaxRecommendationRatio, "limit the recommended/current value ratio as `resource=ratio`")
	cmd.Flags().StringArrayVar(&opts.Clusters, flagDeployCluster, opts.Clusters, "cluster `name` used for recommendations")

	cmd.Flag(flagDeployMaxRecommendationRatio).Hidden = true

	_ = cmd.RegisterFlagCompletionFunc(flagDeployMode, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return validDeployModes, cobra.ShellCompDirectiveDefault
	})
}

func (opts *DeployConfigurationOptions) Apply(deployConfiguration **applications.DeployConfiguration) {
	lazyDeployConfig := func() *applications.DeployConfiguration {
		if *deployConfiguration == nil {
			*deployConfiguration = &applications.DeployConfiguration{}
		}
		return *deployConfiguration
	}

	if opts.Mode != "" {
		lazyDeployConfig().Mode = applications.RecommendationsMode(opts.Mode)
	}

	if opts.Interval > 0 {
		lazyDeployConfig().Interval = api.Duration(opts.Interval)
	}

	if len(opts.MaxRecommendationRatio) > 0 {
		ratios := &applications.ResourceList{}
		for k, v := range opts.MaxRecommendationRatio {
			ratios.Set(strings.ToLower(k), api.FromString(v))
		}
		lazyDeployConfig().MaxRecommendationRatio = ratios
	}

	if len(opts.Clusters) > 0 {
		lazyDeployConfig().Clusters = opts.Clusters
	}
}

// Finish attempts to validate the requested changes.
func Finish(cmd *cobra.Command, appAPI applications.API, app applications.Application, recs applications.RecommendationList, patch *applications.RecommendationList) error {
	var errs ErrorList
	if recs.DeployConfiguration == nil {
		recs.DeployConfiguration = &applications.DeployConfiguration{}
	}

	// Determine the default mode
	var mode applications.RecommendationsMode
	if patch.DeployConfiguration != nil {
		switch patch.DeployConfiguration.Mode {
		case applications.RecommendationsManual, applications.RecommendationsAuto, applications.RecommendationsDisabled, "":
			mode = patch.DeployConfiguration.Mode
		default:
			errs = append(errs, &Error{
				Message:        fmt.Sprintf("invalid deploy mode: %s", patch.DeployConfiguration.Mode),
				FixCommand:     cmd.CommandPath(),
				FixFlag:        flagDeployMode,
				FixValidValues: validDeployModes,
			})
		}
	}
	if mode == "" {
		if !recs.DeployConfiguration.Mode.Enabled() {
			if patch.DeployConfiguration == nil {
				patch.DeployConfiguration = &applications.DeployConfiguration{}
			}
			patch.DeployConfiguration.Mode = applications.RecommendationsManual
			mode = applications.RecommendationsManual
		} else {
			mode = recs.DeployConfiguration.Mode
		}
	}

	// Reject an empty patch
	if patch.DeployConfiguration == nil && len(patch.Configuration) == 0 {
		errs = append(errs, &Error{
			Message:    "missing configuration options",
			FixCommand: cmd.CommandPath(),
		})
		return errs.Err()
	}

	// Validate or default the deploy interval
	deployInterval := patch.DeployConfiguration.Interval
	if deployInterval == 0 {
		deployInterval = recs.DeployConfiguration.Interval
	}
	switch {
	case deployInterval < 0:
		errs = append(errs, &Error{
			Message:        fmt.Sprintf("invalid deploy interval: %s", deployInterval),
			FixCommand:     cmd.CommandPath(),
			FixFlag:        flagDeployInterval,
			FixValidValues: []string{(deployInterval * -1).String()}, // It was less than zero...
		})
	case deployInterval == 0:
		if mode.Enabled() {
			patch.DeployConfiguration.Interval = defaultDeployInterval
		}
	}

	// TODO MaxRecommendationRatio

	// A cluster is required to enable recommendations
	if mode.Enabled() && len(recs.DeployConfiguration.Clusters)+len(patch.DeployConfiguration.Clusters) == 0 {
		q := applications.ClusterListQuery{}
		q.SetModules(applications.ClusterRecommendations)
		list, err := appAPI.ListClusters(cmd.Context(), q)
		if err != nil {
			return err
		}

		names := make([]string, 0, len(list.Items))
		for i := range list.Items {
			names = append(names, list.Items[i].Name.String())
		}

		if len(names) == 1 {
			patch.DeployConfiguration.Clusters = names
		} else {
			errs = append(errs, &Error{
				Message:        "missing deploy cluster",
				FixCommand:     cmd.CommandPath(),
				FixFlag:        flagDeployCluster,
				FixValidValues: names,
			})
		}
	}

	for i := range patch.Configuration {
		if patch.Configuration[i].ContainerResources == nil {
			continue
		}

		// Validate bounds
		bounds := patch.Configuration[i].ContainerResources.Bounds
		if bounds == nil {
			bounds = &applications.Bounds{}
		}
		var defaultBounds *applications.Bounds
		if len(recs.Configuration) > i && recs.Configuration[i].ContainerResources != nil {
			defaultBounds = recs.Configuration[i].ContainerResources.Bounds
		}
		if defaultBounds == nil {
			defaultBounds = &applications.Bounds{}
		}
		limits := func(l *applications.Bounds) *applications.BoundsRange {
			if l.Limits != nil {
				return l.Limits
			}
			return &applications.BoundsRange{}
		}
		requests := func(l *applications.Bounds) *applications.BoundsRange {
			if l.Requests != nil {
				return l.Requests
			}
			return &applications.BoundsRange{}
		}

		errs = append(errs, checkResourceList(
			mode, "limit",
			limits(bounds).Min, limits(defaultBounds).Min,
			limits(bounds).Max, limits(defaultBounds).Max,
			cmd.CommandPath(), flagContainerResourcesBoundsLimitsMin, flagContainerResourcesBoundsLimitsMax,
		)...)

		errs = append(errs, checkResourceList(
			mode, "request",
			requests(bounds).Min, requests(defaultBounds).Min,
			requests(bounds).Max, requests(defaultBounds).Max,
			cmd.CommandPath(), flagContainerResourcesRequestsMin, flagContainerResourcesRequestsMax,
		)...)
	}

	// Application resources are required to enable recommendations
	if mode.Enabled() && len(app.Resources) == 0 {
		errs = append(errs, &Error{
			Message:    "missing application resources",
			FixCommand: strings.Join([]string{cmd.Root().Name(), "edit", "application", app.Name.String()}, " "),
			FixFlag:    "namespace",
		})
	}

	return errs.Err()
}

// checkResourceList looks through the resource lists for errors.
func checkResourceList(mode applications.RecommendationsMode, name string, minList, defMinList, maxList, defMaxList *applications.ResourceList, fixCommand, fixFlagMin, fixFlagMax string) ErrorList {
	var errs ErrorList

	required := mode.Enabled() && name == "request"

	checkResource := func(resourceName, minmax string, value *api.NumberOrString, fixFlag string) bool {
		if value == nil {
			if required {
				errs = append(errs, &Error{
					Message:        fmt.Sprintf("missing %s container %s for %s", minmax, name, resourceName),
					FixCommand:     fixCommand,
					FixFlag:        fixFlag,
					FixValidValues: []string{resourceName + "=VALUE"},
				})
			}
			return false
		}

		if value.IsString || value.Int64Value() < 0 {
			errs = append(errs, &Error{
				Message:        fmt.Sprintf("invalid %s container %s for %s: %s", minmax, name, resourceName, value),
				FixCommand:     fixCommand,
				FixFlag:        fixFlag,
				FixValidValues: []string{resourceName + "=VALUE"},
			})
			return false
		}

		return true
	}

	for _, resourceName := range []string{"cpu", "memory"} {
		min := minList.Get(resourceName)
		if min == nil {
			min = defMinList.Get(resourceName)
		}

		max := maxList.Get(resourceName)
		if max == nil {
			max = defMaxList.Get(resourceName)
		}

		minOk := checkResource(resourceName, "minimum", min, fixFlagMin)
		maxOk := checkResource(resourceName, "maximum", max, fixFlagMax)
		if minOk && maxOk && min.Int64Value() > max.Int64Value() {
			errs = append(errs, &Error{
				Message:    fmt.Sprintf("invalid container %s range for %s: %s-%s", name, resourceName, min, max),
				FixCommand: fixCommand,
			})
		}
	}

	return errs
}

type Error struct {
	Message        string
	FixCommand     string
	FixFlag        string
	FixValidValues []string
}

func (e *Error) Error() string {
	return e.Message
}

type ErrorList []*Error

func (el ErrorList) Err() error {
	if len(el) == 0 {
		return nil
	}
	return el
}

func (el ErrorList) Error() string {
	if len(el) == 0 {
		panic("use ErrorList.Err() to ignore an empty error list")
	}

	var msgs []string
	suggestions := make(map[string][]string)
	for _, err := range el {
		msgs = append(msgs, err.Error())

		if err.FixCommand != "" && err.FixFlag != "" {
			// TODO Can we batch up flags that can be combined, e.g. "--max-request x=y --max-request a=b" => "--max-request x=y,a=b"?
			suggestions[err.FixCommand] = append(suggestions[err.FixCommand], "--"+err.FixFlag)
			if len(err.FixValidValues) > 0 {
				suggestions[err.FixCommand] = append(suggestions[err.FixCommand], strings.Join(err.FixValidValues, "|"))
			}
		}
	}

	msg := strings.Join(msgs, "\n")

	if len(suggestions) > 0 {
		msg += "\n\nTry running:"
		// TODO Do a sorted iteration instead?
		for cmd, args := range suggestions {
			msg += "\n  " + cmd
			if len(args) > 0 {
				msg += " " + strings.Join(args, " ")
			}
		}
	}

	return msg
}
