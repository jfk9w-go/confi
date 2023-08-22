package confi_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jfk9w-go/confi"
)

type patternedValue string

func (patternedValue) SchemaPattern() string { return `\d+` }

type formattedValue string

func (formattedValue) SchemaFormat() string { return "duration" }

func TestGenerateSchema(t *testing.T) {
	type InnerObj struct {
		String string `yaml:"string,omitempty" default:"default_value"`
	}

	tests := []struct {
		name     string
		value    any
		expected confi.Schema
	}{
		{
			name: "primitives",
			value: struct {
				String     string         `yaml:"string" default:"value1" enum:"value1,value2,value3" minlen:"6" maxlen:"6"`
				Float      float64        `yaml:"float,omitempty" default:"15" enum:"14,15,16" min:"14" max:"16"`
				Integer    *int           `yaml:"integer,omitempty" default:"10" enum:"10,11,12" xmin:"9" xmax:"13"`
				Bool       bool           `yaml:"bool,omitempty" doc:"Boolean field" examples:"true,false" default:"true"`
				Time       time.Time      `yaml:"time,omitempty"`
				Duration   time.Duration  `yaml:"duration,omitempty"`
				Patterned  patternedValue `yaml:"patterned,omitempty"`
				Formatted  formattedValue `yaml:"formatted,omitempty"`
				unexported string
			}{},
			expected: confi.Schema{
				Type:                 "object",
				Required:             []string{"string"},
				AdditionalProperties: false,
				Properties: map[string]confi.Schema{
					"string": {
						Type:      "string",
						Default:   "value1",
						Enum:      []string{"value1", "value2", "value3"},
						MinLength: 6,
						MaxLength: pointer.To(uint64(6)),
					},
					"float": {
						Type:    "number",
						Default: float64(15),
						Enum:    []float64{14, 15, 16},
						Minimum: float64(14),
						Maximum: float64(16),
					},
					"integer": {
						Type:    "integer",
						Default: pointer.To(10),
						Enum: []*int{
							pointer.To(10),
							pointer.To(11),
							pointer.To(12),
						},
						ExclusiveMinimum: pointer.To(9),
						ExclusiveMaximum: pointer.To(13),
					},
					"bool": {
						Type:        "boolean",
						Description: "Boolean field",
						Examples:    []bool{true, false},
						Default:     true,
					},
					"time": {
						Type:   "string",
						Format: "date-time",
					},
					"duration": {
						Type:    "string",
						Pattern: `(\d+h)?(\d+m)?(\d+s)?(\d+ms)?(\d+µs)?(\d+ns)?`,
					},
					"patterned": {
						Type:    "string",
						Pattern: patternedValue("").SchemaPattern(),
					},
					"formatted": {
						Type:   "string",
						Format: formattedValue("").SchemaFormat(),
					},
				},
			},
		},
		{
			name: "complex",
			value: new(struct {
				InnerObj `yaml:",inline"`
				Inner    InnerObj             `yaml:"inner" default:"{string: aaa}" enum:"{string: aaa}, {string: bbb}"`
				InnerPtr *InnerObj            `yaml:"innerPtr,omitempty" default:"{string: bbb}" examples:"{string: bbb}, {string: ccc}"`
				Slice    []*InnerObj          `yaml:"slice,omitempty" default:"[{string: ccc}]" enum:"{string: ccc}, {string: ddd}" minsize:"1" maxsize:"3"`
				Array    [1]InnerObj          `yaml:"array,omitempty" default:"[{string: ddd}]" examples:"{string: ddd}, {string: eee}" unique:"true"`
				Map      map[string]*InnerObj `yaml:"map,omitempty" default:"{eee: {string: ggg}}" enum:"{string: eee}, {string: ggg}" minprops:"1" maxprops:"3"`
			}),
			expected: confi.Schema{
				Type:                 "object",
				Required:             []string{"inner"},
				AdditionalProperties: false,
				Properties: map[string]confi.Schema{
					"string": {Type: "string", Default: "default_value"},
					"inner": {
						Type:                 "object",
						AdditionalProperties: false,
						Properties: map[string]confi.Schema{
							"string": {Type: "string", Default: "default_value"},
						},
						Default: InnerObj{String: "aaa"},
						Enum:    []InnerObj{{String: "aaa"}, {String: "bbb"}},
					},
					"innerPtr": {
						Type:                 "object",
						AdditionalProperties: false,
						Properties: map[string]confi.Schema{
							"string": {Type: "string", Default: "default_value"},
						},
						Default:  &InnerObj{String: "bbb"},
						Examples: []*InnerObj{{String: "bbb"}, {String: "ccc"}},
					},
					"slice": {
						Type:     "array",
						MinItems: 1,
						MaxItems: pointer.To(uint64(3)),
						Default:  []*InnerObj{{String: "ccc"}},
						Items: &confi.Schema{
							Type:                 "object",
							AdditionalProperties: false,
							Properties: map[string]confi.Schema{
								"string": {Type: "string", Default: "default_value"},
							},
							Enum: []*InnerObj{{String: "ccc"}, {String: "ddd"}},
						},
					},
					"array": {
						Type:        "array",
						MinItems:    1,
						MaxItems:    pointer.To(uint64(1)),
						UniqueItems: true,
						Default:     [1]InnerObj{{String: "ddd"}},
						Items: &confi.Schema{
							Type:                 "object",
							AdditionalProperties: false,
							Properties: map[string]confi.Schema{
								"string": {Type: "string", Default: "default_value"},
							},
							Examples: []InnerObj{{String: "ddd"}, {String: "eee"}},
						},
					},
					"map": {
						Type:          "object",
						MinProperties: 1,
						MaxProperties: pointer.To(uint64(3)),
						Default:       map[string]*InnerObj{"eee": {String: "ggg"}},
						AdditionalProperties: &confi.Schema{
							Type:                 "object",
							AdditionalProperties: false,
							Properties: map[string]confi.Schema{
								"string": {Type: "string", Default: "default_value"},
							},
							Enum: []*InnerObj{{String: "eee"}, {String: "ggg"}},
						},
					},
				},
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

func TestSchema_ApplyDefaults(t *testing.T) {
	type InnerObj struct {
		InnerString string `yaml:"innerString" default:"default_inner_string"`
	}

	type Value struct {
		EmbeddedObj          InnerObj             `yaml:",inline"`
		String               string               `yaml:"string" default:"default_string"`
		SetString            string               `yaml:"setString" default:"another_default_string"`
		Integer              *int                 `yaml:"integer" default:"20"`
		SetInteger           *int                 `yaml:"setInteger" default:"25"`
		Float                float64              `yaml:"float" default:"30"`
		SetFloat             float64              `yaml:"setFloat" default:"35"`
		Bool                 *bool                `yaml:"bool" default:"true"`
		SetBool              *bool                `yaml:"setBool" default:"true"`
		InnerObj             InnerObj             `yaml:"innerObj" default:"{innerString: another_default_inner_string}"`
		InnerObjSlice        []InnerObj           `yaml:"innerObjSlice" default:"[{innerString: aaa}, {innerString: bbb}]"`
		HalfSetInnerObjSlice []InnerObj           `yaml:"halfSetInnerObjSlice"`
		InnerObjPtr          *InnerObj            `yaml:"innerObjPtr"`
		FilledInnerObjPtr    *InnerObj            `yaml:"filledInnerObjPtr" default:"{innerString: default_inner_string}"`
		Map                  map[string]InnerObj  `yaml:"map" default:"{aaa: {innerString: bbb}}"`
		HalfSetMap           map[string]*InnerObj `yaml:"halfSetMap"`
	}

	value := Value{
		SetString:  "123",
		SetInteger: pointer.To(1),
		SetFloat:   2,
		SetBool:    pointer.To(false),
		HalfSetInnerObjSlice: []InnerObj{
			{InnerString: "aaa"},
			{},
		},
		HalfSetMap: map[string]*InnerObj{
			"aaa": {InnerString: "bbb"},
			"ccc": {},
			"ddd": nil, // this will be ignored
		},
	}

	schema, err := confi.GenerateSchema(new(Value))
	require.NoError(t, err)
	require.NoError(t, schema.ApplyDefaults(&value))

	assert.Equal(t, Value{
		EmbeddedObj:          InnerObj{InnerString: "default_inner_string"},
		String:               "default_string",
		SetString:            "123",
		Integer:              pointer.To(20),
		SetInteger:           pointer.To(1),
		Float:                30,
		SetFloat:             2,
		Bool:                 pointer.To(true),
		SetBool:              pointer.To(false),
		InnerObj:             InnerObj{InnerString: "another_default_inner_string"},
		InnerObjSlice:        []InnerObj{{InnerString: "aaa"}, {InnerString: "bbb"}},
		HalfSetInnerObjSlice: []InnerObj{{InnerString: "aaa"}, {InnerString: "default_inner_string"}},
		FilledInnerObjPtr:    pointer.To(InnerObj{InnerString: "default_inner_string"}),
		Map:                  map[string]InnerObj{"aaa": {InnerString: "bbb"}},
		HalfSetMap:           map[string]*InnerObj{"aaa": {InnerString: "bbb"}, "ccc": {InnerString: "default_inner_string"}, "ddd": nil},
	}, value)
}
