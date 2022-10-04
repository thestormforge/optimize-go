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
	flagHPATargetUtilizationMax             = "hpa-max-target-utilization"
	flagHPATargetUtilizationMin             = "hpa-min-target-utilization"
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

// ConfigurationOptions contains options for building the recommender configuration
// for optimizing container resources.
type ConfigurationOptions struct {
	Selector                string
	Interval                time.Duration
	TargetUtilization       map[string]string
	Tolerance               map[string]string
	BoundsLimitsMax         map[string]string
	BoundsLimitsMin         map[string]string
	BoundsRequestsMax       map[string]string
	BoundsRequestsMin       map[string]string
	HPATargetUtilizationMax map[string]string
	HPATargetUtilizationMin map[string]string
}

func (opts *ConfigurationOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.Selector, flagContainerResourcesSelector, opts.Selector, "`selector` for application resources which should have container resource optimization applied")
	cmd.Flags().DurationVar(&opts.Interval, flagContainerResourcesInterval, opts.Interval, "amount of `time` between container resource recommendation computations")
	cmd.Flags().StringToStringVar(&opts.TargetUtilization, flagContainerResourcesTargetUtilization, opts.TargetUtilization, "container resource target utilization as `resource=value`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.Tolerance, flagContainerResourcesTolerance, opts.Tolerance, "container resource tolerance as `resource=tolerance`; resource is one of: cpu|memory; tolerance is one of: low|medium|high")
	cmd.Flags().StringToStringVar(&opts.BoundsLimitsMax, flagContainerResourcesBoundsLimitsMax, opts.BoundsLimitsMax, "per-container resource max limits as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.BoundsLimitsMin, flagContainerResourcesBoundsLimitsMin, opts.BoundsLimitsMin, "per-container resource min limits as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.BoundsRequestsMax, flagContainerResourcesRequestsMax, opts.BoundsRequestsMax, "per-container resource max requests as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.BoundsRequestsMin, flagContainerResourcesRequestsMin, opts.BoundsRequestsMin, "per-container resource min requests as `resource=quantity`; resource is one of: cpu|memory")
	cmd.Flags().StringToStringVar(&opts.HPATargetUtilizationMax, flagHPATargetUtilizationMax, opts.HPATargetUtilizationMax, "per-hpa resource max target utilization as `resource=value`; resource is one of: cpu")
	cmd.Flags().StringToStringVar(&opts.HPATargetUtilizationMin, flagHPATargetUtilizationMin, opts.HPATargetUtilizationMin, "per-hpa resource min target utilization as `resource=value`; resource is one of: cpu")

	cmd.Flag(flagContainerResourcesInterval).Hidden = true
	cmd.Flag(flagContainerResourcesTargetUtilization).Hidden = true
}

func (opts *ConfigurationOptions) Apply(configuration *[]applications.Configuration) {

	lazyConfig := func() *applications.Configuration {
		if len(*configuration) == 0 {
			*configuration = append(*configuration, applications.Configuration{ContainerResources: &applications.ContainerResources{}, HPAResources: &applications.HPAResources{}})
		}
		return &(*configuration)[0]
	}

	lazyContainerResources := func() *applications.ContainerResources {
		config := lazyConfig()
		if config.ContainerResources == nil {
			config.ContainerResources = &applications.ContainerResources{}
		}
		return config.ContainerResources
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
			targetUtilization.Set(strings.ToLower(k), api.FromValue(v))
		}
		lazyContainerResources().TargetUtilization = targetUtilization
	}

	if size := len(opts.Tolerance); size > 0 {
		tolerance := &applications.ResourceList{}
		for k, v := range opts.Tolerance {
			tolerance.Set(strings.ToLower(k), api.NumberOrString(applications.ToleranceFrom(v)))
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
			limits.Max.Set(strings.ToLower(k), api.FromValue(v))
		}
	}
	if len(opts.BoundsLimitsMin) > 0 {
		limits := lazyLimits()
		if limits.Min == nil {
			limits.Min = &applications.ResourceList{}
		}
		for k, v := range opts.BoundsLimitsMin {
			limits.Min.Set(strings.ToLower(k), api.FromValue(v))
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
			requests.Max.Set(strings.ToLower(k), api.FromValue(v))
		}
	}
	if len(opts.BoundsRequestsMin) > 0 {
		requests := lazyRequests()
		if requests.Min == nil {
			requests.Min = &applications.ResourceList{}
		}
		for k, v := range opts.BoundsRequestsMin {
			requests.Min.Set(strings.ToLower(k), api.FromValue(v))
		}
	}
	if bounds.Limits != nil || bounds.Requests != nil {
		lazyContainerResources().Bounds = bounds
	}

	lazyHPAResources := func() *applications.HPAResources {
		config := lazyConfig()
		if config.HPAResources == nil {
			config.HPAResources = &applications.HPAResources{}
		}
		return config.HPAResources
	}

	hpaBounds := &applications.Bounds{}
	lazyTargetUtilization := func() *applications.BoundsRange {
		if hpaBounds.TargetUtilization == nil {
			hpaBounds.TargetUtilization = &applications.BoundsRange{}
		}
		return hpaBounds.TargetUtilization
	}
	if len(opts.HPATargetUtilizationMax) > 0 {
		targetUtilization := lazyTargetUtilization()
		if targetUtilization.Max == nil {
			targetUtilization.Max = &applications.ResourceList{}
		}
		for k, v := range opts.HPATargetUtilizationMax {
			targetUtilization.Max.Set(strings.ToLower(k), api.FromValue(v))
		}
	}
	if len(opts.HPATargetUtilizationMin) > 0 {
		targetUtilization := lazyTargetUtilization()
		if targetUtilization.Min == nil {
			targetUtilization.Min = &applications.ResourceList{}
		}
		for k, v := range opts.HPATargetUtilizationMin {
			targetUtilization.Min.Set(strings.ToLower(k), api.FromValue(v))
		}
	}
	if hpaBounds.TargetUtilization != nil {
		lazyHPAResources().Bounds = hpaBounds
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

	// NOTE: The service will not merge configurations; do it here instead
	// NOTE: Only work with index `Configuration[0]` because we can't make things line up otherwise
	if len(patch.Configuration) > 0 {
		if len(recs.Configuration) > 0 {
			configuration, err := applications.MergeConfigurations(&recs.Configuration[0], &patch.Configuration[0])
			if err != nil {
				return err
			}
			patch.Configuration[0] = *configuration
		}

		// Validate bounds
		bounds := patch.Configuration[0].ContainerResources.Bounds
		if bounds == nil {
			bounds = &applications.Bounds{}
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
		targetUtilization := func(l *applications.Bounds) *applications.BoundsRange {
			if l.TargetUtilization != nil {
				return l.TargetUtilization
			}
			return &applications.BoundsRange{}
		}

		errs = append(errs, checkResourceList(
			mode, "limit",
			limits(bounds).Min, limits(bounds).Max,
			cmd.CommandPath(), flagContainerResourcesBoundsLimitsMin, flagContainerResourcesBoundsLimitsMax,
		)...)

		errs = append(errs, checkResourceList(
			mode, "request",
			requests(bounds).Min, requests(bounds).Max,
			cmd.CommandPath(), flagContainerResourcesRequestsMin, flagContainerResourcesRequestsMax,
		)...)

		errs = append(errs, checkResourceList(
			mode, "target-utilization",
			targetUtilization(bounds).Min, targetUtilization(bounds).Max,
			cmd.CommandPath(), flagHPATargetUtilizationMin, flagHPATargetUtilizationMax,
		)...)
	}

	// Application resources are required to enable recommendations
	if mode.Enabled() && len(app.Resources) == 0 {
		errs = append(errs, &Error{
			Message:        "missing application resources",
			FixCommand:     strings.Join([]string{cmd.Root().Name(), "edit", "application", app.Name.String()}, " "),
			FixFlag:        "namespace",
			FixValidValues: []string{"default"},
		})
	}

	return errs.Err()
}

// checkResourceList looks through the resource lists for errors.
func checkResourceList(mode applications.RecommendationsMode, name string, minList, maxList *applications.ResourceList, fixCommand, fixFlagMin, fixFlagMax string) ErrorList {
	var errs ErrorList
	resourceAsPercentage := name == "target-utilization"

	// minmax=minimum|maximum, name=request|limit|target-utilization, resourceName=cpu|memory

	checkResource := func(resourceName, minmax string, value *api.NumberOrString, fixFlag string) bool {
		if value == nil {
			// Enforce required values
			if mode.Enabled() && name == "request" {
				errs = append(errs, &Error{
					Message:        fmt.Sprintf("missing %s container %s for %s", minmax, name, resourceName),
					FixCommand:     fixCommand,
					FixFlag:        fixFlag,
					FixValidValues: []string{fmt.Sprintf("%s=%s", resourceName, "VALUE")},
				})
			}

			// Even if it is allowed, we can't use it to compare to other values
			return false
		}

		// Require that the value convert to quantity that is NOT negative ('signbit == true' means negative)
		if q := value.Quantity(); q == nil || q.Signbit() {
			errs = append(errs, &Error{
				Message:        fmt.Sprintf("invalid %s container %s for %s: %s", minmax, name, resourceName, value),
				FixCommand:     fixCommand,
				FixFlag:        fixFlag,
				FixValidValues: []string{fmt.Sprintf("%s=%s", resourceName, "VALUE")},
			})
			return false
		}

		// Help prevent misconfiguration
		if err := LikelyInvalid(resourceName, value, resourceAsPercentage); err != nil {
			errs = append(errs, &Error{
				Message:        fmt.Sprintf("invalid %s container %s for %s: %s", minmax, name, resourceName, err.Error()),
				FixCommand:     fixCommand,
				FixFlag:        fixFlag,
				FixValidValues: []string{fmt.Sprintf("%s=%s", resourceName, "VALUE")},
			})
			return false
		}

		return true
	}

	for _, resourceName := range []string{"cpu", "memory"} {
		if resourceAsPercentage && resourceName != "cpu" {
			continue
		}

		min := minList.Get(resourceName)
		max := maxList.Get(resourceName)

		minOk := checkResource(resourceName, "minimum", min, fixFlagMin)
		maxOk := checkResource(resourceName, "maximum", max, fixFlagMax)

		// Make sure max is greater than or equal to min
		if minOk && maxOk && QuantityLess(max, min) {
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
