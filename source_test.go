package confi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jfk9w-go/confi"
)

func TestSource_GetValues(t *testing.T) {
	tests := []struct {
		name     string
		source   confi.Source
		expected map[string]any
	}{
		{
			name: "basic properties",
			source: confi.PropertySource{
				{Path: []string{"map", "key1"}, Value: "value1"},
				{Path: []string{"map", "key2"}, Value: "value2"},
				{Path: []string{"map", "key3"}, Value: "value3"},
				{Path: []string{"map", "key3", "key1"}, Value: "value31"},
				{Path: []string{"map"}, Value: "map_value"},
			},
			expected: map[string]any{
				"map": map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": map[string]any{
						"key1": "value31",
					},
				},
			},
		},
		{
			name: "basic properties input",
			source: confi.InputSource{
				Input: confi.Reader{
					R: bytes.NewReader([]byte("map.key1=value1\n#map.key1=value22\nmap.key2=value2\nmap.key3=value3\nmap.key3.key1=value31\nmap=map_value\n")),
				},
				Format: "properties",
			},
			expected: map[string]any{
				"map": map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": map[string]any{
						"key1": "value31",
					},
				},
			},
		},
		{
			name: "basic yaml input",
			source: confi.InputSource{
				Input: confi.Reader{
					R: bytes.NewReader([]byte(`{map: {key1: value1, key2: value2, key3: {key1: value31}}}`)),
				},
				Format: "yaml",
			},
			expected: map[string]any{
				"map": map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": map[string]any{
						"key1": "value31",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := tt.source.GetValues(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, values)
		})
	}
}
