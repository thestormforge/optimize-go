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
			assert.Equal(t, c.experimentName, actualExperimentName.Name())
			assert.Equal(t, c.trialNumber, actualTrialNumber)
		})
	}
}
