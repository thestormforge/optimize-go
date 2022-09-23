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
	"math"
	"math/big"
	"regexp"

	"github.com/thestormforge/optimize-go/pkg/api"
)

// quantityRegExp comes from Kubernetes, but we remove the leading "-" to only accept non-negative numbers.
var quantityRegExp = regexp.MustCompile(`^([+]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`)

// IsNonNegativeQuantity returns true if the supplied value represents a Kubernetes
// quantity with a value greater than or equal to 0.
func IsNonNegativeQuantity(num *api.NumberOrString) bool {
	if num.IsString {
		return quantityRegExp.MatchString(num.StrVal)
	}

	if v, err := num.NumVal.Int64(); err == nil {
		return v >= 0
	}

	if v, err := num.NumVal.Float64(); err == nil {
		return v >= 0
	}

	return false
}

// QuantityLess returns true if 'a' is strictly less than 'b'.
func QuantityLess(a, b *api.NumberOrString) bool {
	var af, bf *big.Float

	if a.IsString {
		af = guessValue(a.StrVal)
	} else if v, err := a.NumVal.Float64(); err == nil {
		af = big.NewFloat(v)
	}

	if b.IsString {
		bf = guessValue(b.StrVal)
	} else if v, err := b.NumVal.Float64(); err == nil {
		bf = big.NewFloat(v)
	}

	if a == nil {
		return b != nil
	} else if b == nil {
		return a != nil
	}

	return af.Cmp(bf) < 0
}

// guessValue attempts to approximate Kubernetes quantity parsing using a `big.Float`.
func guessValue(val string) *big.Float {
	parts := quantityRegExp.FindStringSubmatch(val)
	v, f := parts[1], 1.0
	switch parts[2] {
	case "":
		f = 1.0
	case "Ki":
		f = math.Pow(2, 10)
	case "Mi":
		f = math.Pow(2, 20)
	case "Gi":
		f = math.Pow(2, 30)
	case "Ti":
		f = math.Pow(2, 40)
	case "Pi":
		f = math.Pow(2, 50)
	case "Ei":
		f = math.Pow(2, 60)
	case "n":
		f = math.Pow10(-9)
	case "u":
		f = math.Pow10(-6)
	case "m":
		f = math.Pow10(-3)
	case "k":
		f = math.Pow10(3)
	case "M":
		f = math.Pow10(9)
	case "G":
		f = math.Pow10(9)
	case "T":
		f = math.Pow10(12)
	case "P":
		f = math.Pow10(15)
	case "E":
		f = math.Pow10(18)
	default:
		v += parts[2]
	}

	if result, _, _ := new(big.Float).Parse(v, 10); result != nil {
		return result.Mul(result, big.NewFloat(f))
	}

	return nil
}
