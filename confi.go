package confi

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type App interface {
	Stdout() io.Writer
	Exit()
}

type defaultApp struct{}

func (defaultApp) Stdout() io.Writer { return os.Stdout }
func (defaultApp) Exit()             { os.Exit(0) }

var DefaultApp App = defaultApp{}

func Default[T any](ctx context.Context, appName string) (*T, error) {
	replacer := strings.NewReplacer(`-`, `_`, `.`, `_`)
	provider := &DefaultSourceProvider{
		EnvPrefix: replacer.Replace(appName) + "_",
		Env:       os.Environ(),
		Args:      os.Args[1:],
		Stdin:     os.Stdin,
	}

	return Parse[T](ctx, provider, DefaultApp)
}

func Parse[T any](ctx context.Context, provider SourceProvider, app App) (*T, error) {
	sources, err := provider.GetSources(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get sources")
	}

	var value struct {
		Config struct {
			Values string `yaml:"values,omitempty" doc:"Dump configuration values in specified format to stdin." enum:"json,yaml,gob"`
			Schema string `yaml:"schema,omitempty" doc:"Dump configuration schema in specified format to stdin." enum:"json,yaml,gob"`
		} `yaml:"config,omitempty"`

		Values T `yaml:",inline"`
	}

	schema, err := GenerateSchema(value)
	if err != nil {
		return nil, errors.Wrapf(err, "generate schema")
	}

	for _, source := range sources {
		values, err := source.GetValues(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "get values from %s", source)
		}

		data, err := yaml.Marshal(specify(values))
		if err != nil {
			return nil, errors.Wrapf(err, "marshal values from %s to yaml", source)
		}

		if err := yaml.Unmarshal(data, &value); err != nil {
			return nil, errors.Wrapf(err, "unmarshal values from %s from yaml", source)
		}
	}

	if err := schema.ApplyDefaults(&value); err != nil {
		return nil, errors.Wrap(err, "apply defaults")
	}

	if err := dump(value.Config.Schema, schema, app); err != nil {
		return nil, errors.Wrap(err, "dump schema")
	}

	if err := dump(value.Config.Values, value.Values, app); err != nil {
		return nil, errors.Wrap(err, "dump config")
	}

	return &value.Values, nil
}

func dump(format string, value any, app App) error {
	if format == "" {
		return nil
	}

	codec, ok := Codecs[format]
	if !ok {
		return errors.Errorf("no codec found for %s", format)
	}

	if err := codec.Marshal(value, app.Stdout()); err != nil {
		return errors.Wrapf(err, "write to stdout")
	}

	app.Exit()
	return nil
}

func specify(value any) any {
	return value
}
