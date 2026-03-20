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
