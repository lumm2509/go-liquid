package liquid

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper para tests de filtros via template completo.
func renderFilter(t *testing.T, filter string, data map[string]interface{}) string {
	t.Helper()
	tmpl, err := Parse(fmt.Sprintf(`{{ val | %s }}`, filter), nil)
	require.NoError(t, err)
	out, err := tmpl.Render(data, nil)
	require.NoError(t, err)
	return out
}

// --- Size ---

func TestSize(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 5, f.Size("hello"))
	require.Equal(t, 3, f.Size([]int{1, 2, 3}))
	require.Equal(t, 0, f.Size(""))
	require.Equal(t, 0, f.Size(nil))
}

// --- Downcase ---

var downcaseTests = []struct {
	input    interface{}
	expected string
}{
	{"HELLO", "hello"},
	{"", ""},
	{nil, ""},
	{123, "123"},
	{"Already lower", "already lower"},
}

func TestDowncase(t *testing.T) {
	f := StandardFilters{}
	for _, tc := range downcaseTests {
		t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
			require.Equal(t, tc.expected, f.Downcase(tc.input))
		})
	}
}

// --- Upcase ---

func TestUpcase(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "HELLO", f.Upcase("hello"))
	require.Equal(t, "", f.Upcase(""))
	require.Equal(t, "", f.Upcase(nil))
	require.Equal(t, "123", f.Upcase(123))
}

// --- Capitalize ---

func TestCapitalize(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "Hello world", f.Capitalize("hello world"))
	require.Equal(t, "Hello world", f.Capitalize("HELLO WORLD"))
	require.Equal(t, "", f.Capitalize(""))
	require.Equal(t, "", f.Capitalize(nil))
}

// --- Escape ---

func TestEscape(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "&lt;p&gt;", f.Escape("<p>"))
	require.Equal(t, "hello", f.Escape("hello"))
	require.Equal(t, "", f.Escape(nil))
	require.Equal(t, "a &amp; b", f.Escape("a & b"))
}

// --- Join ---

func TestJoin(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "a, b, c", f.Join([]interface{}{"a", "b", "c"}, ", "))
	require.Equal(t, "abc", f.Join([]interface{}{"a", "b", "c"}, ""))
	require.Equal(t, "", f.Join(nil, ", "))
	require.Equal(t, "a", f.Join([]interface{}{"a"}, ", "))
}

// --- Strip ---

func TestStrip(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "hello", f.Strip("  hello  "))
	require.Equal(t, "hello", f.Strip("hello"))
	require.Equal(t, "", f.Strip("   "))
	require.Equal(t, "", f.Strip(nil))
}

// --- Split ---

func TestSplit(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, []string{"a", "b", "c"}, f.Split("a,b,c", ","))
	require.Equal(t, []string{"hello"}, f.Split("hello", ","))
	require.Equal(t, []string{""}, f.Split("", ","))
}

// --- Replace ---

func TestReplace(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "b-b-b", f.Replace("a-a-a", "a", "b"))
	require.Equal(t, "hello", f.Replace("hello", "x", "y"))
	require.Equal(t, "", f.Replace(nil, "a", "b"))
}

// --- Plus / Minus ---

func TestPlus(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 5.0, f.Plus(3, 2))
	require.Equal(t, 5.5, f.Plus(3.5, 2.0))
	require.Equal(t, 0.0, f.Plus(nil, nil))
}

func TestMinus(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 1.0, f.Minus(3, 2))
	require.Equal(t, 1.5, f.Minus(3.5, 2.0))
}

// --- Abs ---

func TestAbs(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 5.0, f.Abs(-5))
	require.Equal(t, 5.0, f.Abs(5))
	require.Equal(t, 0.0, f.Abs(0))
	require.Equal(t, 0.0, f.Abs(nil))
}

// --- Times ---

func TestTimes(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 6.0, f.Times(2, 3))
	require.Equal(t, 0.0, f.Times(0, 100))
	require.Equal(t, 0.0, f.Times(nil, 5))
}

// --- DividedBy ---

func TestDividedBy(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 2.0, f.DividedBy(6, 3))
	require.Equal(t, 0.0, f.DividedBy(10, 0)) // división por cero → 0
	require.Equal(t, 2.5, f.DividedBy(5, 2))
}

// --- Modulo ---

func TestModulo(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 1.0, f.Modulo(7, 3))
	require.Equal(t, 0.0, f.Modulo(6, 3))
	require.Equal(t, 0.0, f.Modulo(5, 0)) // módulo por cero → 0
}

// --- Round / Ceil / Floor ---

func TestRound(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 2.0, f.Round(2.3, nil))
	require.Equal(t, 3.0, f.Round(2.7, nil))
	// Float precision: 2.135 is stored as 2.134999... in IEEE 754, rounds to 2.13
	require.Equal(t, 2.13, f.Round(2.135, 2))
	require.Equal(t, 2.5, f.Round(2.5, 1))
}

func TestCeil(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 3.0, f.Ceil(2.1))
	require.Equal(t, 3.0, f.Ceil(3.0))
	require.Equal(t, -2.0, f.Ceil(-2.9))
}

func TestFloor(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, 2.0, f.Floor(2.9))
	require.Equal(t, 3.0, f.Floor(3.0))
	require.Equal(t, -3.0, f.Floor(-2.1))
}

// --- First / Last ---

func TestFirst(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "a", f.First([]interface{}{"a", "b", "c"}))
	require.Nil(t, f.First([]interface{}{}))
	require.Nil(t, f.First(nil))
}

func TestLast(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "c", f.Last([]interface{}{"a", "b", "c"}))
	require.Nil(t, f.Last([]interface{}{}))
	require.Nil(t, f.Last(nil))
}

// --- Default ---

func TestDefault(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "fallback", f.Default(nil, "fallback"))
	require.Equal(t, "fallback", f.Default("", "fallback"))
	require.Equal(t, "fallback", f.Default([]interface{}{}, "fallback"))
	require.Equal(t, "value", f.Default("value", "fallback"))
	require.Equal(t, "fallback", f.Default(false, "fallback"))
}

// --- Append / Prepend ---

func TestAppend(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "hello world", f.Append("hello", " world"))
	require.Equal(t, "hello", f.Append("hello", ""))
	require.Equal(t, " world", f.Append(nil, " world"))
}

func TestPrepend(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "hello world", f.Prepend("world", "hello "))
	require.Equal(t, "world", f.Prepend("world", nil))
}

// --- StripHtml ---

func TestStripHtml(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "hello", f.StripHtml("<b>hello</b>"))
	require.Equal(t, "hello world", f.StripHtml("<p>hello</p> world"))
	require.Equal(t, "", f.StripHtml("<br/>"))
	require.Equal(t, "", f.StripHtml(nil))
}

// --- Truncatewords ---

func TestTruncatewords(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, "one two...", f.Truncatewords("one two three", 2, nil))
	require.Equal(t, "one two—", f.Truncatewords("one two three", 2, "—"))
	require.Equal(t, "one two three", f.Truncatewords("one two three", 5, nil))
	require.Equal(t, "", f.Truncatewords(nil, 3, nil))
}

// --- Json ---

func TestJson(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, `"hello"`, f.Json("hello"))
	require.Equal(t, `42`, f.Json(42))
	require.Equal(t, `null`, f.Json(nil))
	require.Equal(t, `["a","b"]`, f.Json([]string{"a", "b"}))
}

// --- Uniq ---

func TestUniq(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, []interface{}{"a", "b", "c"}, f.Uniq([]interface{}{"a", "b", "a", "c", "b"}))
	require.Equal(t, []interface{}{}, f.Uniq([]interface{}{}))
	require.Equal(t, []interface{}{nil}, f.Uniq(nil)) // non-slice: wrapped
}

// --- Sort ---

func TestSort(t *testing.T) {
	f := StandardFilters{}
	require.Equal(t, []interface{}{"a", "b", "c"}, f.Sort([]interface{}{"c", "a", "b"}))
	require.Equal(t, []interface{}{}, f.Sort(nil))
	require.Equal(t, []interface{}{}, f.Sort("not a slice"))
	require.Equal(t, []interface{}{"x"}, f.Sort([]interface{}{"x"}))
}

// --- Map ---

func TestMap(t *testing.T) {
	f := StandardFilters{}
	input := []interface{}{
		map[string]interface{}{"name": "Alice", "age": 30},
		map[string]interface{}{"name": "Bob", "age": 25},
	}
	result := f.Map(input, "name")
	require.Equal(t, []interface{}{"Alice", "Bob"}, result)

	require.Equal(t, []interface{}{}, f.Map(nil, "name"))
	require.Equal(t, []interface{}{}, f.Map("not a slice", "name"))
}

// --- Where ---

func TestWhere(t *testing.T) {
	f := StandardFilters{}
	input := []interface{}{
		map[string]interface{}{"active": true, "name": "Alice"},
		map[string]interface{}{"active": false, "name": "Bob"},
		map[string]interface{}{"active": true, "name": "Carol"},
	}
	result := f.Where(input, "active", true)
	require.Len(t, result, 2)

	require.Equal(t, []interface{}{}, f.Where(nil, "x"))
}

// --- Date ---

func TestDate(t *testing.T) {
	f := StandardFilters{}
	// nil input retorna input tal cual
	require.Equal(t, nil, f.Date(nil, "%Y"))
	// formato vacío retorna input tal cual
	require.Equal(t, "2024-01-15", f.Date("2024-01-15", ""))
	// formato no vacío y fecha válida
	result := f.Date("2024-01-15", "%Y")
	require.Equal(t, "2024", result)
}
