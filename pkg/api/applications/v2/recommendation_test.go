package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thestormforge/optimize-go/pkg/api"
)

func TestMergeConfigurations(t *testing.T) {
	cases := []struct {
		desc     string
		first    Configuration
		second   Configuration
		expected Configuration
	}{
		{
			desc: "empty",
		},
		{
			desc: "no second",
			first: Configuration{
				ContainerResources: &ContainerResources{
					Selector: "foo",
					Interval: 1,
					TargetUtilization: &ResourceList{
						CPU:    &api.NumberOrString{NumVal: "1"},
						Memory: &api.NumberOrString{NumVal: "2"},
					},
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Max: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "1"},
								Memory: &api.NumberOrString{NumVal: "2"},
							},
						},
					},
				},
			},
			expected: Configuration{
				ContainerResources: &ContainerResources{
					Selector: "foo",
					Interval: 1,
					TargetUtilization: &ResourceList{
						CPU:    &api.NumberOrString{NumVal: "1"},
						Memory: &api.NumberOrString{NumVal: "2"},
					},
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Max: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "1"},
								Memory: &api.NumberOrString{NumVal: "2"},
							},
						},
					},
				},
			},
		},
		{
			desc: "merge min-max bounds",
			first: Configuration{
				ContainerResources: &ContainerResources{
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Min: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "1"},
								Memory: &api.NumberOrString{NumVal: "2"},
							},
						},
					},
				},
			},
			second: Configuration{
				ContainerResources: &ContainerResources{
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Max: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "3"},
								Memory: &api.NumberOrString{NumVal: "4"},
							},
						},
					},
				},
			},
			expected: Configuration{
				ContainerResources: &ContainerResources{
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Min: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "1"},
								Memory: &api.NumberOrString{NumVal: "2"},
							},
							Max: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "3"},
								Memory: &api.NumberOrString{NumVal: "4"},
							},
						},
					},
				},
			},
		},
		{
			desc: "merge cpu-memory bounds",
			first: Configuration{
				ContainerResources: &ContainerResources{
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Min: &ResourceList{
								Memory: &api.NumberOrString{NumVal: "2"},
							},
							Max: &ResourceList{
								Memory: &api.NumberOrString{NumVal: "4"},
							},
						},
					},
				},
			},
			second: Configuration{
				ContainerResources: &ContainerResources{
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Min: &ResourceList{
								CPU: &api.NumberOrString{NumVal: "1"},
							},
							Max: &ResourceList{
								CPU: &api.NumberOrString{NumVal: "3"},
							},
						},
					},
				},
			},
			expected: Configuration{
				ContainerResources: &ContainerResources{
					Bounds: &Bounds{
						Requests: &BoundsRange{
							Min: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "1"},
								Memory: &api.NumberOrString{NumVal: "2"},
							},
							Max: &ResourceList{
								CPU:    &api.NumberOrString{NumVal: "3"},
								Memory: &api.NumberOrString{NumVal: "4"},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := MergeConfigurations(&tc.first, &tc.second)
			if assert.NoError(t, err) {
				assert.Equal(t, &tc.expected, actual)
			}
		})
	}
}
