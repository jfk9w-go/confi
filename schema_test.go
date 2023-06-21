package confi

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_makeSchema(t *testing.T) {
	type innerStruct struct {
		String string `yaml:"string"`
	}

	var testValue struct {
		String  string  `yaml:"string" doc:"Test string"`
		Float   float64 `yaml:"float,omitempty"`
		Integer int64   `yaml:"integer,omitempty"`
		Slice   []struct {
			Strings []string `yaml:"strings" enum:"test1,test2"`
		} `yaml:"slice,omitempty"`
		Array [2]struct {
			Float float64 `yaml:"float" examples:"1,2,3"`
		} `yaml:"array"`
		Map map[string]struct {
			Integer int64 `yaml:"integer" default:"10"`
		} `yaml:"map,omitempty"`
		Struct   innerStruct `yaml:"struct,omitempty" default:"{string: 123}"`
		Embedded struct {
			EmbeddedString string `yaml:"embeddedString" minlen:"10" maxlen:"20"`
		} `yaml:"-,inline"`
	}

	testSchema, err := makeSchema(testValue, "")
	require.NoError(t, err)
	assert.Equal(t, &Schema{
		Type: "object",
		Properties: map[string]Schema{
			"string": {
				Type:        "string",
				Description: "Test string",
			},
			"float": {
				Type: "number",
			},
			"integer": {
				Type: "integer",
			},
			"slice": {
				Type: "array",
				Items: &Schema{
					Type: "object",
					Properties: map[string]Schema{
						"strings": {
							Type: "array",
							Items: &Schema{
								Type: "string",
								Enum: []string{"test1", "test2"},
							},
						},
					},
					AdditionalProperties: false,
					Required:             []string{"strings"},
				},
			},
			"array": {
				Type:     "array",
				MinItems: pointer.To[uint64](2),
				MaxItems: pointer.To[uint64](2),
				Items: &Schema{
					Type: "object",
					Properties: map[string]Schema{
						"float": {
							Type:     "number",
							Examples: []float64{1, 2, 3},
						},
					},
					AdditionalProperties: false,
					Required:             []string{"float"},
				},
			},
			"map": {
				Type: "object",
				AdditionalProperties: &Schema{
					Type: "object",
					Properties: map[string]Schema{
						"integer": {
							Type:    "integer",
							Default: int64(10),
						},
					},
					AdditionalProperties: false,
					Required:             []string{"integer"},
				},
			},
			"struct": {
				Type: "object",
				Properties: map[string]Schema{
					"string": {
						Type: "string",
					},
				},
				AdditionalProperties: false,
				Default: innerStruct{
					String: "123",
				},
				Required: []string{"string"},
			},
			"embeddedString": {
				Type:      "string",
				MinLength: pointer.To[uint64](10),
				MaxLength: pointer.To[uint64](20),
			},
		},
		AdditionalProperties: false,
		Required:             []string{"string", "array", "embeddedString"},
	}, testSchema)
}
