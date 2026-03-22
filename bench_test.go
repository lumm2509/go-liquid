package liquid

import (
	"fmt"
	"testing"
)

func BenchmarkParseSimple(b *testing.B) {
	for b.Loop() {
		Parse(`Hello {{ name }}`, nil)
	}
}

func BenchmarkParseWithFilters(b *testing.B) {
	for b.Loop() {
		Parse(`{{ name | upcase | truncate: 20 }}`, nil)
	}
}

func BenchmarkRenderSimple(b *testing.B) {
	tmpl, _ := Parse(`Hello {{ name }}`, nil)
	data := map[string]interface{}{"name": "World"}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

func BenchmarkRenderWithFilters(b *testing.B) {
	tmpl, _ := Parse(`{{ name | upcase | append: "!" }}`, nil)
	data := map[string]interface{}{"name": "hello"}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

func BenchmarkRenderForLoop100(b *testing.B) {
	items := make([]string, 100)
	for i := range items {
		items[i] = fmt.Sprintf("item%d", i)
	}
	tmpl, _ := Parse(`{% for i in items %}{{ i }}{% endfor %}`, nil)
	data := map[string]interface{}{"items": items}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

func BenchmarkRenderCondition(b *testing.B) {
	tmpl, _ := Parse(`{% if user.active and user.verified %}yes{% endif %}`, nil)
	data := map[string]interface{}{"user": map[string]interface{}{"active": true, "verified": true}}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

func BenchmarkRenderNestedLookup(b *testing.B) {
	tmpl, _ := Parse(`{{ user.address.city }}`, nil)
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"address": map[string]interface{}{
				"city": "Madrid",
			},
		},
	}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

func BenchmarkRenderForLoopWithFilters(b *testing.B) {
	items := make([]string, 50)
	for i := range items {
		items[i] = fmt.Sprintf("item number %d", i)
	}
	tmpl, _ := Parse(`{% for i in items %}{{ i | upcase | truncate: 12 }}{% endfor %}`, nil)
	data := map[string]interface{}{"items": items}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

// ---------------------------------------------------------------------------
// S1 — baseline benchmarks required by TASKS.md
// ---------------------------------------------------------------------------

// BenchmarkConditionEval measures the hot path: parsing + evaluating a
// two-sided condition with a comparison operator.
func BenchmarkConditionEval(b *testing.B) {
	tmpl, _ := Parse(`{% if score >= 90 %}A{% elsif score >= 80 %}B{% else %}C{% endif %}`, nil)
	data := map[string]interface{}{"score": 85}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

// BenchmarkVariableRender measures rendering a variable through a chain of
// builtin filters — the S2 fast path target.
func BenchmarkVariableRender(b *testing.B) {
	tmpl, _ := Parse(`{{ title | downcase | strip | truncate: 30 }}`, nil)
	data := map[string]interface{}{"title": "  Hello World  "}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

// BenchmarkFindVariable measures variable lookup across a multi-level scope
// stack (common in templates with for-loops and partials).
func BenchmarkFindVariable(b *testing.B) {
	tmpl, _ := Parse(`{{ user.address.city }}`, nil)
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"address": map[string]interface{}{"city": "Madrid"},
		},
	}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

// BenchmarkTemplateCacheGet measures concurrent cache reads at high goroutine
// count — the B1/B2 target.
func BenchmarkTemplateCacheGet(b *testing.B) {
	env := NewEnvironment()
	cache := NewTemplateCache(env)
	src := `Hello {{ name | upcase }}!`
	// Pre-warm the cache so we measure reads, not parses.
	cache.Get("tpl", src)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get("tpl", src)
		}
	})
}

// BenchmarkSortFilter measures the | sort: "property" filter on a list of
// maps — the B5 optimisation target.
func BenchmarkSortFilter(b *testing.B) {
	items := make([]interface{}, 100)
	for i := range items {
		items[i] = map[string]interface{}{"score": 100 - i, "name": fmt.Sprintf("item%d", i)}
	}
	tmpl, _ := Parse(`{% assign sorted = items | sort: "score" %}{% for it in sorted %}{{ it.name }}{% endfor %}`, nil)
	data := map[string]interface{}{"items": items}
	for b.Loop() {
		tmpl.Render(data, nil)
	}
}

// BenchmarkForLoop measures iteration over a typed slice of 1000 items — the
// B4 lazy-iterator target.
func BenchmarkForLoop(b *testing.B) {
	b.Run("1000items", func(b *testing.B) {
		items := make([]string, 1000)
		for i := range items {
			items[i] = fmt.Sprintf("item%d", i)
		}
		tmpl, _ := Parse(`{% for i in items %}{{ i }}{% endfor %}`, nil)
		data := map[string]interface{}{"items": items}
		for b.Loop() {
			tmpl.Render(data, nil)
		}
	})
}
