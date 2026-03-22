package engine

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var UnixTimestampRegex = regexp.MustCompile(`^\d+$`)

// mapKeyString returns the string representation of a map key without fmt.Sprintf allocations
// for the common case where the key is already a string.
func mapKeyString(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprintf("%v", v.Interface())
}

func SliceCollection(collection interface{}, from, to *int) []interface{} {
	// Fast path: map[string]interface{} — avoids reflect.Value map key handling entirely
	if m, ok := collection.(map[string]interface{}); ok {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		res := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			res = append(res, []interface{}{k, m[k]})
		}
		start := 0
		if from != nil {
			start = *from
		}
		end := len(res)
		if to != nil && *to < end {
			end = *to
		}
		if start > end {
			return []interface{}{}
		}
		return res[start:end]
	}

	rv := reflect.ValueOf(collection)
	if rv.Kind() == reflect.Map {
		keys := rv.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(mapKeyString(a), mapKeyString(b))
		})
		res := make([]interface{}, 0, rv.Len())
		for _, key := range keys {
			val := rv.MapIndex(key).Interface()
			res = append(res, []interface{}{key.Interface(), val})
		}
		start := 0
		if from != nil {
			start = *from
		}
		end := len(res)
		if to != nil && *to < end {
			end = *to
		}
		if start > end {
			return []interface{}{}
		}
		return res[start:end]
	}

	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		if s, ok := collection.(string); ok {
			if s == "" {
				return []interface{}{}
			}
			return []interface{}{s}
		}
		return []interface{}{}
	}

	start := 0
	if from != nil {
		start = *from
	}
	end := rv.Len()
	if to != nil && *to < end {
		end = *to
	}
	if start > end {
		return []interface{}{}
	}
	res := make([]interface{}, end-start)
	for i := start; i < end; i++ {
		res[i-start] = rv.Index(i).Interface()
	}
	return res
}

func UtilsToInteger(num interface{}) (int, error) {
	switch v := num.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid integer")
		}
		return i, nil
	default:
		return 0, fmt.Errorf("invalid integer")
	}
}

func UtilsToNumber(obj interface{}) interface{} {
	switch v := obj.(type) {
	case float64:
		return v
	case int:
		return v
	case int64:
		return v
	case string:
		s := strings.TrimSpace(v)
		if strings.ContainsRune(s, '.') {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		} else {
			if i, err := strconv.Atoi(s); err == nil {
				return i
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
		return 0
	default:
		return 0
	}
}

func UtilsToDate(obj interface{}) *time.Time {
	if t, ok := obj.(time.Time); ok {
		return &t
	}
	if t, ok := obj.(*time.Time); ok {
		return t
	}
	s, ok := obj.(string)
	if !ok {
		return nil
	}
	if s == "" {
		return nil
	}
	s = strings.ToLower(s)
	if s == "now" || s == "today" {
		t := time.Now()
		return &t
	}
	if UnixTimestampRegex.MatchString(s) {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			t := time.Unix(i, 0)
			return &t
		}
	}
	formats := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return &t
		}
	}
	return nil
}

func UtilsToString(obj interface{}) string {
	if obj == nil {
		return ""
	}
	switch v := obj.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", obj)
	}
}

func ToInt(input interface{}) int {
	i, _ := UtilsToInteger(input)
	return i
}

// toFloat64 for internal numeric filter use
func toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}
