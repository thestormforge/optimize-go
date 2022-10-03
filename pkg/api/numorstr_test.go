package api

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromValue(t *testing.T) {
	cases := []struct {
		desc     string
		value    string
		expected NumberOrString
	}{
		{
			desc:     "string",
			value:    "foobar",
			expected: FromString("foobar"),
		},
		{
			desc:     "integer",
			value:    "1",
			expected: FromInt64(1),
		},
		{
			desc:     "float",
			value:    "1.1",
			expected: FromFloat64(1.1),
		},
		{
			desc:     "number",
			value:    "1",
			expected: FromNumber("1"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, FromValue(tc.value))
		})
	}
}

func TestNumberOrString_String(t *testing.T) {
	cases := []struct {
		desc     string
		value    NumberOrString
		expected string
	}{
		{
			desc:     "string",
			value:    FromString("foobar"),
			expected: "foobar",
		},
		{
			desc:     "integer",
			value:    FromInt64(1),
			expected: "1",
		},
		{
			desc:     "float",
			value:    FromFloat64(1.1),
			expected: "1.1",
		},
		{
			desc:     "number",
			value:    FromNumber("1"),
			expected: "1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}

	// Only Stringer is supported on nil receivers
	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, "null", (*NumberOrString)(nil).String())
	})
}

func TestNumberOrString_Int64Value(t *testing.T) {
	cases := []struct {
		desc     string
		value    NumberOrString
		expected int64
	}{
		{
			desc:     "string",
			value:    FromString("foobar"),
			expected: 0,
		},
		{
			desc:     "integer",
			value:    FromInt64(1),
			expected: 1,
		},
		{
			desc:     "float",
			value:    FromFloat64(1.1),
			expected: 0,
		},
		{
			desc:     "number",
			value:    FromNumber("1"),
			expected: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.Int64Value())
		})
	}

	// Only Stringer is supported on nil receivers
	t.Run("nil", func(t *testing.T) {
		assert.Panics(t, func() { _ = (*NumberOrString)(nil).Int64Value() })
	})
}

func TestNumberOrString_Float64Value(t *testing.T) {
	cases := []struct {
		desc     string
		value    NumberOrString
		expected float64
	}{
		{
			desc:     "string",
			value:    FromString("foobar"),
			expected: 0.0,
		},
		{
			desc:     "integer",
			value:    FromInt64(1),
			expected: 1.0,
		},
		{
			desc:     "float",
			value:    FromFloat64(1.1),
			expected: 1.1,
		},
		{
			desc:     "number",
			value:    FromNumber("1"),
			expected: 1.0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.Float64Value())
		})
	}

	// Only Stringer is supported on nil receivers
	t.Run("nil", func(t *testing.T) {
		assert.Panics(t, func() { _ = (*NumberOrString)(nil).Float64Value() })
	})
}

func TestNumberOrString_MarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		value    NumberOrString
		expected []byte
	}{
		{
			desc:     "string",
			value:    FromString("foobar"),
			expected: []byte(`"foobar"`),
		},
		{
			desc:     "integer",
			value:    FromInt64(1),
			expected: []byte(`1`),
		},
		{
			desc:     "float",
			value:    FromFloat64(1.1),
			expected: []byte(`1.1`),
		},
		{
			desc:     "number",
			value:    FromNumber("1"),
			expected: []byte(`1`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := json.Marshal(tc.value)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestNumberOrString_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		data     []byte
		expected NumberOrString
	}{
		{
			desc:     "string",
			data:     []byte(`"foobar"`),
			expected: FromString("foobar"),
		},
		{
			desc:     "integer",
			data:     []byte(`1`),
			expected: FromInt64(1),
		},
		{
			desc:     "float",
			data:     []byte(`1.1`),
			expected: FromFloat64(1.1),
		},
		{
			desc:     "number",
			data:     []byte(`1`),
			expected: FromNumber("1"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var actual NumberOrString
			if err := json.Unmarshal(tc.data, &actual); assert.NoError(t, err) {
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestNumberOrString_Quantity(t *testing.T) {
	cases := []struct {
		value    string
		expected float64
	}{
		{value: "1Ki", expected: 1024},
		{value: "1Mi", expected: 1048576},
		{value: "1Gi", expected: 1073741824},
		{value: "1Ti", expected: 1099511627776},
		{value: "1Pi", expected: 1125899906842624},
		{value: "1Ei", expected: 1152921504606846976},
		{value: "1n", expected: 1e-9},
		{value: "1u", expected: 1e-6},
		{value: "1m", expected: 0.001},
		{value: "1", expected: 1},
		{value: "1e1", expected: 10},
		{value: "1k", expected: 1000},
		{value: "1M", expected: 1000000},
		{value: "1G", expected: 1000000000},
		{value: "1T", expected: 1000000000000},
		{value: "1P", expected: 1000000000000000},
		{value: "1E", expected: 1000000000000000000},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			q := FromString(tc.value)
			assert.Equal(t, new(big.Float).SetFloat64(tc.expected), q.Quantity().SetPrec(53))
		})
	}

	t.Run("string", func(t *testing.T) {
		q := FromString("foobar")
		assert.Nil(t, q.Quantity())
	})
	t.Run("integer", func(t *testing.T) {
		q := FromInt64(1)
		assert.Equal(t, new(big.Float).SetInt64(1), q.Quantity())
	})
	t.Run("float", func(t *testing.T) {
		q := FromFloat64(1.1)
		assert.Equal(t, new(big.Float).SetFloat64(1.1), q.Quantity())
	})
}
