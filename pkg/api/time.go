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

package api

import (
	"encoding/json"
	"time"
)

// Duration is an alternate duration type that marshals as a JSON string.
type Duration time.Duration

// UnmarshalJSON handles the string formatted duration.
func (d *Duration) UnmarshalJSON(bytes []byte) error {
	var str string
	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}
	td, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*d = Duration(td)
	return nil
}

// MarshalJSON produces a string formatted duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// String returns the string representation of the duration.
func (d Duration) String() string {
	return time.Duration(d).String()
}
