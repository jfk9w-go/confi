package confi

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Formatted interface {
	SchemaFormat() string
}

type Patterned interface {
	SchemaPattern() string
}

type Schema struct {
	Type                 string            `yaml:"type"`
	Items                *Schema           `yaml:"items,omitempty"`
	Properties           map[string]Schema `yaml:"properties,omitempty"`
	AdditionalProperties any               `yaml:"additionalProperties,omitempty"`
	Required             []string          `yaml:"required,omitempty"`

	// properties below are applied to primitive or inner types
	Enum             any     `yaml:"enum,omitempty" prop:"inner,array"`
	Examples         any     `yaml:"examples,omitempty" prop:"inner,array"`
	Pattern          string  `yaml:"pattern,omitempty" prop:"inner"`
	Format           string  `yaml:"format,omitempty" prop:"inner" alias:"fmt"`
	Minimum          any     `yaml:"minimum,omitempty" prop:"inner" alias:"min"`
	ExclusiveMinimum any     `yaml:"exclusiveMinimum,omitempty" prop:"inner" alias:"xmin"`
	Maximum          any     `yaml:"maximum,omitempty" prop:"inner" alias:"max"`
	ExclusiveMaximum any     `yaml:"exclusiveMaximum,omitempty" prop:"inner" alias:"xmax"`
	MultipleOf       any     `yaml:"multipleOf,omitempty" prop:"inner" alias:"mul"`
	MinLength        *uint64 `yaml:"minLength,omitempty" prop:"inner" alias:"minlen"`
	MaxLength        *uint64 `yaml:"maxLength,omitempty" prop:"inner" alias:"maxlen"`

	// properties below are applied to primitive or outer types
	Description   string  `yaml:"description,omitempty" prop:"outer" alias:"desc,doc"`
	Default       any     `yaml:"default,omitempty" prop:"outer" alias:"def"`
	MinItems      *uint64 `yaml:"minItems,omitempty" prop:"outer" alias:"minsize"`
	MaxItems      *uint64 `yaml:"maxItems,omitempty" prop:"outer" alias:"maxsize"`
	UniqueItems   bool    `yaml:"uniqueItems,omitempty" prop:"outer" alias:"unique"`
	MinProperties *uint64 `yaml:"minProperties,omitempty" prop:"outer" alias:"minprops"`
	MaxProperties *uint64 `yaml:"maxProperties,omitempty" prop:"outer" alias:"maxprops"`
}

func makeSchema(source any, tag reflect.StructTag) (*Schema, error) {
	value := reflect.ValueOf(source)
	for value.Kind() == reflect.Pointer {
		value = reflect.Indirect(value)
	}

	var node yaml.Node
	if err := node.Encode(value.Interface()); err != nil {
		return nil, errors.Wrap(err, "encode")
	}

	var (
		valueType = value.Type()
		elemType  reflect.Type
	)

	var s Schema
	switch {
	case node.Tag == "!!str":
		s.Type = "string"
		switch value.Interface().(type) {
		case regexp.Regexp:
			s.Format = "regex"
		case time.Duration:
			s.Pattern = `(\d+h)?(\d+m)?(\d+s)?(\d+ms)?(\d+µs)?(\d+ns)?`
		case time.Time:
			s.Format = "date-time"
		default:
			if formatted, ok := value.Interface().(Formatted); ok {
				s.Format = formatted.SchemaFormat()
			}
			if patterned, ok := value.Interface().(Patterned); ok {
				s.Pattern = patterned.SchemaPattern()
			}
		}

	case node.Tag == "!!int":
		switch {
		case value.CanInt(), value.CanUint():
			s.Type = "integer"
		default:
			s.Type = "number"
		}

	case node.Tag == "!!float":
		s.Type = "number"

	case node.Tag == "!!bool":
		s.Type = "boolean"

	case node.Tag == "!!seq" && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array):
		elemType = value.Type().Elem()
		items, err := makeSchema(reflect.New(elemType).Interface(), tag)
		if err != nil {
			return nil, errors.Wrap(err, "generate items")
		}

		s.Type = "array"
		s.Items = items
		if value.Kind() == reflect.Array {
			s.MinItems = pointer.To[uint64](uint64(value.Len()))
			s.MaxItems = pointer.To[uint64](uint64(value.Len()))
		}

	case node.Tag == "!!map" && value.Kind() == reflect.Map:
		elemType = value.Type().Elem()
		additionalProperties, err := makeSchema(reflect.New(elemType).Interface(), tag)
		if err != nil {
			return nil, errors.Wrap(err, "generate additionalProperties")
		}

		s.Type = "object"
		s.AdditionalProperties = additionalProperties

	case node.Tag == "!!map" && value.Kind() == reflect.Struct:
		properties, required, err := makeStructSchema(value)
		if err != nil {
			return nil, errors.Wrap(err, "generate properties & required")
		}

		s.Type = "object"
		s.Properties = properties
		s.AdditionalProperties = false
		s.Required = required
	}

	if s.Type == "" {
		return nil, errors.Errorf("unable to detect type for %s %T", node.Tag, source)
	}

	if err := applySchemaProps(&s, tag, valueType, elemType); err != nil {
		return nil, errors.Wrap(err, "apply props")
	}

	return &s, nil
}

func applySchemaProps(s *Schema, tag reflect.StructTag, valueType, elemType reflect.Type) error {
	schema := reflect.ValueOf(s)
	for schema.Kind() == reflect.Pointer {
		schema = reflect.Indirect(schema)
	}

	schemaType := schema.Type()
	for fieldNum := 0; fieldNum < schemaType.NumField(); fieldNum++ {
		var (
			field    = schemaType.Field(fieldNum)
			propType string
			isArray  bool
			aliases  []string
		)

		for i, option := range strings.Split(field.Tag.Get("prop"), ",") {
			switch {
			case i == 0:
				propType = option
			case option == "array":
				isArray = true
			}
		}

		if propType == "" {
			continue
		}

		for _, option := range strings.Split(field.Tag.Get("yaml"), ",") {
			aliases = append(aliases, option)
			break
		}

		for _, option := range strings.Split(field.Tag.Get("alias"), ",") {
			aliases = append(aliases, option)
		}

		if len(aliases) == 0 {
			continue
		}

		var prop string
		for _, alias := range aliases {
			if tag, ok := tag.Lookup(alias); ok {
				prop = tag
				break
			}
		}

		if prop == "" {
			continue
		}

		if elemType == nil {
			elemType = valueType
		}

		var fieldValue reflect.Value
		if field.Type.String() == "interface {}" {
			if isArray {
				prop = "[" + prop + "]"
				fieldValue = reflect.New(reflect.SliceOf(elemType))
			} else {
				fieldValue = reflect.New(elemType)
			}
		} else {
			fieldValue = reflect.New(field.Type)
		}

		if err := yaml.Unmarshal([]byte(prop), fieldValue.Interface()); err != nil {
			return errors.Wrapf(err, "unmarshal %s", prop)
		}

		fieldValue = reflect.Indirect(fieldValue)
		if s.Type == "array" && propType == "inner" {
			reflect.Indirect(reflect.ValueOf(s.Items)).Field(fieldNum).Set(fieldValue)
		} else if s.Type == "object" && propType == "inner" && s.AdditionalProperties != false {
			reflect.ValueOf(s.AdditionalProperties).Field(fieldNum).Set(fieldValue)
		} else {
			schema.Field(fieldNum).Set(fieldValue)
		}
	}

	return nil
}

func makeStructSchema(value reflect.Value) (map[string]Schema, []string, error) {
	var (
		properties = make(map[string]Schema)
		required   = make([]string, 0)
		valueType  = value.Type()
	)

	for fieldNum := 0; fieldNum < valueType.NumField(); fieldNum++ {
		var (
			name      string
			inline    bool
			omitempty bool
		)

		field := valueType.Field(fieldNum)
		if !field.IsExported() {
			continue
		}

		for i, flag := range strings.Split(field.Tag.Get("yaml"), ",") {
			switch {
			case i == 0:
				name = flag
			case flag == "inline":
				inline = true
			case flag == "omitempty":
				omitempty = true
			}
		}

		if name == "" {
			name = field.Name
		}

		if inline {
			embedded, err := makeSchema(reflect.New(field.Type).Interface(), "")
			if err != nil {
				return nil, nil, errors.Wrapf(err, "generate embedded schema for %s", name)
			}

			required = append(required, embedded.Required...)
			for name, property := range embedded.Properties {
				properties[name] = property
			}

			continue
		}

		if name == "-" {
			continue
		}

		if !omitempty {
			required = append(required, name)
		}

		property, err := makeSchema(value.Field(fieldNum).Interface(), field.Tag)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "generate schema for %s", name)
		}

		properties[name] = *property
	}

	return properties, required, nil
}
