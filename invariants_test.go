package liquid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// El mismo Template renderizado dos veces con datos distintos
// debe producir resultados distintos e independientes.
func TestContextDoesNotLeakStateBetweenRenders(t *testing.T) {
	tmpl, err := Parse(`{{ x }}`, nil)
	require.NoError(t, err)

	out1, err1 := tmpl.Render(map[string]interface{}{"x": "A"}, nil)
	out2, err2 := tmpl.Render(map[string]interface{}{"x": "B"}, nil)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.Equal(t, "A", out1)
	require.Equal(t, "B", out2)
}

// Render no debe mutar los datos del usuario.
func TestRenderDoesNotMutateInput(t *testing.T) {
	tmpl, err := Parse(`{% for item in items %}{{ item }}{% endfor %}`, nil)
	require.NoError(t, err)

	items := []string{"a", "b", "c"}
	data := map[string]interface{}{"items": items}

	_, err = tmpl.Render(data, nil)
	require.NoError(t, err)

	require.Equal(t, []string{"a", "b", "c"}, data["items"])
	require.Equal(t, items, data["items"].([]string))
}

// El mismo template renderizado N veces produce el mismo output.
func TestRenderIsDeterministic(t *testing.T) {
	tmpl, err := Parse(`{% for i in items %}{{ i | upcase }}{% endfor %}`, nil)
	require.NoError(t, err)

	data := map[string]interface{}{"items": []string{"a", "b", "c"}}
	results := make([]string, 5)
	for i := range results {
		out, err := tmpl.Render(data, nil)
		require.NoError(t, err)
		results[i] = out
	}

	for i := 1; i < len(results); i++ {
		require.Equal(t, results[0], results[i], "render %d produjo resultado distinto", i)
	}
}

// Variables asignadas en un render no deben persistir en Template.InstanceAssigns.
func TestAssignDoesNotLeakBetweenRenders(t *testing.T) {
	tmpl, err := Parse(`{% assign x = "leaked" %}{{ x }}`, nil)
	require.NoError(t, err)

	// Primer render: assign funciona dentro del render
	out1, err := tmpl.Render(map[string]interface{}{}, nil)
	require.NoError(t, err)
	require.Equal(t, "leaked", out1)

	// InstanceAssigns no debe estar contaminado
	require.Empty(t, tmpl.InstanceAssigns, "assign leaked into Template.InstanceAssigns")

	// Segundo render: x sigue funcionando (viene del assign tag, no del leak)
	out2, err := tmpl.Render(map[string]interface{}{}, nil)
	require.NoError(t, err)
	require.Equal(t, "leaked", out2)
}

// Scope de for loop no debe filtrarse fuera del loop.
func TestForLoopVariableDoesNotLeakOutOfScope(t *testing.T) {
	tmpl, err := Parse(`{% for i in items %}{% endfor %}{{ i }}`, nil)
	require.NoError(t, err)

	out, err := tmpl.Render(map[string]interface{}{"items": []string{"a", "b"}}, nil)
	require.NoError(t, err)
	require.Equal(t, "", out)
}

// increment/decrement no deben mutar el mapa de assigns del usuario.
// Regresión: tag_counters usaba context.Environments[0] que es el mapa del usuario.
func TestIncrementDoesNotMutateAssigns(t *testing.T) {
	tmpl, err := Parse(`{% increment counter %}{% increment counter %}`, nil)
	require.NoError(t, err)

	assigns := map[string]interface{}{"counter": "original"}
	out, err := tmpl.Render(assigns, nil)
	require.NoError(t, err)
	require.Equal(t, "01", out)
	require.Equal(t, "original", assigns["counter"], "increment must not mutate user assigns")
}

func TestDecrementDoesNotMutateAssigns(t *testing.T) {
	tmpl, err := Parse(`{% decrement counter %}{% decrement counter %}`, nil)
	require.NoError(t, err)

	assigns := map[string]interface{}{"counter": "original"}
	out, err := tmpl.Render(assigns, nil)
	require.NoError(t, err)
	require.Equal(t, "-1-2", out)
	require.Equal(t, "original", assigns["counter"], "decrement must not mutate user assigns")
}

// El mismo Template puede usarse múltiples veces con datos distintos correctamente.
func TestTemplateIsReusable(t *testing.T) {
	tmpl, err := Parse(`Hello {{ name }}!`, nil)
	require.NoError(t, err)

	out1, _ := tmpl.Render(map[string]interface{}{"name": "Alice"}, nil)
	out2, _ := tmpl.Render(map[string]interface{}{"name": "Bob"}, nil)
	out3, _ := tmpl.Render(map[string]interface{}{"name": "Alice"}, nil)

	require.Equal(t, "Hello Alice!", out1)
	require.Equal(t, "Hello Bob!", out2)
	require.Equal(t, "Hello Alice!", out3)
}
