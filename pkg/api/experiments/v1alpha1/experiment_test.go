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

package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thestormforge/optimize-go/pkg/api"
)

func TestExperimentList_UnmarshalJSON(t *testing.T) {
	data := []byte(`
{
  "experiments": [
    {
      "_metadata": {
        "Link": "</test1>; rel=self"
      },
      "displayName": "Test1"
    },
    {
      "_metadata": {
        "Link": [
          "</test2>; rel=self",
          "</test2?offset=5>; rel=next"
        ]
      },
      "displayName": "Test2"
    }
  ]
}
`)

	l := ExperimentList{}
	err := json.Unmarshal(data, &l)
	if assert.NoError(t, err) {
		assert.Len(t, l.Experiments, 2)

		assert.Equal(t, "/test1", l.Experiments[0].Link(api.RelationSelf))
		assert.Equal(t, "Test1", l.Experiments[0].DisplayName)

		assert.Equal(t, "/test2", l.Experiments[1].Link(api.RelationSelf))
		assert.Equal(t, "/test2?offset=5", l.Experiments[1].Link(api.RelationNext))
		assert.Equal(t, "Test2", l.Experiments[1].DisplayName)
	}
}
