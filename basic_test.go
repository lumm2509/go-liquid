package liquid

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBasicRender(t *testing.T) {
	source := "Hello {{ name }}"
	template, err := Parse(source, nil)
	require.NoError(t, err)
	require.NotNil(t, template)

	output, err := template.Render(map[string]interface{}{"name": "World"}, nil)
	require.NoError(t, err)
	require.Equal(t, "Hello World", output)
}
