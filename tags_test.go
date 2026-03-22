package liquid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper para tests de tags via template completo.
func mustRender(t *testing.T, src string, data map[string]interface{}) string {
	t.Helper()
	tmpl, err := Parse(src, nil)
	require.NoError(t, err)
	out, err := tmpl.Render(data, nil)
	require.NoError(t, err)
	return out
}

// --- assign ---

var assignTests = []struct {
	name     string
	template string
	data     map[string]interface{}
	expected string
}{
	{"basic assign", `{% assign x = "hello" %}{{ x }}`, nil, "hello"},
	{"assign from var", `{% assign y = x %}{{ y }}`, map[string]interface{}{"x": "world"}, "world"},
	{"assign number", `{% assign n = 42 %}{{ n }}`, nil, "42"},
	{"assign overrides", `{% assign x = "new" %}{{ x }}`, map[string]interface{}{"x": "old"}, "new"},
}

func TestAssignTag(t *testing.T) {
	for _, tc := range assignTests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, mustRender(t, tc.template, tc.data))
		})
	}
}

// --- if ---

var ifTagTests = []struct {
	name     string
	template string
	data     map[string]interface{}
	expected string
}{
	{"true condition", `{% if x %}yes{% endif %}`, map[string]interface{}{"x": true}, "yes"},
	{"false condition", `{% if x %}yes{% endif %}`, map[string]interface{}{"x": false}, ""},
	{"nil condition", `{% if x %}yes{% endif %}`, map[string]interface{}{}, ""},
	{"elsif", `{% if x %}a{% elsif y %}b{% endif %}`, map[string]interface{}{"y": true}, "b"},
	{"else", `{% if x %}a{% else %}b{% endif %}`, map[string]interface{}{}, "b"},
	{"and operator", `{% if a and b %}yes{% endif %}`, map[string]interface{}{"a": true, "b": true}, "yes"},
	{"and false", `{% if a and b %}yes{% endif %}`, map[string]interface{}{"a": true, "b": false}, ""},
	{"or operator", `{% if a or b %}yes{% endif %}`, map[string]interface{}{"a": false, "b": true}, "yes"},
	{"eq operator", `{% if x == 1 %}yes{% endif %}`, map[string]interface{}{"x": 1}, "yes"},
	{"neq operator", `{% if x != 1 %}yes{% endif %}`, map[string]interface{}{"x": 2}, "yes"},
}

func TestIfTag(t *testing.T) {
	for _, tc := range ifTagTests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, mustRender(t, tc.template, tc.data))
		})
	}
}

// --- unless ---

func TestUnlessTag(t *testing.T) {
	require.Equal(t, "no", mustRender(t, `{% unless x %}no{% endunless %}`, map[string]interface{}{"x": false}))
	require.Equal(t, "", mustRender(t, `{% unless x %}no{% endunless %}`, map[string]interface{}{"x": true}))
	require.Equal(t, "no", mustRender(t, `{% unless x %}no{% endunless %}`, map[string]interface{}{}))
}

// --- for ---

var forTagTests = []struct {
	name     string
	template string
	data     map[string]interface{}
	expected string
}{
	{"basic for", `{% for i in items %}{{ i }}{% endfor %}`, map[string]interface{}{"items": []string{"a", "b", "c"}}, "abc"},
	{"empty collection", `{% for i in items %}{{ i }}{% endfor %}`, map[string]interface{}{"items": []string{}}, ""},
	{"nil collection", `{% for i in items %}{{ i }}{% endfor %}`, map[string]interface{}{}, ""},
	{"for reversed", `{% for i in items reversed %}{{ i }}{% endfor %}`, map[string]interface{}{"items": []string{"a", "b", "c"}}, "cba"},
	{"for with nested access", `{% for i in items %}{{ i.name }}{% endfor %}`, map[string]interface{}{"items": []interface{}{map[string]interface{}{"name": "Alice"}}}, "Alice"},
}

func TestForTag(t *testing.T) {
	for _, tc := range forTagTests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, mustRender(t, tc.template, tc.data))
		})
	}
}

// --- capture ---

func TestCaptureTag(t *testing.T) {
	require.Equal(t, "hello world", mustRender(t, `{% capture msg %}hello {{ name }}{% endcapture %}{{ msg }}`, map[string]interface{}{"name": "world"}))
	require.Equal(t, "AB", mustRender(t, `{% capture x %}A{% endcapture %}{% capture y %}B{% endcapture %}{{ x }}{{ y }}`, nil))
	require.Equal(t, "", mustRender(t, `{% capture x %}{% endcapture %}{{ x }}`, nil))
}

// --- case ---

func TestCaseTag(t *testing.T) {
	require.Equal(t, "one", mustRender(t, `{% case x %}{% when 1 %}one{% when 2 %}two{% else %}other{% endcase %}`, map[string]interface{}{"x": 1}))
	require.Equal(t, "two", mustRender(t, `{% case x %}{% when 1 %}one{% when 2 %}two{% else %}other{% endcase %}`, map[string]interface{}{"x": 2}))
	require.Equal(t, "other", mustRender(t, `{% case x %}{% when 1 %}one{% when 2 %}two{% else %}other{% endcase %}`, map[string]interface{}{"x": 99}))
	require.Equal(t, "", mustRender(t, `{% case x %}{% when 1 %}one{% endcase %}`, map[string]interface{}{"x": 99}))
}

// --- comment ---

func TestCommentTag(t *testing.T) {
	require.Equal(t, "", mustRender(t, `{% comment %}this is hidden{% endcomment %}`, nil))
	// Note: comment tag leaves surrounding whitespace intact
	require.Equal(t, "before  after", mustRender(t, `before {% comment %}hidden{% endcomment %} after`, nil))
}

// --- raw ---

func TestRawTag(t *testing.T) {
	require.Equal(t, "{{ not rendered }}", mustRender(t, `{% raw %}{{ not rendered }}{% endraw %}`, nil))
	require.Equal(t, "{% also raw %}", mustRender(t, `{% raw %}{% also raw %}{% endraw %}`, nil))
}

// --- echo ---

func TestEchoTag(t *testing.T) {
	require.Equal(t, "hello", mustRender(t, `{% echo name %}`, map[string]interface{}{"name": "hello"}))
	require.Equal(t, "WORLD", mustRender(t, `{% echo name | upcase %}`, map[string]interface{}{"name": "world"}))
	require.Equal(t, "", mustRender(t, `{% echo missing %}`, nil))
}

// --- cycle ---

func TestCycleTag(t *testing.T) {
	out := mustRender(t, `{% for i in items %}{% cycle "a","b","c" %}{% endfor %}`, map[string]interface{}{"items": []string{"x", "x", "x", "x"}})
	require.Equal(t, "abca", out)
}

func TestCycleTagWithQuotedCommas(t *testing.T) {
	// Regression: old strings.Split(",") would break "a,b" into two tokens.
	out := mustRender(t, `{% cycle "a,b","c,d" %}{% cycle "a,b","c,d" %}`, nil)
	require.Equal(t, "a,bc,d", out)
}

func TestCycleTagNamedGroup(t *testing.T) {
	out := mustRender(t, `{% cycle "g": "x","y" %}{% cycle "g": "x","y" %}{% cycle "g": "x","y" %}`, nil)
	require.Equal(t, "xyx", out)
}

// --- increment / decrement ---

func TestIncrementTag(t *testing.T) {
	require.Equal(t, "012", mustRender(t, `{% increment x %}{% increment x %}{% increment x %}`, nil))
}

func TestDecrementTag(t *testing.T) {
	require.Equal(t, "-1-2-3", mustRender(t, `{% decrement x %}{% decrement x %}{% decrement x %}`, nil))
}

// --- break / continue ---

func TestBreakTag(t *testing.T) {
	out := mustRender(t, `{% for i in items %}{% if i == "b" %}{% break %}{% endif %}{{ i }}{% endfor %}`, map[string]interface{}{"items": []string{"a", "b", "c"}})
	require.Equal(t, "a", out)
}

func TestContinueTag(t *testing.T) {
	out := mustRender(t, `{% for i in items %}{% if i == "b" %}{% continue %}{% endif %}{{ i }}{% endfor %}`, map[string]interface{}{"items": []string{"a", "b", "c"}})
	require.Equal(t, "ac", out)
}

// --- ifchanged ---

func TestIfchangedTag(t *testing.T) {
	out := mustRender(t, `{% for i in items %}{% ifchanged %}{{ i }}{% endifchanged %}{% endfor %}`, map[string]interface{}{"items": []string{"a", "a", "b", "b", "c"}})
	require.Equal(t, "abc", out)
}

// --- tablerow ---

func TestTablerowTag(t *testing.T) {
	out := mustRender(t, `{% tablerow i in items %}{{ i }}{% endtablerow %}`, map[string]interface{}{"items": []string{"a", "b", "c"}})
	require.Contains(t, out, "<tr")
	require.Contains(t, out, "<td")
	require.Contains(t, out, "a")
	require.Contains(t, out, "b")
	require.Contains(t, out, "c")
}

// --- inline comment ---

func TestInlineCommentTag(t *testing.T) {
	require.Equal(t, "", mustRender(t, `{% # this is a comment %}`, nil))
	require.Equal(t, "hello", mustRender(t, `{% # comment %}hello`, nil))
}

// --- include / partial loading ---

func TestIncludeWithMissingTemplate(t *testing.T) {
	// With eager partial loading (8.C), a FileSystem that rejects includes
	// now causes Parse to fail — not the first Render call.
	_, err := Parse(`{% include 'nonexistent' %}`, nil)
	require.Error(t, err, "expected error when template file is not found")
	require.NotContains(t, err.Error(), "interface conversion", "error must be descriptive, not a runtime panic message")
}

func TestIncludeWithNoFileSystem(t *testing.T) {
	// With C5, env.FileSystem = nil now triggers a parse-time error instead of
	// a silent skip followed by a render-time failure.
	env := NewEnvironment()
	env.FileSystem = nil

	_, err := ParseWithEnv(`{% include 'partial' %}`, env, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "FileSystem", "error must identify the missing FileSystem configuration")
}
