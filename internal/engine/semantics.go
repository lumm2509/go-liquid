package engine

import (
	"reflect"
	"strings"
)

// IsTruthy defines the truthiness semantics for this engine.
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
	// D6: fast path for dominant Liquid collection types — avoids reflect.ValueOf
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	case []string:
		return len(val) > 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		}
		return true
	}
}

// CompareValues compares two values for ordering. Returns -1, 0, or 1.
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
		return 1
	}

	// D7: UtilsToString has fast paths for all primitive types — avoids fmt.Sprintf alloc
	return strings.Compare(UtilsToString(a), UtilsToString(b))
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
