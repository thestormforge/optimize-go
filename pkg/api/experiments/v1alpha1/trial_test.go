package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thestormforge/optimize-go/pkg/api"
)

func TestTrialList_UnmarshalJSON(t *testing.T) {
	data := []byte(`
{
  "trials": [
    {
      "_metadata": {
        "Link": [
          "</labelTest1>; rel=https://stormforge.io/rel/labels"
        ]
      },
      "number": 1,
      "failureReason": "test",
      "labels": {
        "best": "true"
      }
    },
    {
      "_metadata": {
        "Link": [
          "</labelTest2>; rel=https://stormforge.io/rel/labels"
        ]
      },
      "number": 2,
      "failureReason": "test",
      "labels": {
        "manually_created": "true"
      }
    }
  ]
}
`)

	l := TrialList{}
	err := json.Unmarshal(data, &l)
	if assert.NoError(t, err) {
		assert.Len(t, l.Trials, 2)

		assert.Equal(t, "/labelTest1", l.Trials[0].Link(api.RelationLabels))
		assert.Equal(t, int64(1), l.Trials[0].Number)
		assert.Equal(t, "test", l.Trials[0].FailureReason)
		assert.Equal(t, "true", l.Trials[0].Labels["best"])

		assert.Equal(t, "/labelTest2", l.Trials[1].Link(api.RelationLabels))
		assert.Equal(t, int64(2), l.Trials[1].Number)
		assert.Equal(t, "test", l.Trials[1].FailureReason)
		assert.Equal(t, "true", l.Trials[1].Labels["manually_created"])
	}
}
