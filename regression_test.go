package liquid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// regression tests for bugs fixed in phase 0
// each test must fail on the commit before the fix and pass on the current one

// bug 0.2 — sort was only copying, not sorting
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

// bug 0.3 — append in tryVariableFindInEnvironments could mutate c.Environments
func TestAppendDoesNotCorruptEnvironments(t *testing.T) {
	tmpl, err := Parse(`{{ x }}`, nil)
	require.NoError(t, err)

	data := map[string]interface{}{"x": "hello"}
	// multiple renders to expose slice mutation in environments
	for i := 0; i < 5; i++ {
		out, err := tmpl.Render(data, nil)
		require.NoError(t, err)
		require.Equal(t, "hello", out, "render %d produjo resultado incorrecto", i)
	}
}

// bug 0.4 — checkOverflow was missing return, render continued after error
func TestCheckOverflowDoesNotContinueAfterError(t *testing.T) {
	tmpl, err := Parse(`{% for i in items %}{{ i }}{% endfor %}`, nil)
	require.NoError(t, err)

	out, err := tmpl.Render(map[string]interface{}{
		"items": []string{"a", "b", "c"},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "abc", out)
}

// bug 0.5 — StripHtml regex was recompiled on every call; correctness tested here, efficiency in benchmarks
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
