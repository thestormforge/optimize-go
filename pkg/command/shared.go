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

package command

import (
	"strings"
)

// Config represents the configuration necessary to run a command.
type Config interface {
	// Address returns the base address for the API endpoints.
	Address() string
}

// parseLabelSelector returns a map of simple equality based label selectors.
func parseLabelSelector(s string) map[string]string {
	if s == "" {
		return nil
	}

	// Note: we used to use the Kubernetes code to implement matching on the client side,
	// this code implements a significantly reduced set of simple AND'd equals expressions
	selector := make(map[string]string)
	for _, sel := range strings.Split(s, ",") {
		pair := strings.SplitN(sel, "=", 2)
		var value string
		if len(pair) > 1 {
			value = strings.TrimSpace(pair[1])
		}
		selector[strings.TrimSpace(pair[0])] = value
	}

	return selector
}

// argsToNamesAndLabels returns a list of names and label mappings from an argument stream.
func argsToNamesAndLabels(args []string) ([]string, map[string]string) {
	names := make([]string, 0, len(args))
	labels := make(map[string]string, len(args))

	for _, arg := range args {
		if p := strings.SplitN(arg, "=", 2); len(p) == 2 {
			labels[p[0]] = p[1]
		} else if p := strings.TrimSuffix(arg, "-"); p != arg {
			labels[p] = ""
		} else {
			names = append(names, arg)
		}
	}

	return names, labels
}
