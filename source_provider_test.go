package confi_test

import (
	"context"
	"os"
	"testing"

	"github.com/jfk9w-go/confi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSourceProvider_GetSources(t *testing.T) {
	tests := []struct {
		name     string
		provider *confi.DefaultSourceProvider
		expected []confi.Source
	}{
		{
			name: "basic env",
			provider: &confi.DefaultSourceProvider{
				EnvPrefix: "test_app_",
				Env: []string{
					"test_app_properties_property=value",
					"test_app_properties_duration=10m",
					"ignored_property=ignored",
					"=",
				},
			},
			expected: []confi.Source{
				confi.PropertySource{
					{Path: []string{"properties", "property"}, Value: "value"},
					{Path: []string{"properties", "duration"}, Value: "10m"},
				},
			},
		},
		{
			name: "basic args",
			provider: &confi.DefaultSourceProvider{
				Args: []string{
					"--properties.property=value",
					"--properties.duration=10m",
					"meme",
					"--properties.flag",
				},
			},
			expected: []confi.Source{
				confi.PropertySource{
					{Path: []string{"properties", "property"}, Value: "value"},
					{Path: []string{"properties", "duration"}, Value: "10m"},
					{Path: []string{"properties", "flag"}, Value: "true"},
				},
			},
		},
		{
			name: "env file and stdin",
			provider: &confi.DefaultSourceProvider{
				EnvPrefix: "test_app_",
				Env: []string{
					"test_app_config_file=config.yaml",
					"test_app_config_stdin=json",
				},
			},
			expected: []confi.Source{
				confi.InputSource{Input: confi.File("config.yaml"), Format: "yaml"},
				confi.InputSource{Input: confi.Reader{R: os.Stdin}, Format: "json"},
			},
		},
		{
			name: "arg file and stdin",
			provider: &confi.DefaultSourceProvider{
				Args: []string{
					"--config.file=config.yaml",
					"--config.stdin=json",
				},
			},
			expected: []confi.Source{
				confi.InputSource{Input: confi.File("config.yaml"), Format: "yaml"},
				confi.InputSource{Input: confi.Reader{R: os.Stdin}, Format: "json"},
			},
		},
		{
			name: "override env file and stdin with arg",
			provider: &confi.DefaultSourceProvider{
				EnvPrefix: "test_app_",
				Env: []string{
					"test_app_config_file=config.yaml",
					"test_app_config_stdin=json",
					"test_app_properties_property=value",
				},
				Args: []string{
					"--config.file=config1.yaml",
					"--config.file=config2.xml",
					"--config.stdin=",
					"--properties.duration=10m",
				},
			},
			expected: []confi.Source{
				confi.PropertySource{
					{Path: []string{"properties", "property"}, Value: "value"},
				},
				confi.InputSource{Input: confi.File("config1.yaml"), Format: "yaml"},
				confi.InputSource{Input: confi.File("config2.xml"), Format: "xml"},
				confi.PropertySource{
					{Path: []string{"properties", "duration"}, Value: "10m"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources, err := tt.provider.GetSources(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, sources)
		})
	}
}
