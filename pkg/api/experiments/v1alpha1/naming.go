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

package v1alpha1

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// ExperimentName represents a name token used to identify an experiment.
type ExperimentName string

func (n ExperimentName) String() string { return string(n) }

func extractExperimentName(md api.Metadata) ExperimentName {
	l := md.Link(api.RelationSelf)
	if l == "" {
		l = md.Location()
	}

	u, err := url.Parse(l)
	if err != nil || u.Path == "" {
		return ""
	}

	n := ExperimentName(path.Base(u.Path))
	if n == "/" {
		return ""
	}

	return n
}

// JoinTrialName combines an experiment and a trial.
func JoinTrialName(e *Experiment, number int64) string {
	if e != nil && e.Name != "" {
		return fmt.Sprintf("%s-%03d", e.Name, number)
	}

	return strconv.FormatInt(number, 10)
}

// SplitTrialName provides a consistent experience when trying to split a "trial name" into an experiment
// name and a trial number. When the provided name does not contain a number, the resulting number will
// be less than zero.
func SplitTrialName(name string) (ExperimentName, int64) {
	// Names with slashes are always split (since the slash can't be in the name)
	p := strings.LastIndex(name, "/")
	if p >= 0 {
		if num, err := strconv.ParseInt(name[p+1:], 10, 64); err == nil {
			return ExperimentName(name[0:p]), num
		}
		return ExperimentName(name[0:p]), -1
	}

	// The only other allowable separator is the hyphen
	p = strings.LastIndex(name, "-")
	if p >= 0 {
		// Strip off a valid number after the "-". If your experiment name has a "-<NUM>" suffix, use a slash
		if num, err := strconv.ParseInt(name[p+1:], 10, 64); err == nil {
			return ExperimentName(name[0:p]), num
		}
	}

	return ExperimentName(name), -1
}
