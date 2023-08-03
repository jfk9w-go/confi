package confi_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jfk9w-go/confi"
)

type mockSource struct {
	format string
	data   string
}

type mockSourceProvider []mockSource

func (p mockSourceProvider) GetSources(ctx context.Context) ([]confi.Source, error) {
	sources := make([]confi.Source, len(p))
	for i, mock := range p {
		sources[i] = confi.InputSource{
			Input:  confi.Reader{R: bytes.NewReader([]byte(mock.data))},
			Format: mock.format,
		}
	}

	return sources, nil
}

func TestFromProvider(t *testing.T) {
	type A struct {
		AA string `yaml:"aa"`
	}

	type B struct {
		BB int `yaml:"bb"`
	}

	type C struct {
		CC string `yaml:"cc"`
		DD int    `yaml:"dd" default:"125"`
	}

	type Config struct {
		A A          `yaml:"a"`
		B B          `yaml:",inline"`
		C map[int]*C `yaml:"c"`
	}

	tests := []struct {
		name     string
		provider confi.SourceProvider
		expected Config
	}{
		{
			name: "basic",
			provider: mockSourceProvider{
				{"json", `{"a": {"aa": 123}, "c": {"456": {"cc": "hello"}}}`},
				{"yaml", `{a: {aa: 456}, bb: 789}`},
			},
			expected: Config{
				A: A{
					AA: "456",
				},
				B: B{
					BB: 789,
				},
				C: map[int]*C{
					456: {
						CC: "hello",
						DD: 125,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, _, err := confi.FromProvider[Config](context.Background(), tt.provider)
			if assert.NoError(t, err) {
				assert.Equal(t, tt.expected, *actual)
			}
		})
	}
}

func TestSpecifyType(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{
			name:     "basic string",
			value:    "sup",
			expected: "sup",
		},
		{
			name:     "string to bool",
			value:    "true",
			expected: true,
		},
		{
			name:     "string to uint",
			value:    "123",
			expected: uint64(123),
		},
		{
			name:     "string to int",
			value:    "-123",
			expected: int64(-123),
		},
		{
			name:     "string to float",
			value:    "123.456",
			expected: 123.456,
		},
		{
			name:     "string not to float",
			value:    "+123.456",
			expected: "+123.456",
		},
		{
			name:     "slice",
			value:    []any{"sup", "true", []any{"123", "-123"}, "123.456"},
			expected: []any{"sup", true, []any{uint64(123), int64(-123)}, 123.456},
		},
		{
			name: "map",
			value: map[string]any{
				"sup":   true,
				"123":   -123,
				"slice": []any{"123.456"},
			},
			expected: map[any]any{
				"sup":       true,
				uint64(123): -123,
				"slice":     []any{123.456},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, confi.SpecifyType(tt.value))
		})
	}
}
