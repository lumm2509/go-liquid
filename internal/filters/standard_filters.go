package filters

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/lumm2509/go-liquid/internal/engine"
)

var stripHtmlRegex = regexp.MustCompile(`<[^>]*>`)

var strftimeReplacer = strings.NewReplacer(
	"%Y", "2006", "%y", "06", "%m", "01", "%d", "02",
	"%H", "15", "%M", "04", "%S", "05",
	"%B", "January", "%b", "Jan", "%A", "Monday", "%a", "Mon",
	"%e", "2", "%j", "002", "%p", "PM", "%Z", "MST", "%z", "-0700", "%I", "03",
)

// StandardFilters implements the standard Liquid filter set.
type StandardFilters struct{}

func (f StandardFilters) Size(input interface{}) int {
	if input == nil {
		return 0
	}
	rv := reflect.ValueOf(input)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len()
	}
	return 0
}

func (f StandardFilters) Downcase(input interface{}) string {
	return strings.ToLower(engine.UtilsToString(input))
}

func (f StandardFilters) Upcase(input interface{}) string {
	return strings.ToUpper(engine.UtilsToString(input))
}

func (f StandardFilters) Capitalize(input interface{}) string {
	s := engine.UtilsToString(input)
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[0:1]) + strings.ToLower(s[1:])
}

func (f StandardFilters) Escape(input interface{}) engine.SafeHTML {
	return engine.SafeHTML(html.EscapeString(engine.UtilsToString(input)))
}

// Raw marks the input as safe HTML, bypassing AutoEscape when active.
// Use {{ var | raw }} to emit trusted HTML content.
func (f StandardFilters) Raw(input interface{}) engine.SafeHTML {
	return engine.SafeHTML(engine.UtilsToString(input))
}

func (f StandardFilters) Join(input interface{}, glue interface{}) string {
	rv := reflect.ValueOf(input)
	var s []string
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		s = make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			s = append(s, engine.UtilsToString(rv.Index(i).Interface()))
		}
	}
	return strings.Join(s, engine.UtilsToString(glue))
}

func (f StandardFilters) Strip(input interface{}) string {
	return strings.TrimSpace(engine.UtilsToString(input))
}

func (f StandardFilters) Split(input interface{}, delimiter interface{}) []string {
	return strings.Split(engine.UtilsToString(input), engine.UtilsToString(delimiter))
}

func (f StandardFilters) Replace(input interface{}, anchor interface{}, replacement interface{}) string {
	return strings.ReplaceAll(engine.UtilsToString(input), engine.UtilsToString(anchor), engine.UtilsToString(replacement))
}

func (f StandardFilters) Plus(input interface{}, operand interface{}) interface{} {
	return toFloat64(input) + toFloat64(operand)
}

func (f StandardFilters) Minus(input interface{}, operand interface{}) interface{} {
	return toFloat64(input) - toFloat64(operand)
}

func (f StandardFilters) Date(input interface{}, format interface{}) interface{} {
	t, ok := engine.UtilsToDate(input)
	if !ok {
		return input
	}
	fstr := engine.UtilsToString(format)
	if fstr == "" {
		return input
	}
	return t.Format(strftimeReplacer.Replace(fstr))
}

func (f StandardFilters) First(input interface{}) interface{} {
	rv := reflect.ValueOf(input)
	if (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
		return rv.Index(0).Interface()
	}
	return nil
}

func (f StandardFilters) Last(input interface{}) interface{} {
	rv := reflect.ValueOf(input)
	if (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
		return rv.Index(rv.Len() - 1).Interface()
	}
	return nil
}

func (f StandardFilters) Abs(input interface{}) interface{} {
	val := toFloat64(input)
	if val < 0 {
		return -val
	}
	return val
}

func (f StandardFilters) Times(input interface{}, operand interface{}) interface{} {
	return toFloat64(input) * toFloat64(operand)
}

func (f StandardFilters) DividedBy(input interface{}, operand interface{}) interface{} {
	op := toFloat64(operand)
	if op == 0 {
		return 0.0
	}
	return toFloat64(input) / op
}

func (f StandardFilters) Modulo(input interface{}, operand interface{}) interface{} {
	op := toFloat64(operand)
	if op == 0 {
		return 0.0
	}
	return math.Mod(toFloat64(input), op)
}

func (f StandardFilters) Round(input interface{}, n interface{}) interface{} {
	val := toFloat64(input)
	decimals := 0
	if n != nil {
		decimals, _ = engine.UtilsToInteger(n)
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(val*pow) / pow
}

func (f StandardFilters) Ceil(input interface{}) float64 {
	return math.Ceil(toFloat64(input))
}

func (f StandardFilters) Floor(input interface{}) float64 {
	return math.Floor(toFloat64(input))
}

func (f StandardFilters) Default(input interface{}, defaultValue interface{}) interface{} {
	if !engine.IsTruthy(input) {
		return defaultValue
	}
	return input
}

func (f StandardFilters) Append(input interface{}, suffix interface{}) string {
	return engine.UtilsToString(input) + engine.UtilsToString(suffix)
}

func (f StandardFilters) Prepend(input interface{}, prefix interface{}) string {
	return engine.UtilsToString(prefix) + engine.UtilsToString(input)
}

func (f StandardFilters) StripHtml(input interface{}) string {
	return stripHtmlRegex.ReplaceAllString(engine.UtilsToString(input), "")
}

func (f StandardFilters) Truncatewords(input interface{}, words interface{}, ellipsis interface{}) string {
	s := engine.UtilsToString(input)
	n := 15
	if words != nil {
		if val, err := engine.UtilsToInteger(words); err == nil {
			n = val
		}
	}
	el := "..."
	if ellipsis != nil {
		el = engine.UtilsToString(ellipsis)
	}
	parts := strings.Fields(s)
	if len(parts) <= n {
		return s
	}
	return strings.Join(parts[:n], " ") + el
}

func (f StandardFilters) Json(input interface{}) string {
	b, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return string(b)
}

func (f StandardFilters) Uniq(input interface{}) []interface{} {
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{input}
	}
	n := rv.Len()
	result := make([]interface{}, 0, n)
	seen := make(map[interface{}]struct{}, n)

	for i := 0; i < n; i++ {
		val := rv.Index(i).Interface()
		switch val.(type) {
		case string, int, int64, float64, bool:
			// fast path: directly comparable and hashable types
			if _, exists := seen[val]; !exists {
				seen[val] = struct{}{}
				result = append(result, val)
			}
		default:
			// Slow path: unhashable types (structs, maps, slices).
			// Linear scan with DeepEqual — avoids fmt.Sprintf alloc per element.
			// Liquid arrays are typically < 100 elements, so O(n²) is acceptable.
			duplicate := false
			for _, s := range result {
				if reflect.DeepEqual(val, s) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				result = append(result, val)
			}
		}
	}
	return result
}

func (f StandardFilters) Map(input interface{}, property interface{}) []interface{} {
	prop := engine.UtilsToString(property)
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	result := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result = append(result, getProperty(rv.Index(i).Interface(), prop))
	}
	return result
}

func (f StandardFilters) Where(input interface{}, property interface{}, targetValue ...interface{}) []interface{} {
	prop := engine.UtilsToString(property)
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}

	var target interface{}
	checkValue := len(targetValue) > 0
	if checkValue {
		target = targetValue[0]
	}

	result := make([]interface{}, 0)
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()
		val := getProperty(item, prop)
		if val == nil && !isMap(item) {
			continue
		}
		if checkValue {
			if engine.CompareValues(val, target) == 0 {
				result = append(result, item)
			}
		} else if engine.IsTruthy(val) {
			result = append(result, item)
		}
	}
	return result
}

func (f StandardFilters) ColorBrightness(input interface{}) int {
	s := engine.UtilsToString(input)
	if !strings.HasPrefix(s, "#") {
		return 0
	}
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return (r*299 + g*587 + b*114) / 1000
}

func (f StandardFilters) ColorLighten(input interface{}, percentage interface{}) string {
	s := engine.UtilsToString(input)
	if !strings.HasPrefix(s, "#") {
		return s
	}
	p := toFloat64(percentage) / 100.0
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	r = int(float64(r) + (255-float64(r))*p)
	g = int(float64(g) + (255-float64(g))*p)
	b = int(float64(b) + (255-float64(b))*p)
	return fmt.Sprintf("#%02X%02X%02X", clamp(r), clamp(g), clamp(b))
}

func (f StandardFilters) ColorDarken(input interface{}, percentage interface{}) string {
	s := engine.UtilsToString(input)
	if !strings.HasPrefix(s, "#") {
		return s
	}
	p := toFloat64(percentage) / 100.0
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	r = int(float64(r) * (1.0 - p))
	g = int(float64(g) * (1.0 - p))
	b = int(float64(b) * (1.0 - p))
	return fmt.Sprintf("#%02X%02X%02X", clamp(r), clamp(g), clamp(b))
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func (f StandardFilters) Sort(input interface{}, property ...interface{}) []interface{} {
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	n := rv.Len()
	res := make([]interface{}, n)
	for i := range res {
		res[i] = rv.Index(i).Interface()
	}
	if len(property) > 0 && property[0] != nil {
		prop := engine.UtilsToString(property[0])
		// Pre-compute keys once; sort an index slice to avoid a second full value alloc.
		keys := make([]interface{}, n)
		for i, v := range res {
			keys[i] = getProperty(v, prop)
		}
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		sort.SliceStable(indices, func(a, b int) bool {
			return engine.CompareValues(keys[indices[a]], keys[indices[b]]) < 0
		})
		sorted := make([]interface{}, n)
		for i, idx := range indices {
			sorted[i] = res[idx]
		}
		return sorted
	}
	sort.SliceStable(res, func(i, j int) bool {
		return engine.CompareValues(res[i], res[j]) < 0
	})
	return res
}

func (f StandardFilters) Reverse(input interface{}) []interface{} {
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	n := rv.Len()
	res := make([]interface{}, n)
	for i := 0; i < n; i++ {
		res[i] = rv.Index(n - 1 - i).Interface()
	}
	return res
}

func (f StandardFilters) Compact(input interface{}) []interface{} {
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	res := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		if v := rv.Index(i).Interface(); v != nil {
			res = append(res, v)
		}
	}
	return res
}

func (f StandardFilters) ReplaceFirst(input interface{}, anchor interface{}, replacement interface{}) string {
	return strings.Replace(engine.UtilsToString(input), engine.UtilsToString(anchor), engine.UtilsToString(replacement), 1)
}

func (f StandardFilters) Remove(input interface{}, anchor interface{}) string {
	return strings.ReplaceAll(engine.UtilsToString(input), engine.UtilsToString(anchor), "")
}

func (f StandardFilters) RemoveFirst(input interface{}, anchor interface{}) string {
	return strings.Replace(engine.UtilsToString(input), engine.UtilsToString(anchor), "", 1)
}

func (f StandardFilters) StripNewlines(input interface{}) string {
	s := engine.UtilsToString(input)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\r' {
			if i+1 < len(s) && s[i+1] == '\n' {
				i += 2 // skip \r\n as a unit
			} else {
				i++ // skip bare \r
			}
		} else if c == '\n' {
			i++ // skip \n
		} else {
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func (f StandardFilters) NewlineToBr(input interface{}) engine.SafeHTML {
	s := html.EscapeString(engine.UtilsToString(input))
	return engine.SafeHTML(strings.ReplaceAll(s, "\n", "<br />\n"))
}

func (f StandardFilters) Lstrip(input interface{}) string {
	return strings.TrimLeft(engine.UtilsToString(input), " \t\n\r")
}

func (f StandardFilters) Rstrip(input interface{}) string {
	return strings.TrimRight(engine.UtilsToString(input), " \t\n\r")
}

func (f StandardFilters) Truncate(input interface{}, length interface{}, ellipsis interface{}) string {
	s := engine.UtilsToString(input)
	n := 50
	if length != nil {
		if v, err := engine.UtilsToInteger(length); err == nil {
			n = v
		}
	}
	el := "..."
	if ellipsis != nil {
		el = engine.UtilsToString(ellipsis)
	}
	runes := []rune(s)
	elRunes := []rune(el)
	if len(runes) <= n {
		return s
	}
	cut := n - len(elRunes)
	if cut < 0 {
		cut = 0
	}
	return string(runes[:cut]) + el
}

func (f StandardFilters) Slice(input interface{}, offset interface{}, length ...interface{}) interface{} {
	s := engine.UtilsToString(input)
	runes := []rune(s)
	n := len(runes)

	off := 0
	if offset != nil {
		if v, err := engine.UtilsToInteger(offset); err == nil {
			off = v
		}
	}
	if off < 0 {
		off = n + off
	}
	if off < 0 {
		off = 0
	}
	if off >= n {
		return ""
	}

	cnt := 1
	if len(length) > 0 && length[0] != nil {
		if v, err := engine.UtilsToInteger(length[0]); err == nil {
			cnt = v
		}
	}
	end := off + cnt
	if end > n {
		end = n
	}
	_ = utf8.RuneCountInString
	return string(runes[off:end])
}

func (f StandardFilters) UrlEncode(input interface{}) string {
	return url.QueryEscape(engine.UtilsToString(input))
}

func (f StandardFilters) UrlDecode(input interface{}) string {
	s, err := url.QueryUnescape(engine.UtilsToString(input))
	if err != nil {
		return engine.UtilsToString(input)
	}
	return s
}

func (f StandardFilters) Base64Encode(input interface{}) string {
	return base64.StdEncoding.EncodeToString([]byte(engine.UtilsToString(input)))
}

func (f StandardFilters) Base64Decode(input interface{}) string {
	b, err := base64.StdEncoding.DecodeString(engine.UtilsToString(input))
	if err != nil {
		return ""
	}
	return string(b)
}

func (f StandardFilters) EscapeOnce(input interface{}) engine.SafeHTML {
	return engine.SafeHTML(html.EscapeString(html.UnescapeString(engine.UtilsToString(input))))
}

func (f StandardFilters) Concat(input interface{}, other interface{}) []interface{} {
	a := toSlice(input)
	b := toSlice(other)
	res := make([]interface{}, 0, len(a)+len(b))
	return append(append(res, a...), b...)
}

func (f StandardFilters) SortNatural(input interface{}, property ...interface{}) []interface{} {
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	n := rv.Len()

	type sortItem struct {
		val interface{}
		key string
	}
	items := make([]sortItem, n)
	for i := 0; i < n; i++ {
		v := rv.Index(i).Interface()
		var raw interface{}
		if len(property) > 0 && property[0] != nil {
			raw = getProperty(v, engine.UtilsToString(property[0]))
		} else {
			raw = v
		}
		items[i] = sortItem{val: v, key: strings.ToLower(engine.UtilsToString(raw))}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].key < items[j].key
	})

	res := make([]interface{}, n)
	for i, it := range items {
		res[i] = it.val
	}
	return res
}

// --- helpers ---

func getProperty(item interface{}, prop string) interface{} {
	if m, ok := item.(map[string]interface{}); ok {
		return m[prop]
	}
	rv := reflect.ValueOf(item)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName(prop); f.IsValid() {
			return f.Interface()
		}
	}
	return nil
}

func isMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

func toSlice(v interface{}) []interface{} {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	res := make([]interface{}, rv.Len())
	for i := range res {
		res[i] = rv.Index(i).Interface()
	}
	return res
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
		f, _ := engine.UtilsToNumber(v).(float64)
		return f
	}
	return 0
}

// valueToString reads a Value as string without going through interface{}.
// For KindString it returns the payload directly; other kinds fall back to UtilsToString.
func valueToString(v engine.Value) string {
	if v.Kind() == engine.KindString {
		return v.String()
	}
	return engine.UtilsToString(v.ToInterface())
}

// valueToFloat64 converts a Value to float64 for arithmetic filters.
func valueToFloat64(v engine.Value) float64 {
	switch v.Kind() {
	case engine.KindFloat:
		return v.Float()
	case engine.KindInt:
		return float64(v.Int())
	case engine.KindString:
		f, _ := engine.UtilsToNumber(v.String()).(float64)
		return f
	}
	return 0
}

func init() {
	// size — avoids reflect for the common string case
	engine.RegisterBuiltin("size", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		switch input.Kind() {
		case engine.KindString:
			return engine.ValueInt(int64(len(input.String())))
		case engine.KindObject:
			if obj := input.Object(); obj != nil {
				rv := reflect.ValueOf(obj)
				switch rv.Kind() {
				case reflect.Slice, reflect.Array, reflect.Map:
					return engine.ValueInt(int64(rv.Len()))
				}
			}
		}
		return engine.ValueInt(0)
	})

	// downcase / upcase / capitalize
	engine.RegisterBuiltin("downcase", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueString(strings.ToLower(valueToString(input)))
	})
	engine.RegisterBuiltin("upcase", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueString(strings.ToUpper(valueToString(input)))
	})
	engine.RegisterBuiltin("capitalize", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		s := valueToString(input)
		if len(s) == 0 {
			return engine.ValueString("")
		}
		return engine.ValueString(strings.ToUpper(s[0:1]) + strings.ToLower(s[1:]))
	})

	// escape / strip / strip_html
	engine.RegisterBuiltin("escape", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueObject(engine.SafeHTML(html.EscapeString(valueToString(input))))
	})
	engine.RegisterBuiltin("strip", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueString(strings.TrimSpace(valueToString(input)))
	})
	engine.RegisterBuiltin("strip_html", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueString(stripHtmlRegex.ReplaceAllString(valueToString(input), ""))
	})

	// append / prepend
	engine.RegisterBuiltin("append", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		suffix := ""
		if len(args) > 0 {
			suffix = engine.UtilsToString(args[0].ToInterface())
		}
		return engine.ValueString(valueToString(input) + suffix)
	})
	engine.RegisterBuiltin("prepend", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		prefix := ""
		if len(args) > 0 {
			prefix = engine.UtilsToString(args[0].ToInterface())
		}
		return engine.ValueString(prefix + valueToString(input))
	})

	// replace / replace_first
	engine.RegisterBuiltin("replace", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		s := valueToString(input)
		if len(args) < 2 {
			return engine.ValueString(s)
		}
		return engine.ValueString(strings.ReplaceAll(s, engine.UtilsToString(args[0].ToInterface()), engine.UtilsToString(args[1].ToInterface())))
	})
	engine.RegisterBuiltin("replace_first", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		s := valueToString(input)
		if len(args) < 2 {
			return engine.ValueString(s)
		}
		return engine.ValueString(strings.Replace(s, engine.UtilsToString(args[0].ToInterface()), engine.UtilsToString(args[1].ToInterface()), 1))
	})

	// split
	engine.RegisterBuiltin("split", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		delim := ""
		if len(args) > 0 {
			delim = engine.UtilsToString(args[0].ToInterface())
		}
		return engine.ValueObject(strings.Split(valueToString(input), delim))
	})

	// plus / minus / times / divided_by / modulo
	engine.RegisterBuiltin("plus", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		if len(args) == 0 {
			return engine.ValueFloat(valueToFloat64(input))
		}
		return engine.ValueFloat(valueToFloat64(input) + valueToFloat64(args[0]))
	})
	engine.RegisterBuiltin("minus", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		if len(args) == 0 {
			return engine.ValueFloat(valueToFloat64(input))
		}
		return engine.ValueFloat(valueToFloat64(input) - valueToFloat64(args[0]))
	})
	engine.RegisterBuiltin("times", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		if len(args) == 0 {
			return engine.ValueFloat(0)
		}
		return engine.ValueFloat(valueToFloat64(input) * valueToFloat64(args[0]))
	})
	engine.RegisterBuiltin("divided_by", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		if len(args) == 0 {
			return engine.ValueFloat(0)
		}
		op := valueToFloat64(args[0])
		if op == 0 {
			return engine.ValueFloat(0)
		}
		return engine.ValueFloat(valueToFloat64(input) / op)
	})
	engine.RegisterBuiltin("modulo", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		if len(args) == 0 {
			return engine.ValueFloat(0)
		}
		op := valueToFloat64(args[0])
		if op == 0 {
			return engine.ValueFloat(0)
		}
		return engine.ValueFloat(math.Mod(valueToFloat64(input), op))
	})

	// abs / ceil / floor / round
	engine.RegisterBuiltin("abs", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		v := valueToFloat64(input)
		if v < 0 {
			return engine.ValueFloat(-v)
		}
		return engine.ValueFloat(v)
	})
	engine.RegisterBuiltin("ceil", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueFloat(math.Ceil(valueToFloat64(input)))
	})
	engine.RegisterBuiltin("floor", func(_ *engine.Context, input engine.Value, _ []engine.Value) engine.Value {
		return engine.ValueFloat(math.Floor(valueToFloat64(input)))
	})
	engine.RegisterBuiltin("round", func(_ *engine.Context, input engine.Value, args []engine.Value) engine.Value {
		val := valueToFloat64(input)
		decimals := 0
		if len(args) > 0 && args[0].Kind() != engine.KindNil {
			if n, err := engine.UtilsToInteger(args[0].ToInterface()); err == nil {
				decimals = n
			}
		}
		pow := math.Pow(10, float64(decimals))
		return engine.ValueFloat(math.Round(val*pow) / pow)
	})
}
