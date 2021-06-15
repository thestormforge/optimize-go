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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexQuery_SetLimit(t *testing.T) {
	q := IndexQuery{}

	q.SetLimit(11)
	q.SetLimit(97)
	assert.Equal(t, []string{"97"}, q[ParamLimit])

	q.SetLimit(0)
	assert.NotContains(t, q, ParamLimit)
}

func TestIndexQuery_SetLabelSelector(t *testing.T) {
	q := IndexQuery{}

	q.SetLabelSelector(map[string]string{
		"application": "my-app",
		"scenario":    "cyber-monday",
	})
	assert.Equal(t, []string{"application=my-app,scenario=cyber-monday"}, q[ParamLabelSelector])

	q.SetLabelSelector(map[string]string{
		"best": "true",
	})
	assert.Equal(t, []string{"application=my-app,scenario=cyber-monday", "best=true"}, q[ParamLabelSelector])
}
