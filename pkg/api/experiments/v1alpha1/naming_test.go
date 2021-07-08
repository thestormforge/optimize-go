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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitTrialName(t *testing.T) {
	cases := []struct {
		name           string
		experimentName string
		trialNumber    int64
	}{
		{
			name:           "test-001",
			experimentName: "test",
			trialNumber:    1,
		},
		{
			name:           "dash-name-001",
			experimentName: "dash-name",
			trialNumber:    1,
		},
		{
			name:           "notoctal-010",
			experimentName: "notoctal",
			trialNumber:    10,
		},
		{
			name:           "morethenthreedigits-1000",
			experimentName: "morethenthreedigits",
			trialNumber:    1000,
		},
		{
			name:           "no-number",
			experimentName: "no-number",
			trialNumber:    -1,
		},
		{
			name:           "nodash",
			experimentName: "nodash",
			trialNumber:    -1,
		},
		{
			name:           "slash-name/1",
			experimentName: "slash-name",
			trialNumber:    1,
		},
		{
			name:           "yolo-2/",
			experimentName: "yolo-2",
			trialNumber:    -1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actualExperimentName, actualTrialNumber := SplitTrialName(c.name)
			assert.Equal(t, c.experimentName, actualExperimentName.String())
			assert.Equal(t, c.trialNumber, actualTrialNumber)
		})
	}
}
