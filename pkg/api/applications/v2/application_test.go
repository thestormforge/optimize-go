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

package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thestormforge/optimize-go/pkg/api"
)

func TestApplicationList_UnmarshalJSON(t *testing.T) {
	data := []byte(`
{
  "_metadata": {
    "Link": [ "</should/be/ignored>; rel=prev", "</comes/from/headers>; rel=next" ]
  },
  "applications": [
    {
      "_metadata": {
        "Link": "</test1>; rel=self",
        "Title": "Test1"
      },
      "name": "test1",
      "title": "Test1"
    },
    {
      "_metadata": {
        "Link": "</test2>; rel=self",
        "Title": "Test2"
      },
      "name": "test-two",
      "title": "Test2"
    }
  ]
}
`)

	// The top-level metadata generally comes in via HTTP headers prior to unmarshalling,
	// we want to verify it is not overwritten by the entity body content
	h := http.Header{
		"Link": []string{
			"</test?limit=5>; rel=prev",
			"</test?offset=10&limit=5>; rel=next",
		},
	}
	l := ApplicationList{
		Metadata: api.Metadata(h),
	}

	if err := json.Unmarshal(data, &l); assert.NoError(t, err) {
		assert.Equal(t, "/test?limit=5", l.Link(api.RelationPrev))
		assert.Equal(t, "/test?offset=10&limit=5", l.Link(api.RelationNext))
		assert.Len(t, l.Applications, 2)

		assert.Equal(t, "/test1", l.Applications[0].Link(api.RelationSelf))
		assert.Equal(t, "test1", l.Applications[0].Name.String())
		assert.Equal(t, "Test1", l.Applications[0].DisplayName)
		assert.Equal(t, "Test1", l.Applications[0].Title())

		assert.Equal(t, "/test2", l.Applications[1].Link(api.RelationSelf))
		assert.Equal(t, "test-two", l.Applications[1].Name.String())
		assert.Equal(t, "Test2", l.Applications[1].DisplayName)
		assert.Equal(t, "Test2", l.Applications[1].Title())
	}
}
