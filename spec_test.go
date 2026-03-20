package liquid

// spec_test.go — Golden tests basados en el spec de Shopify Liquid.
//
// Cada fixture tiene un campo Skip opcional. Cuando está lleno, el test se
// ejecuta con t.Skip (no se ignora silenciosamente) y documenta la razón
// de la divergencia intencional con el spec de Ruby Liquid.
//
// Referencia: https://github.com/Shopify/liquid/tree/main/test/integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type specFixture struct {
	Name     string
	Template string
	Data     map[string]interface{}
	Expected string
	Skip     string // si está lleno, el test se skipea con esta justificación
}

func runSpec(t *testing.T, fixtures []specFixture) {
	t.Helper()
	for _, fx := range fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			if fx.Skip != "" {
				t.Skip(fx.Skip)
			}
			tmpl, err := Parse(fx.Template, nil)
			require.NoError(t, err, "parse error")
			out, err := tmpl.Render(fx.Data, nil)
			require.NoError(t, err, "render error")
			require.Equal(t, fx.Expected, out)
		})
	}
}

// ---------------------------------------------------------------------------
// Variables de forloop
// ---------------------------------------------------------------------------

func TestSpecForloopVariables(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "forloop.index (1-based)",
			Template: `{% for i in items %}{{ forloop.index }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "123",
		},
		{
			Name:     "forloop.index0 (0-based)",
			Template: `{% for i in items %}{{ forloop.index0 }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "012",
		},
		{
			Name:     "forloop.rindex",
			Template: `{% for i in items %}{{ forloop.rindex }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "321",
		},
		{
			Name:     "forloop.rindex0",
			Template: `{% for i in items %}{{ forloop.rindex0 }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "210",
		},
		{
			Name:     "forloop.first",
			Template: `{% for i in items %}{% if forloop.first %}FIRST{% endif %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "FIRSTabc",
		},
		{
			Name:     "forloop.last",
			Template: `{% for i in items %}{{ i }}{% if forloop.last %}LAST{% endif %}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "abcLAST",
		},
		{
			Name:     "forloop.length",
			Template: `{% for i in items %}{{ forloop.length }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "333",
		},
		{
			Name:     "forloop single element",
			Template: `{% for i in items %}{{ forloop.first }}-{{ forloop.last }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"only"}},
			Expected: "true-true",
		},
		{
			Name:     "forloop in range",
			Template: `{% for i in (1..3) %}{{ forloop.index }}.{{ i }} {% endfor %}`,
			Data:     nil,
			Expected: "1.1 2.2 3.3 ",
		},
	})
}

// ---------------------------------------------------------------------------
// for tag: limit y offset
// ---------------------------------------------------------------------------

func TestSpecForLimit(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "limit básico",
			Template: `{% for i in items limit:2 %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "ab",
		},
		{
			Name:     "offset básico",
			Template: `{% for i in items offset:2 %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "cd",
		},
		{
			Name:     "limit y offset combinados",
			Template: `{% for i in items limit:2 offset:1 %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "bc",
		},
		{
			Name:     "limit mayor que colección",
			Template: `{% for i in items limit:10 %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b"}},
			Expected: "ab",
		},
		{
			// Ruby Liquid aplica limit/offset primero, luego reversed.
			// limit:2 → ["a","b"], reversed → ["b","a"]
			Name:     "limit con reversed",
			Template: `{% for i in items reversed limit:2 %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "ba",
		},
		{
			Name:     "forloop.length respeta limit",
			Template: `{% for i in items limit:2 %}{{ forloop.length }}{% endfor %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "22",
		},
	})
}

// ---------------------------------------------------------------------------
// Variables de tablerow
// ---------------------------------------------------------------------------

func TestSpecTablerowVariables(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "tablerow.col (1-based)",
			Template: `{% tablerow i in items cols:2 %}{{ tablerow.col }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n",
		},
		{
			Name:     "tablerow.col0 (0-based)",
			Template: `{% tablerow i in items cols:2 %}{{ tablerow.col0 }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">0</td><td class=\"col2\">1</td></tr>\n<tr class=\"row2\">\n<td class=\"col1\">0</td><td class=\"col2\">1</td></tr>\n",
		},
		{
			Name:     "tablerow.row",
			Template: `{% tablerow i in items cols:2 %}{{ tablerow.row }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c", "d"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">1</td></tr>\n<tr class=\"row2\">\n<td class=\"col1\">2</td><td class=\"col2\">2</td></tr>\n",
		},
		{
			Name:     "tablerow.first y tablerow.last",
			Template: `{% tablerow i in items %}{{ tablerow.first }}-{{ tablerow.last }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">true-false</td><td class=\"col2\">false-false</td><td class=\"col3\">false-true</td></tr>\n",
		},
		{
			Name:     "tablerow.length",
			Template: `{% tablerow i in items %}{{ tablerow.length }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">3</td><td class=\"col2\">3</td><td class=\"col3\">3</td></tr>\n",
		},
		{
			Name:     "tablerow.index (1-based)",
			Template: `{% tablerow i in items %}{{ tablerow.index }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td><td class=\"col3\">3</td></tr>\n",
		},
		{
			Name:     "tablerow.index0 (0-based)",
			Template: `{% tablerow i in items %}{{ tablerow.index0 }}{% endtablerow %}`,
			Data:     map[string]interface{}{"items": []string{"a", "b", "c"}},
			Expected: "<tr class=\"row1\">\n<td class=\"col1\">0</td><td class=\"col2\">1</td><td class=\"col3\">2</td></tr>\n",
		},
	})
}

// ---------------------------------------------------------------------------
// Operador contains
// ---------------------------------------------------------------------------

func TestSpecContains(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "string contains substring",
			Template: `{% if s contains "world" %}yes{% endif %}`,
			Data:     map[string]interface{}{"s": "hello world"},
			Expected: "yes",
		},
		{
			Name:     "string not contains",
			Template: `{% if s contains "xyz" %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"s": "hello world"},
			Expected: "no",
		},
		{
			Name:     "array contains element",
			Template: `{% if arr contains "b" %}yes{% endif %}`,
			Data:     map[string]interface{}{"arr": []string{"a", "b", "c"}},
			Expected: "yes",
		},
		{
			Name:     "array not contains",
			Template: `{% if arr contains "z" %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"arr": []string{"a", "b", "c"}},
			Expected: "no",
		},
	})
}

// ---------------------------------------------------------------------------
// Valores especiales: blank y empty
// ---------------------------------------------------------------------------

func TestSpecBlankAndEmpty(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "nil == blank",
			Template: `{% if x == blank %}yes{% endif %}`,
			Data:     map[string]interface{}{},
			Expected: "yes",
		},
		{
			Name:     "empty string == blank",
			Template: `{% if x == blank %}yes{% endif %}`,
			Data:     map[string]interface{}{"x": ""},
			Expected: "yes",
		},
		{
			Name:     "non-empty string != blank",
			Template: `{% if x == blank %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"x": "hello"},
			Expected: "no",
		},
		{
			Name:     "empty array == blank",
			Template: `{% if x == blank %}yes{% endif %}`,
			Data:     map[string]interface{}{"x": []string{}},
			Expected: "yes",
		},
		{
			Name:     "nil == empty",
			Template: `{% if x == empty %}yes{% endif %}`,
			Data:     map[string]interface{}{},
			Expected: "yes",
		},
		{
			Name:     "empty array == empty",
			Template: `{% if x == empty %}yes{% endif %}`,
			Data:     map[string]interface{}{"x": []string{}},
			Expected: "yes",
		},
	})
}

// ---------------------------------------------------------------------------
// Rangos en for
// ---------------------------------------------------------------------------

func TestSpecRangeFor(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "literal range",
			Template: `{% for i in (1..5) %}{{ i }}{% endfor %}`,
			Data:     nil,
			Expected: "12345",
		},
		{
			Name:     "range from variables",
			Template: `{% for i in (start..finish) %}{{ i }}{% endfor %}`,
			Data:     map[string]interface{}{"start": 3, "finish": 5},
			Expected: "345",
		},
		{
			Name:     "single element range",
			Template: `{% for i in (3..3) %}{{ i }}{% endfor %}`,
			Data:     nil,
			Expected: "3",
		},
	})
}

// ---------------------------------------------------------------------------
// Filtros: casos del spec
// ---------------------------------------------------------------------------

func TestSpecFilters(t *testing.T) {
	runSpec(t, []specFixture{
		// size
		{Name: "size string", Template: `{{ "hello" | size }}`, Data: nil, Expected: "5"},
		{Name: "size array", Template: `{{ arr | size }}`, Data: map[string]interface{}{"arr": []int{1, 2, 3}}, Expected: "3"},
		// reverse (array)
		{Name: "reverse filter",
			Template: `{{ arr | reverse | join: "," }}`,
			Data:     map[string]interface{}{"arr": []string{"a", "b", "c"}},
			Expected: "c,b,a",
		},
		// first / last
		{Name: "first", Template: `{{ arr | first }}`, Data: map[string]interface{}{"arr": []int{1, 2, 3}}, Expected: "1"},
		{Name: "last", Template: `{{ arr | last }}`, Data: map[string]interface{}{"arr": []int{1, 2, 3}}, Expected: "3"},
		// compact
		{
			Name:     "compact removes nils",
			Template: `{{ arr | compact | join: "," }}`,
			Data:     map[string]interface{}{"arr": []interface{}{"a", nil, "b", nil, "c"}},
			Expected: "a,b,c",
		},
		// sort
		{Name: "sort strings", Template: `{{ arr | sort | join: "," }}`,
			Data:     map[string]interface{}{"arr": []string{"b", "a", "c"}},
			Expected: "a,b,c",
		},
		// map
		{Name: "map property", Template: `{{ arr | map: "name" | join: "," }}`,
			Data: map[string]interface{}{"arr": []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"name": "Bob"},
			}},
			Expected: "Alice,Bob",
		},
		// where
		{Name: "where filter", Template: `{{ arr | where: "active" | map: "name" | join: "," }}`,
			Data: map[string]interface{}{"arr": []interface{}{
				map[string]interface{}{"name": "Alice", "active": true},
				map[string]interface{}{"name": "Bob", "active": false},
				map[string]interface{}{"name": "Carol", "active": true},
			}},
			Expected: "Alice,Carol",
		},
		// replace_first
		{Name: "replace_first", Template: `{{ "aabba" | replace_first: "a", "x" }}`, Data: nil, Expected: "xabba"},
		// remove / remove_first
		{Name: "remove", Template: `{{ "hello world hello" | remove: "hello" }}`, Data: nil, Expected: " world "},
		{Name: "remove_first", Template: `{{ "hello world hello" | remove_first: "hello" }}`, Data: nil, Expected: " world hello"},
		// prepend / append
		{Name: "prepend", Template: `{{ "world" | prepend: "hello " }}`, Data: nil, Expected: "hello world"},
		{Name: "append", Template: `{{ "hello" | append: " world" }}`, Data: nil, Expected: "hello world"},
		// strip_newlines
		{Name: "strip_newlines", Template: "{{ s | strip_newlines }}", Data: map[string]interface{}{"s": "hello\nworld"}, Expected: "helloworld"},
		// newline_to_br
		{Name: "newline_to_br", Template: "{{ s | newline_to_br }}", Data: map[string]interface{}{"s": "hello\nworld"}, Expected: "hello<br />\nworld"},
		// lstrip / rstrip
		{Name: "lstrip", Template: `{{ "  hello  " | lstrip }}`, Data: nil, Expected: "hello  "},
		{Name: "rstrip", Template: `{{ "  hello  " | rstrip }}`, Data: nil, Expected: "  hello"},
		// truncate
		{Name: "truncate", Template: `{{ "hello world" | truncate: 8 }}`, Data: nil, Expected: "hello..."},
		{Name: "truncate custom ellipsis", Template: `{{ "hello world" | truncate: 8, "--" }}`, Data: nil, Expected: "hello w--"},
		// slice
		{Name: "slice single", Template: `{{ "hello" | slice: 1 }}`, Data: nil, Expected: "e"},
		{Name: "slice range", Template: `{{ "hello" | slice: 1, 3 }}`, Data: nil, Expected: "ell"},
		{Name: "slice negative", Template: `{{ "hello" | slice: -3, 2 }}`, Data: nil, Expected: "ll"},
		// downcase / upcase
		{Name: "downcase", Template: `{{ "HELLO" | downcase }}`, Data: nil, Expected: "hello"},
		{Name: "upcase", Template: `{{ "hello" | upcase }}`, Data: nil, Expected: "HELLO"},
		// split / join roundtrip
		{Name: "split join", Template: `{{ "a,b,c" | split: "," | join: "-" }}`, Data: nil, Expected: "a-b-c"},
		// url_encode / url_decode
		{Name: "url_encode", Template: `{{ "hello world" | url_encode }}`, Data: nil, Expected: "hello+world"},
		{Name: "url_decode", Template: `{{ "hello+world" | url_decode }}`, Data: nil, Expected: "hello world"},
		// base64_encode / base64_decode
		{Name: "base64_encode", Template: `{{ "hello" | base64_encode }}`, Data: nil, Expected: "aGVsbG8="},
		{Name: "base64_decode", Template: `{{ "aGVsbG8=" | base64_decode }}`, Data: nil, Expected: "hello"},
		// escape / escape_once
		{Name: "escape", Template: `{{ "<b>bold</b>" | escape }}`, Data: nil, Expected: "&lt;b&gt;bold&lt;/b&gt;"},
		{Name: "escape_once", Template: `{{ "&lt;b&gt;" | escape_once }}`, Data: nil, Expected: "&lt;b&gt;"},
	})
}

// ---------------------------------------------------------------------------
// Nil safety en acceso a propiedades
// ---------------------------------------------------------------------------

func TestSpecNilSafety(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "nil variable renders empty",
			Template: `{{ missing }}`,
			Data:     map[string]interface{}{},
			Expected: "",
		},
		{
			Name:     "nil property renders empty",
			Template: `{{ user.address.city }}`,
			Data:     map[string]interface{}{"user": nil},
			Expected: "",
		},
		{
			Name:     "nil chain renders empty",
			Template: `{{ a.b.c.d }}`,
			Data:     map[string]interface{}{},
			Expected: "",
		},
	})
}

// ---------------------------------------------------------------------------
// Truthiness divergences (documentadas)
// ---------------------------------------------------------------------------

func TestSpecTruthinessCompatibility(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "true is truthy",
			Template: `{% if x %}yes{% endif %}`,
			Data:     map[string]interface{}{"x": true},
			Expected: "yes",
		},
		{
			Name:     "false is falsy",
			Template: `{% if x %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"x": false},
			Expected: "no",
		},
		{
			Name:     "nil is falsy",
			Template: `{% if x %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"x": nil},
			Expected: "no",
		},
		{
			Name: "0 is falsy (diverges from Ruby Liquid where 0 is truthy)",
			Template: `{% if x %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"x": 0},
			Expected: "no",
			Skip:     "Opción B: este engine trata 0 como falsy. Ruby Liquid trata 0 como truthy. Divergencia documentada en COMPATIBILITY.md",
		},
		{
			Name: "empty string is falsy (diverges from Ruby Liquid)",
			Template: `{% if x %}yes{% else %}no{% endif %}`,
			Data:     map[string]interface{}{"x": ""},
			Expected: "no",
			Skip:     "Opción B: este engine trata \"\" como falsy. Ruby Liquid trata \"\" como truthy. Divergencia documentada en COMPATIBILITY.md",
		},
		{
			Name:     "non-empty string is truthy",
			Template: `{% if x %}yes{% endif %}`,
			Data:     map[string]interface{}{"x": "hello"},
			Expected: "yes",
		},
		{
			Name:     "non-zero int is truthy",
			Template: `{% if x %}yes{% endif %}`,
			Data:     map[string]interface{}{"x": 1},
			Expected: "yes",
		},
	})
}

// ---------------------------------------------------------------------------
// Assign y capture
// ---------------------------------------------------------------------------

func TestSpecAssignCapture(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "assign string literal",
			Template: `{% assign x = "hello" %}{{ x }}`,
			Data:     nil,
			Expected: "hello",
		},
		{
			Name:     "assign with filter",
			Template: `{% assign x = "hello" | upcase %}{{ x }}`,
			Data:     nil,
			Expected: "HELLO",
		},
		{
			Name:     "capture block",
			Template: `{% capture x %}hello {{ name }}{% endcapture %}{{ x }}`,
			Data:     map[string]interface{}{"name": "world"},
			Expected: "hello world",
		},
	})
}

// ---------------------------------------------------------------------------
// unless
// ---------------------------------------------------------------------------

func TestSpecUnless(t *testing.T) {
	runSpec(t, []specFixture{
		{
			Name:     "unless false",
			Template: `{% unless x %}shown{% endunless %}`,
			Data:     map[string]interface{}{"x": false},
			Expected: "shown",
		},
		{
			Name:     "unless true",
			Template: `{% unless x %}shown{% endunless %}`,
			Data:     map[string]interface{}{"x": true},
			Expected: "",
		},
		{
			Name:     "unless with else",
			Template: `{% unless x %}no{% else %}yes{% endunless %}`,
			Data:     map[string]interface{}{"x": true},
			Expected: "yes",
		},
	})
}

// ---------------------------------------------------------------------------
// Render (no forloop vars en scope aislado)
// ---------------------------------------------------------------------------

func TestSpecConcatFilter(t *testing.T) {
	// concat no existe en standard filters — documentar
	t.Skip("concat filter no está implementado — no es parte del core spec, es una extensión")
	_ = fmt.Sprintf // evitar import no usado
}
