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

package api

import (
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"strconv"
)

// NumberOrString is value that can a JSON number or string.
type NumberOrString struct {
	IsString bool
	NumVal   json.Number
	StrVal   string
}

// FromInt64 returns the supplied value as a NumberOrString
func FromInt64(val int64) NumberOrString {
	return NumberOrString{NumVal: json.Number(strconv.FormatInt(val, 10))}
}

// FromFloat64 returns the supplied value as a NumberOrString
func FromFloat64(val float64) NumberOrString {
	s := strconv.FormatFloat(val, 'f', -1, 64)
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return NumberOrString{StrVal: s, IsString: true}
	}
	return NumberOrString{NumVal: json.Number(s)}
}

// FromNumber returns the supplied value as a NumberOrString
func FromNumber(val json.Number) NumberOrString {
	return NumberOrString{NumVal: val}
}

// FromString returns the supplied value as a NumberOrString
func FromString(val string) NumberOrString {
	return NumberOrString{StrVal: val, IsString: true}
}

// FromValue returns the supplied value as a NumberOrString based
// on an attempt to parse the supplied value as an int or float.
func FromValue(val string) NumberOrString {
	if v, err := strconv.ParseInt(val, 10, 64); err == nil {
		return FromInt64(v)
	}
	if v, err := strconv.ParseFloat(val, 64); err == nil {
		return FromFloat64(v)
	}
	return FromString(val)
}

// String coerces the value to a string.
func (s *NumberOrString) String() string {
	if s == nil {
		return "null"
	}
	if s.IsString {
		return s.StrVal
	}
	return s.NumVal.String()
}

// Int64Value coerces the value to an int64.
func (s *NumberOrString) Int64Value() int64 {
	if s.IsString {
		v, _ := strconv.ParseInt(s.StrVal, 10, 64)
		return v
	}
	v, _ := s.NumVal.Int64()
	return v
}

// Float64Value coerces the value to a float64.
func (s *NumberOrString) Float64Value() float64 {
	if s.IsString {
		v, _ := strconv.ParseFloat(s.StrVal, 64)
		return v
	}
	v, _ := s.NumVal.Float64()
	return v
}

// MarshalJSON writes the value with the appropriate type.
func (s NumberOrString) MarshalJSON() ([]byte, error) {
	if s.IsString {
		return json.Marshal(s.StrVal)
	}
	return json.Marshal(s.NumVal)
}

// UnmarshalJSON reads the value from either a string or number.
func (s *NumberOrString) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		s.IsString = true
		return json.Unmarshal(b, &s.StrVal)
	}
	return json.Unmarshal(b, &s.NumVal)
}

// quantityRegExp comes from Kubernetes.
var quantityRegExp = regexp.MustCompile(`^([-+]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`)

// Quantity attempts to return a big float using the same logic as a Kubernetes
// quantity. If parsing fails, this will return nil.
func (s *NumberOrString) Quantity() *big.Float {
	if !s.IsString {
		if v, err := s.NumVal.Int64(); err == nil {
			return new(big.Float).SetInt64(v)
		}
		if v, err := s.NumVal.Float64(); err == nil {
			return new(big.Float).SetFloat64(v)
		}
	} else if parts := quantityRegExp.FindStringSubmatch(s.StrVal); len(parts) == 3 {
		strVal, op := parts[1], 1.0
		switch parts[2] {
		case "":
			op = 1.0
		case "Ki":
			op = math.Pow(2, 10)
		case "Mi":
			op = math.Pow(2, 20)
		case "Gi":
			op = math.Pow(2, 30)
		case "Ti":
			op = math.Pow(2, 40)
		case "Pi":
			op = math.Pow(2, 50)
		case "Ei":
			op = math.Pow(2, 60)
		case "n":
			op = math.Pow10(-9)
		case "u":
			op = math.Pow10(-6)
		case "m":
			op = math.Pow10(-3)
		case "k":
			op = math.Pow10(3)
		case "M":
			op = math.Pow10(6)
		case "G":
			op = math.Pow10(9)
		case "T":
			op = math.Pow10(12)
		case "P":
			op = math.Pow10(15)
		case "E":
			op = math.Pow10(18)
		default:
			strVal += parts[2]
		}

		if v, ok := new(big.Float).SetString(strVal); ok {
			return v.Mul(v, big.NewFloat(op))
		}
	} else if x, err := strconv.ParseFloat(s.StrVal, 64); err == nil { // +Inf, etc.
		// We use `ParseFloat` instead of `s.Float64Value()` so we can explicitly check the error
		return big.NewFloat(x)
	}

	return nil
}
