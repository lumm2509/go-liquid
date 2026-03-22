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

// mapKeyString returns a map key as string, avoiding fmt.Sprintf for the common string-key case
func mapKeyString(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprintf("%v", v.Interface())
}

// Iterable is a lazy sequence — avoids materialising a full []interface{} copy before iteration
type Iterable interface {
	Len() int
	At(i int) interface{}
}

// interfaceSliceIterable is the fast path for []interface{} — no reflection.
type interfaceSliceIterable struct{ s []interface{} }

func (it interfaceSliceIterable) Len() int             { return len(it.s) }
func (it interfaceSliceIterable) At(i int) interface{} { return it.s[i] }

// reflectSliceIterable wraps any slice or array via reflect.Value — no per-element copies
type reflectSliceIterable struct {
	rv    reflect.Value
	start int
	end   int
}

func (it reflectSliceIterable) Len() int             { return it.end - it.start }
func (it reflectSliceIterable) At(i int) interface{} { return it.rv.Index(it.start + i).Interface() }

func iterBounds(length int, from, to *int) (int, int) {
	start := 0
	if from != nil && *from > 0 {
		start = *from
		if start > length {
			start = length
		}
	}
	end := length
	if to != nil && *to < end {
		end = *to
	}
	if start > end {
		end = start
	}
	return start, end
}

// ToIterable returns an Iterable over the collection window [from, to); []interface{} skips
// reflection, typed slices wrap reflect.Value lazily, maps are materialised and sorted
func ToIterable(collection interface{}, from, to *int) Iterable {
	// fast path: []interface{} — no reflection needed
	if s, ok := collection.([]interface{}); ok {
		start, end := iterBounds(len(s), from, to)
		return interfaceSliceIterable{s: s[start:end]}
	}

	rv := reflect.ValueOf(collection)

	// maps must be materialised to sort by key
	if rv.Kind() == reflect.Map {
		return interfaceSliceIterable{s: SliceCollection(collection, from, to)}
	}

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		start, end := iterBounds(rv.Len(), from, to)
		return reflectSliceIterable{rv: rv, start: start, end: end}
	}

	// scalar string — single-element collection
	if s, ok := collection.(string); ok && s != "" {
		return interfaceSliceIterable{s: []interface{}{s}}
	}

	return interfaceSliceIterable{}
}

func SliceCollection(collection interface{}, from, to *int) []interface{} {
	// fast path: map[string]interface{} — avoids reflect.Value map key handling entirely
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

func UtilsToDate(obj interface{}) (time.Time, bool) {
	if t, ok := obj.(time.Time); ok {
		return t, true
	}
	if t, ok := obj.(*time.Time); ok && t != nil {
		return *t, true
	}
	s, ok := obj.(string)
	if !ok || s == "" {
		return time.Time{}, false
	}
	s = strings.ToLower(s)
	if s == "now" || s == "today" {
		return time.Now(), true
	}
	if UnixTimestampRegex.MatchString(s) {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(i, 0), true
		}
	}
	for _, f := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func UtilsToString(obj interface{}) string {
	if obj == nil {
		return ""
	}
	switch v := obj.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
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
