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
	"unicode"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// QuantityLess returns true if 'a' is strictly less than 'b'.
func QuantityLess(a, b *api.NumberOrString) bool {
	af, bf := a.Quantity(), b.Quantity()

	if af == nil {
		return bf != nil
	} else if bf == nil {
		return af != nil
	}

	return af.Cmp(bf) < 0
}

// LikelyInvalid returns true if the value is likely to be invalid.
func LikelyInvalid(resourceName string, val *api.NumberOrString, resourceAsPercentage bool) error {
	if resourceAsPercentage {
		// Already verified Quantity is not negative, so only checking max
		max := api.FromString("100")

		if QuantityLess(&max, val) {
			return fmt.Errorf("%s must be at most %s", &max, val)
		}
		return nil
	}

	switch resourceName {
	case "cpu":
		// There is a 1 millicore granularity requirement, you can't specify less than that
		minCPU := api.FromString("1m")

		if QuantityLess(val, &minCPU) {
			return fmt.Errorf("%s must be at least %s", val, &minCPU)
		}
		return nil

	case "memory":
		// While not a hard requirement, specifying less than a megabyte probably isn't going to work
		minMemory := api.FromString("1M")
		if val.IsString && strings.TrimFunc(strings.TrimLeft(val.StrVal, "-+"), unicode.IsDigit) != "" {
			minMemory = api.FromInt64(0)
		}

		if QuantityLess(val, &minMemory) {
			return fmt.Errorf("%s must be at least %s", val, &minMemory)
		}
		return nil

	default:
		return nil
	}
}
