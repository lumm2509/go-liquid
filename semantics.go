package liquid

import (
	"fmt"
	"reflect"
	"strings"
)

// IsTruthy define la semántica de verdad para este engine.
// Esta tabla es parte del contrato público de la librería.
//
// Decisión de diseño (Opción B — Liquid-inspired):
// A diferencia de Ruby Liquid, este engine trata "", 0 y colecciones vacías como falsy.
// Si necesitas compatibilidad exacta con Ruby Liquid (donde "" y 0 son truthy),
// consulta COMPATIBILITY.md.
//
//	nil          → false
//	false        → false
//	true         → true
//	""           → false   (Ruby Liquid: true)
//	"0"          → true    (string no-vacío es truthy)
//	0 (int)      → false   (Ruby Liquid: true)
//	0.0 (float)  → false   (Ruby Liquid: true)
//	[]           → false
//	[]{1}        → true
//	map{}        → false
func IsTruthy(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return false
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		}
		return true
	}
}

// CompareValues compara dos valores para ordenamiento.
// Retorna -1, 0, o 1 (al estilo de cmp.Compare).
//
// Tabla de comportamiento (contrato público):
//
//	int vs int         → comparación numérica directa
//	float64 vs float64 → comparación numérica directa
//	int vs float64     → coerción a float64, comparación numérica
//	string vs string   → comparación lexicográfica (strings.Compare)
//	string vs int      → NO coerción. string siempre > int. Documentado.
//	nil vs no-nil      → nil < no-nil
//	nil vs nil         → 0
//	tipos no comparables → fallback fmt.Sprintf("%v") — estable pero no semántico
func CompareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Numeric comparisons with type coercion between int/float64
	switch av := a.(type) {
	case int:
		switch bv := b.(type) {
		case int:
			return cmpInt(av, bv)
		case int64:
			return cmpInt64(int64(av), bv)
		case float64:
			return cmpFloat(float64(av), bv)
		}
	case int64:
		switch bv := b.(type) {
		case int:
			return cmpInt64(av, int64(bv))
		case int64:
			return cmpInt64(av, bv)
		case float64:
			return cmpFloat(float64(av), bv)
		}
	case float64:
		switch bv := b.(type) {
		case float64:
			return cmpFloat(av, bv)
		case int:
			return cmpFloat(av, float64(bv))
		case int64:
			return cmpFloat(av, float64(bv))
		}
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}
		// string vs non-string: string is always greater
		return 1
	}

	// Fallback: lexicographic comparison via Sprintf
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
