package confi_test

import (
	"bytes"
	"testing"

	"github.com/jfk9w-go/confi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	type Value struct {
		String string `yaml:"string"`
	}

	for name, codec := range confi.Codecs {
		t.Run(name, func(t *testing.T) {
			expected := Value{String: "123"}
			var b bytes.Buffer
			err := codec.Marshal(expected, &b)
			require.NoError(t, err)

			var actual Value
			err = codec.Unmarshal(bytes.NewReader(b.Bytes()), &actual)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}
