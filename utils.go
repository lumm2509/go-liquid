package liquid

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var DecimalRegex = regexp.MustCompile(`^-?\d+\.\d+$`)
var UnixTimestampRegex = regexp.MustCompile(`^\d+$`)

func SliceCollection(collection interface{}, from, to *int) []interface{} {
	// Basic implementation of collection slicing
	rv := reflect.ValueOf(collection)
	if rv.Kind() == reflect.Map {
		keys := rv.MapKeys()
		// Sort keys for consistent iteration
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(fmt.Sprintf("%v", a.Interface()), fmt.Sprintf("%v", b.Interface()))
		})

		// Liquid spec: iterating a hash yields [key, value] pairs.
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
		if DecimalRegex.MatchString(s) {
			f, err := strconv.ParseFloat(s, 64)
			if err == nil {
				return f
			}
		}
		i, err := strconv.Atoi(s)
		if err == nil {
			return i
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

	// Try some common formats
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
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
