package liquid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// same template rendered twice with different data must produce independent results
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

// assign must not persist into Template.InstanceAssigns across renders
func TestAssignDoesNotLeakBetweenRenders(t *testing.T) {
	tmpl, err := Parse(`{% assign x = "leaked" %}{{ x }}`, nil)
	require.NoError(t, err)

	out1, err := tmpl.Render(map[string]interface{}{}, nil)
	require.NoError(t, err)
	require.Equal(t, "leaked", out1)

	require.Empty(t, tmpl.InstanceAssigns, "assign leaked into Template.InstanceAssigns")

	// x resolves from assign tag each render, not from a leaked state
	out2, err := tmpl.Render(map[string]interface{}{}, nil)
	require.NoError(t, err)
	require.Equal(t, "leaked", out2)
}

// for loop variable must not be visible outside the loop
func TestForLoopVariableDoesNotLeakOutOfScope(t *testing.T) {
	tmpl, err := Parse(`{% for i in items %}{% endfor %}{{ i }}`, nil)
	require.NoError(t, err)

	out, err := tmpl.Render(map[string]interface{}{"items": []string{"a", "b"}}, nil)
	require.NoError(t, err)
	require.Equal(t, "", out)
}

// regression: tag_counters previously wrote into context.Environments[0] (user map)
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
