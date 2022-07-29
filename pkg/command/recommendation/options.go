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
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thestormforge/optimize-go/pkg/api"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
)

// TODO There is a shocking lack of validation here

// ContainerResourcesOptions contains options for building the recommender configuration
// for optimizing container resources.
type ContainerResourcesOptions struct {
	Selector          string
	Interval          time.Duration
	TargetUtilization map[string]string
	Tolerance         map[string]string
}

func (opts *ContainerResourcesOptions) Apply(configuration *[]interface{}) {
	containerResources := map[string]interface{}{}

	if opts.Selector != "" {
		containerResources["selector"] = opts.Selector
	}

	if opts.Interval > 0 {
		containerResources["interval"] = opts.Interval.String()
	}

	if size := len(opts.TargetUtilization); size > 0 {
		targetUtilization := make(map[string]interface{}, size)
		for k, v := range opts.TargetUtilization {
			switch strings.ToLower(k) {
			case "cpu":
				targetUtilization["cpu"] = json.Number(v)
			case "memory":
				targetUtilization["memory"] = json.Number(v)
			default:
				targetUtilization[k] = json.Number(v)
			}
		}
		containerResources["targetUtilization"] = targetUtilization
	}

	if size := len(opts.Tolerance); size > 0 {
		tolerance := make(map[string]interface{}, size)
		for k, v := range opts.Tolerance {
			switch strings.ToLower(k) {
			case "cpu":
				tolerance["cpu"] = v
			case "memory":
				tolerance["memory"] = v
			default:
				tolerance[k] = v
			}
		}
		containerResources["tolerance"] = tolerance
	}

	if len(containerResources) == 0 {
		return
	}

	for i := range *configuration {
		item, ok := (*configuration)[i].(map[string]interface{})
		if !ok {
			continue
		}

		// TODO Implement a proper merge
		if _, ok := item["containerResources"]; ok {
			item["containerResources"] = containerResources
			return
		}
	}

	*configuration = append(*configuration, map[string]interface{}{"containerResources": containerResources})
}

type DeployConfigurationOptions struct {
	Mode                   string
	Interval               time.Duration
	ContainerMax           map[string]string
	ContainerMin           map[string]string
	MaxRecommendationRatio map[string]string
	Clusters               []string
}

func (opts *DeployConfigurationOptions) Apply(deployConfiguration **applications.DeployConfiguration) {
	lazyDeployConfig := func() *applications.DeployConfiguration {
		if *deployConfiguration == nil {
			*deployConfiguration = &applications.DeployConfiguration{}
		}
		return *deployConfiguration
	}

	if opts.Mode != "" {
		lazyDeployConfig().Mode = opts.Mode
	}

	if opts.Interval > 0 {
		lazyDeployConfig().Interval = api.Duration(opts.Interval)
	}

	containerLimit := applications.LimitRangeItem{Type: "Container"}
	if len(opts.ContainerMax) > 0 {
		containerLimit.Max = &applications.ResourceList{}
		for k, v := range opts.ContainerMax {
			str := api.FromString(v)
			switch strings.ToLower(k) {
			case "cpu":
				containerLimit.Max.CPU = &str
			case "memory":
				containerLimit.Max.Memory = &str
			}
		}
	}
	if len(opts.ContainerMin) > 0 {
		containerLimit.Min = &applications.ResourceList{}
		for k, v := range opts.ContainerMin {
			str := api.FromString(v)
			switch strings.ToLower(k) {
			case "cpu":
				containerLimit.Min.CPU = &str
			case "memory":
				containerLimit.Min.Memory = &str
			}
		}
	}
	if containerLimit.Max != nil || containerLimit.Min != nil {
		lazyDeployConfig().Limits = append(lazyDeployConfig().Limits, containerLimit)
	}

	if len(opts.MaxRecommendationRatio) > 0 {
		ratios := &applications.ResourceList{}
		for k, v := range opts.MaxRecommendationRatio {
			str := api.FromString(v)
			switch strings.ToLower(k) {
			case "cpu":
				ratios.CPU = &str
			case "memory":
				ratios.Memory = &str
			}
		}
		lazyDeployConfig().MaxRecommendationRatio = ratios
	}

	if len(opts.Clusters) > 0 {
		lazyDeployConfig().Clusters = opts.Clusters
	}
}

func (opts *ContainerResourcesOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.Selector, "container-resource-selector", "",
		"`selector` for application resources which should have container resource optimization applied")
	cmd.Flags().DurationVar(&opts.Interval, "container-resource-interval", 0,
		"amount of `time` between container resource recommendation computations")
	cmd.Flags().StringToStringVar(&opts.TargetUtilization, "container-resource-target-utilization", nil,
		"container resource target utilization as `resource=quantity`")
	cmd.Flags().StringToStringVar(&opts.Tolerance, "container-resource-tolerance", nil,
		"container resource tolerance as `resource=low|medium|high`")
}

func (opts *DeployConfigurationOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.Mode, "deploy-mode", "",
		"deployment `mode`; one of: disabled|auto|manual")
	cmd.Flags().DurationVar(&opts.Interval, "deploy-interval", 0,
		"desired amount of `time` between deployments")
	cmd.Flags().StringToStringVar(&opts.ContainerMax, "deploy-container-max", nil,
		"per-container resource max limits as `resource=quantity`")
	cmd.Flags().StringToStringVar(&opts.ContainerMin, "deploy-container-min", nil,
		"per-container resource min limits as `resource=quantity`")
	cmd.Flags().StringToStringVar(&opts.MaxRecommendationRatio, "deploy-max-ratio", nil,
		"limit the recommended/current value ratio as `resource=ratio`")
	cmd.Flags().StringArrayVar(&opts.Clusters, "deploy-cluster", nil,
		"cluster `name` used for recommendations")
}
