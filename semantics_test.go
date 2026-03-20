package liquid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTruthinessTable es el Semantic Lock Test para IsTruthy.
// Esta tabla NO SE MODIFICA sin revisión explícita y un commit que
// documente exactamente qué divergencia se introduce y por qué.
// Es el contrato semántico del engine respecto a truthiness.
func TestTruthinessTable(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected bool
		label    string
	}{
		{nil, false, "nil"},
		{false, false, "false"},
		{true, true, "true"},
		{"", false, "empty string"},
		{"0", true, "string zero — non-empty string is truthy"},
		{"false", true, "string false — non-empty string is truthy"},
		{"hello", true, "non-empty string"},
		{0, false, "int zero — differs from Ruby Liquid (Ruby: true)"},
		{1, true, "int one"},
		{-1, true, "int negative"},
		{int64(0), false, "int64 zero"},
		{int64(1), true, "int64 one"},
		{0.0, false, "float zero — differs from Ruby Liquid (Ruby: true)"},
		{1.0, true, "float one"},
		{[]int{}, false, "empty slice"},
		{[]int{1}, true, "non-empty slice"},
		{[]string{}, false, "empty string slice"},
		{map[string]interface{}{}, false, "empty map"},
		{map[string]interface{}{"k": "v"}, true, "non-empty map"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			require.Equal(t, c.expected, IsTruthy(c.input), "IsTruthy(%v)", c.input)
		})
	}
}

// TestComparisons es el Semantic Lock Test para CompareValues.
// Documenta el comportamiento exacto para tipos mixtos.
func TestComparisons(t *testing.T) {
	cases := []struct {
		a, b     interface{}
		expected int
		label    string
	}{
		// Integers
		{1, 2, -1, "int less"},
		{2, 1, 1, "int greater"},
		{1, 1, 0, "int equal"},
		{0, 0, 0, "int zero equal"},

		// Floats
		{1.5, 2.5, -1, "float less"},
		{2.5, 1.5, 1, "float greater"},
		{1.5, 1.5, 0, "float equal"},

		// Mixed numeric (int vs float64)
		{1, 1.0, 0, "int vs float equal"},
		{1, 2.0, -1, "int less than float"},
		{2, 1.0, 1, "int greater than float"},

		// Strings
		{"a", "b", -1, "string less"},
		{"b", "a", 1, "string greater"},
		{"a", "a", 0, "string equal"},

		// String vs int: string is always greater (no coercion)
		{"1", 1, 1, "string vs int — no coercion, string > int"},

		// Nil
		{nil, nil, 0, "nil vs nil"},
		{nil, 1, -1, "nil less than any"},
		{1, nil, 1, "any greater than nil"},
		{nil, "x", -1, "nil less than string"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			require.Equal(t, c.expected, CompareValues(c.a, c.b))
		})
	}
}
