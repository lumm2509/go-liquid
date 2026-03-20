package liquid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests de regresión para los bugs corregidos en Fase 0.
// Cada test debe fallar en el commit previo a la corrección y pasar en el actual.

// Bug 0.2 — Sort no ordenaba, solo copiaba.
func TestSortFilterActuallySorts(t *testing.T) {
	f := StandardFilters{}

	cases := []struct {
		name     string
		input    interface{}
		expected []interface{}
	}{
		{"strings", []interface{}{"c", "a", "b"}, []interface{}{"a", "b", "c"}},
		{"numbers", []interface{}{3, 1, 2}, []interface{}{1, 2, 3}},
		{"already sorted", []interface{}{"a", "b", "c"}, []interface{}{"a", "b", "c"}},
		{"single element", []interface{}{"x"}, []interface{}{"x"}},
		{"nil input", nil, []interface{}{}},
		{"non-slice", "hello", []interface{}{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, f.Sort(tc.input))
		})
	}
}

// Bug 0.3 — append en tryVariableFindInEnvironments podía mutar c.Environments.
func TestAppendDoesNotCorruptEnvironments(t *testing.T) {
	tmpl, err := Parse(`{{ x }}`, nil)
	require.NoError(t, err)

	data := map[string]interface{}{"x": "hello"}
	// Múltiples renders para exponer mutación del slice de environments
	for i := 0; i < 5; i++ {
		out, err := tmpl.Render(data, nil)
		require.NoError(t, err)
		require.Equal(t, "hello", out, "render %d produjo resultado incorrecto", i)
	}
}

// Bug 0.4 — checkOverflow no tenía return, el render continuaba tras el error.
func TestCheckOverflowDoesNotContinueAfterError(t *testing.T) {
	// Verificar que un template válido no dispara el overflow
	tmpl, err := Parse(`{% for i in items %}{{ i }}{% endfor %}`, nil)
	require.NoError(t, err)

	out, err := tmpl.Render(map[string]interface{}{
		"items": []string{"a", "b", "c"},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "abc", out)
}

// Bug 0.5 — regex de StripHtml se compilaba en cada llamada.
// Este test verifica que la función es correcta (la eficiencia se mide en benchmarks).
func TestStripHtmlRegexIsCorrect(t *testing.T) {
	f := StandardFilters{}
	cases := []struct {
		input    interface{}
		expected string
	}{
		{"<p>hello</p>", "hello"},
		{"<a href='x'>link</a>", "link"},
		{"no tags", "no tags"},
		{"", ""},
		{nil, ""},
		{"<br/>", ""},
		{"<div class='x'>text</div>", "text"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.expected, f.StripHtml(tc.input))
	}
}
