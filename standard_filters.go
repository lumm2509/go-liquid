package liquid

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
)

var stripHtmlRegex = regexp.MustCompile(`<[^>]*>`)

// strftimeReplacer convierte directivas strftime al formato de time.Format de Go.
// Un único paso sin doble-sustitución.
var strftimeReplacer = strings.NewReplacer(
	"%Y", "2006",
	"%y", "06",
	"%m", "01",
	"%d", "02",
	"%H", "15",
	"%M", "04",
	"%S", "05",
	"%B", "January",
	"%b", "Jan",
	"%A", "Monday",
	"%a", "Mon",
	"%e", "2",
	"%j", "002",
	"%p", "PM",
	"%Z", "MST",
	"%z", "-0700",
	"%I", "03",
)

type StandardFilters struct{}

func (f StandardFilters) Size(input interface{}) int {
	if input == nil {
		return 0
	}

	rv := reflect.ValueOf(input)

	// Manejar strings
	if rv.Kind() == reflect.String {
		return rv.Len()
	}

	// Manejar slices y arrays de cualquier tipo usando reflexión
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len()
	}

	// Manejar maps
	if rv.Kind() == reflect.Map {
		return rv.Len()
	}

	return 0
}

func (f StandardFilters) Downcase(input interface{}) string {
	return strings.ToLower(UtilsToString(input))
}

func (f StandardFilters) Upcase(input interface{}) string {
	return strings.ToUpper(UtilsToString(input))
}

func (f StandardFilters) Capitalize(input interface{}) string {
	s := UtilsToString(input)
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[0:1]) + strings.ToLower(s[1:])
}

func (f StandardFilters) Escape(input interface{}) string {
	return html.EscapeString(UtilsToString(input))
}

func (f StandardFilters) Join(input interface{}, glue interface{}) string {
	s := []string{}
	rv := reflect.ValueOf(input)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			s = append(s, UtilsToString(rv.Index(i).Interface()))
		}
	}
	return strings.Join(s, UtilsToString(glue))
}

func (f StandardFilters) Strip(input interface{}) string {
	return strings.TrimSpace(UtilsToString(input))
}

func (f StandardFilters) Split(input interface{}, delimiter interface{}) []string {
	return strings.Split(UtilsToString(input), UtilsToString(delimiter))
}

func (f StandardFilters) Replace(input interface{}, anchor interface{}, replacement interface{}) string {
	return strings.ReplaceAll(UtilsToString(input), UtilsToString(anchor), UtilsToString(replacement))
}

func (f StandardFilters) Plus(input interface{}, operand interface{}) interface{} {
	a := toFloat64(input)
	b := toFloat64(operand)
	return a + b
}

func (f StandardFilters) Minus(input interface{}, operand interface{}) interface{} {
	a := toFloat64(input)
	b := toFloat64(operand)
	return a - b
}

func (f StandardFilters) Date(input interface{}, format interface{}) interface{} {
	t := UtilsToDate(input)
	if t == nil {
		return input
	}
	fstr := UtilsToString(format)
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
		// Liquid behavior: divided by 0 returns 0 or potentially infinity/NaN but for safety in Go let's return 0 or the input?
		// Ruby liquid raises, but here we want to avoid panic. Returning 0 is a safer default for web rendering.
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
		decimals, _ = UtilsToInteger(n)
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
	if !IsTruthy(input) {
		return defaultValue
	}
	return input
}

func (f StandardFilters) Append(input interface{}, suffix interface{}) string {
	return UtilsToString(input) + UtilsToString(suffix)
}

func (f StandardFilters) Prepend(input interface{}, prefix interface{}) string {
	return UtilsToString(prefix) + UtilsToString(input)
}

func (f StandardFilters) StripHtml(input interface{}) string {
	return stripHtmlRegex.ReplaceAllString(UtilsToString(input), "")
}

func (f StandardFilters) Truncatewords(input interface{}, words interface{}, ellipsis interface{}) string {
	s := UtilsToString(input)
	n := 15
	if words != nil {
		if val, err := UtilsToInteger(words); err == nil {
			n = val
		}
	}
	el := "..."
	if ellipsis != nil {
		el = UtilsToString(ellipsis)
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

	result := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		val := rv.Index(i).Interface()
		duplicate := false
		for _, seen := range result {
			if CompareValues(val, seen) == 0 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, val)
		}
	}
	return result
}

func (f StandardFilters) Map(input interface{}, property interface{}) []interface{} {
	prop := UtilsToString(property)
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}

	result := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()

		// Intentar como map
		if m, ok := item.(map[string]interface{}); ok {
			if val, exists := m[prop]; exists {
				result = append(result, val)
			} else {
				result = append(result, nil)
			}
			continue
		}

		// Intentar como struct usando reflexión
		itemRv := reflect.ValueOf(item)
		if itemRv.Kind() == reflect.Ptr {
			itemRv = itemRv.Elem()
		}
		if itemRv.Kind() == reflect.Struct {
			field := itemRv.FieldByName(prop)
			if field.IsValid() && field.CanInterface() {
				result = append(result, field.Interface())
				continue
			}
			// Intentar método
			method := itemRv.MethodByName(prop)
			if method.IsValid() && method.Kind() == reflect.Func && method.Type().NumIn() == 0 {
				results := method.Call(nil)
				if len(results) > 0 {
					result = append(result, results[0].Interface())
					continue
				}
			}
		}

		result = append(result, nil)
	}
	return result
}

func (f StandardFilters) Where(input interface{}, property interface{}, targetValue ...interface{}) []interface{} {
	prop := UtilsToString(property)
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}

	var target interface{}
	checkValue := false
	if len(targetValue) > 0 {
		target = targetValue[0]
		checkValue = true
	}

	result := make([]interface{}, 0)
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()
		var val interface{}

		// Intentar como map
		if m, ok := item.(map[string]interface{}); ok {
			val = m[prop]
		} else {
			// Intentar como struct usando reflexión
			itemRv := reflect.ValueOf(item)
			if itemRv.Kind() == reflect.Ptr {
				itemRv = itemRv.Elem()
			}
			if itemRv.Kind() == reflect.Struct {
				field := itemRv.FieldByName(prop)
				if field.IsValid() && field.CanInterface() {
					val = field.Interface()
				} else {
					// Intentar método
					method := itemRv.MethodByName(prop)
					if method.IsValid() && method.Kind() == reflect.Func && method.Type().NumIn() == 0 {
						results := method.Call(nil)
						if len(results) > 0 {
							val = results[0].Interface()
						}
					}
				}
			}

			if val == nil {
				continue
			}
		}

		if checkValue {
			if CompareValues(val, target) == 0 {
				result = append(result, item)
			}
		} else {
			if IsTruthy(val) {
				result = append(result, item)
			}
		}
	}
	return result
}

func (f StandardFilters) ColorBrightness(input interface{}) int {
	s := UtilsToString(input)
	if !strings.HasPrefix(s, "#") {
		return 0
	}
	var r, g, b int
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return (r*299 + g*587 + b*114) / 1000
}

func (f StandardFilters) ColorLighten(input interface{}, percentage interface{}) string {
	s := UtilsToString(input)
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
	s := UtilsToString(input)
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
	res := make([]interface{}, rv.Len())
	for i := range res {
		res[i] = rv.Index(i).Interface()
	}
	if len(property) > 0 && property[0] != nil {
		prop := UtilsToString(property[0])
		sort.SliceStable(res, func(i, j int) bool {
			vi := getProperty(res[i], prop)
			vj := getProperty(res[j], prop)
			return CompareValues(vi, vj) < 0
		})
	} else {
		sort.SliceStable(res, func(i, j int) bool {
			return CompareValues(res[i], res[j]) < 0
		})
	}
	return res
}

// getProperty extrae una propiedad de un map o struct por nombre.
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

// --- Filtros adicionales del spec de Shopify Liquid ---

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
		v := rv.Index(i).Interface()
		if v != nil {
			res = append(res, v)
		}
	}
	return res
}

func (f StandardFilters) ReplaceFirst(input interface{}, anchor interface{}, replacement interface{}) string {
	return strings.Replace(UtilsToString(input), UtilsToString(anchor), UtilsToString(replacement), 1)
}

func (f StandardFilters) Remove(input interface{}, anchor interface{}) string {
	return strings.ReplaceAll(UtilsToString(input), UtilsToString(anchor), "")
}

func (f StandardFilters) RemoveFirst(input interface{}, anchor interface{}) string {
	return strings.Replace(UtilsToString(input), UtilsToString(anchor), "", 1)
}

func (f StandardFilters) StripNewlines(input interface{}) string {
	s := UtilsToString(input)
	s = strings.ReplaceAll(s, "\r\n", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func (f StandardFilters) NewlineToBr(input interface{}) string {
	return strings.ReplaceAll(UtilsToString(input), "\n", "<br />\n")
}

func (f StandardFilters) Lstrip(input interface{}) string {
	return strings.TrimLeft(UtilsToString(input), " \t\n\r")
}

func (f StandardFilters) Rstrip(input interface{}) string {
	return strings.TrimRight(UtilsToString(input), " \t\n\r")
}

func (f StandardFilters) Truncate(input interface{}, length interface{}, ellipsis interface{}) string {
	s := UtilsToString(input)
	n := 50
	if length != nil {
		if v, err := UtilsToInteger(length); err == nil {
			n = v
		}
	}
	el := "..."
	if ellipsis != nil {
		el = UtilsToString(ellipsis)
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

// Slice extracts a substring or sub-array.
// For strings: slice(offset) or slice(offset, length).
// Negative offset counts from the end.
func (f StandardFilters) Slice(input interface{}, offset interface{}, length ...interface{}) interface{} {
	s := UtilsToString(input)
	runes := []rune(s)
	n := len(runes)

	off := 0
	if offset != nil {
		if v, err := UtilsToInteger(offset); err == nil {
			off = v
		}
	}
	// Negative offset
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
		if v, err := UtilsToInteger(length[0]); err == nil {
			cnt = v
		}
	}
	end := off + cnt
	if end > n {
		end = n
	}
	_ = utf8.RuneCountInString // ensure import used
	return string(runes[off:end])
}

func (f StandardFilters) UrlEncode(input interface{}) string {
	return url.QueryEscape(UtilsToString(input))
}

func (f StandardFilters) UrlDecode(input interface{}) string {
	s, err := url.QueryUnescape(UtilsToString(input))
	if err != nil {
		return UtilsToString(input)
	}
	return s
}

func (f StandardFilters) Base64Encode(input interface{}) string {
	return base64.StdEncoding.EncodeToString([]byte(UtilsToString(input)))
}

func (f StandardFilters) Base64Decode(input interface{}) string {
	b, err := base64.StdEncoding.DecodeString(UtilsToString(input))
	if err != nil {
		return ""
	}
	return string(b)
}

func (f StandardFilters) EscapeOnce(input interface{}) string {
	// Decodifica primero entidades ya codificadas y luego vuelve a escapar,
	// evitando doble-codificación. html.UnescapeString + EscapeString.
	return html.EscapeString(html.UnescapeString(UtilsToString(input)))
}

func (f StandardFilters) StripNewlinesAlias(input interface{}) string {
	return f.StripNewlines(input)
}

// Concat concatena dos arrays.
func (f StandardFilters) Concat(input interface{}, other interface{}) []interface{} {
	a := toSlice(input)
	b := toSlice(other)
	res := make([]interface{}, 0, len(a)+len(b))
	res = append(res, a...)
	res = append(res, b...)
	return res
}

func toSlice(v interface{}) []interface{} {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	res := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		res[i] = rv.Index(i).Interface()
	}
	return res
}

// SortNatural sorts case-insensitively.
func (f StandardFilters) SortNatural(input interface{}, property ...interface{}) []interface{} {
	rv := reflect.ValueOf(input)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []interface{}{}
	}
	res := make([]interface{}, rv.Len())
	for i := range res {
		res[i] = rv.Index(i).Interface()
	}
	if len(property) > 0 && property[0] != nil {
		prop := UtilsToString(property[0])
		sort.SliceStable(res, func(i, j int) bool {
			vi := strings.ToLower(fmt.Sprintf("%v", getProperty(res[i], prop)))
			vj := strings.ToLower(fmt.Sprintf("%v", getProperty(res[j], prop)))
			return vi < vj
		})
	} else {
		sort.SliceStable(res, func(i, j int) bool {
			return strings.ToLower(fmt.Sprintf("%v", res[i])) < strings.ToLower(fmt.Sprintf("%v", res[j]))
		})
	}
	return res
}
