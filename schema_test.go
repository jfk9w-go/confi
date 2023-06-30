package confi_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/jfk9w-go/confi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSchema(t *testing.T) {
	type InnerObj struct {
		String string `yaml:"string,omitempty" default:"inner_default_string"`
	}

	tests := []struct {
		name     string
		value    any
		expected confi.Schema
	}{
		{
			name: "basic",
			value: struct {
				String   *string        `yaml:"string,omitempty" doc:"Test string" default:"default_string_value"`
				Integer  int64          `yaml:"integer" enum:"10,11" min:"10" max:"11" default:"10"`
				Float    float64        `yaml:"float" examples:"10.5,16.5" xmin:"10" xmax:"20"`
				Bool     bool           `yaml:"bool"`
				Duration *time.Duration `yaml:"duration,omitempty"`
				Time     *time.Time     `yaml:"time,omitempty"`
			}{},
			expected: confi.Schema{
				Type:     "object",
				Required: []string{"integer", "float", "bool"},
				Properties: map[string]confi.Schema{
					"string": {
						Type:        "string",
						Description: "Test string",
						Default:     pointer.To("default_string_value"),
					},
					"integer": {
						Type:    "integer",
						Enum:    []int64{10, 11},
						Minimum: int64(10),
						Maximum: int64(11),
						Default: int64(10),
					},
					"float": {
						Type:             "number",
						Examples:         []float64{10.5, 16.5},
						ExclusiveMinimum: float64(10),
						ExclusiveMaximum: float64(20),
					},
					"bool": {
						Type: "boolean",
					},
					"duration": {
						Type:    "string",
						Pattern: `(\d+h)?(\d+m)?(\d+s)?(\d+ms)?(\d+µs)?(\d+ns)?`,
					},
					"time": {
						Type:   "string",
						Format: "date-time",
					},
				},
				AdditionalProperties: false,
			},
		},
		{
			name: "nested",
			value: struct {
				InnerObj    *InnerObj           `yaml:"innerObj,omitempty" default:"{string: test_string}"`
				InnerObjs   map[string]InnerObj `yaml:"innerObjs,omitempty" minprops:"1" maxprops:"5" default:"{aaa: {string: bbb}}"`
				EmbeddedObj InnerObj            `yaml:",inline"`
			}{},
			expected: confi.Schema{
				Type: "object",
				Properties: map[string]confi.Schema{
					"innerObj": {
						Type: "object",
						Properties: map[string]confi.Schema{
							"string": {
								Type:    "string",
								Default: "inner_default_string",
							},
						},
						AdditionalProperties: false,
						Default: &InnerObj{
							String: "test_string",
						},
					},
					"innerObjs": {
						Type: "object",
						AdditionalProperties: &confi.Schema{
							Type: "object",
							Properties: map[string]confi.Schema{
								"string": {
									Type:    "string",
									Default: "inner_default_string",
								},
							},
							AdditionalProperties: false,
						},
						MinProperties: 1,
						MaxProperties: pointer.To[uint64](5),
						Default: map[string]InnerObj{
							"aaa": {String: "bbb"},
						},
					},
					"string": {
						Type:    "string",
						Default: "inner_default_string",
					},
				},
				AdditionalProperties: false,
			},
		},
		{
			name: "arrays and slices",
			value: struct {
				Slice []string    `yaml:"slice,omitempty" enum:"aaa,bbb,ccc" minsize:"10" maxsize:"15" unique:"true"`
				Array [3]InnerObj `yaml:"array,omitempty" default:"[{string: aaa}, {string: bbb}, {string: ccc}]"`
			}{},
			expected: confi.Schema{
				Type: "object",
				Properties: map[string]confi.Schema{
					"slice": {
						Type: "array",
						Items: &confi.Schema{
							Type: "string",
							Enum: []string{"aaa", "bbb", "ccc"},
						},
						MinItems:    10,
						MaxItems:    pointer.To(uint64(15)),
						UniqueItems: true,
					},
					"array": {
						Type: "array",
						Items: &confi.Schema{
							Type: "object",
							Properties: map[string]confi.Schema{
								"string": {
									Type:    "string",
									Default: "inner_default_string",
								},
							},
							AdditionalProperties: false,
						},
						Default:  [3]InnerObj{{String: "aaa"}, {String: "bbb"}, {String: "ccc"}},
						MinItems: 3,
						MaxItems: pointer.To(uint64(3)),
					},
				},
				AdditionalProperties: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := confi.GenerateSchema(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, *schema)
		})
	}
}
