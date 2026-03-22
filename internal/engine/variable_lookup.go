package engine

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// structFieldCache maps reflect.Type → map[string]int (field name → field index).
// Built lazily on first access to a struct type; shared across all renders.
var structFieldCache sync.Map

// cachedFieldIndex returns the index of the exported field with the given name
// (or its lowercase-first alias) in struct type rt.
// The map is built once per type and stored in structFieldCache.
func cachedFieldIndex(rt reflect.Type, name string) (int, bool) {
	if v, ok := structFieldCache.Load(rt); ok {
		idx, found := v.(map[string]int)[name]
		return idx, found
	}
	m := make(map[string]int, rt.NumField()*2)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		m[f.Name] = i
		// Liquid templates use lowercase keys; store alias so "index" finds "Index".
		if len(f.Name) > 0 {
			lower := strings.ToLower(f.Name[:1]) + f.Name[1:]
			if lower != f.Name {
				m[lower] = i
			}
		}
	}
	structFieldCache.Store(rt, m)
	idx, found := m[name]
	return idx, found
}

type VariableLookup struct {
	Name         interface{}
	Lookups      []interface{}
	CommandFlags int
}

var CommandMethods = []string{"size", "first", "last"}

func NewVariableLookup(markup string, ss *StringScanner, cache map[string]interface{}) *VariableLookup {
	tokens := parseVariableMarkup(markup)
	if len(tokens) == 0 {
		return &VariableLookup{}
	}

	nameToken := tokens[0]
	var name interface{}
	if strings.HasPrefix(nameToken, "[") && strings.HasSuffix(nameToken, "]") {
		name, _ = ParseExpression(nameToken[1:len(nameToken)-1], ss, cache)
	} else {
		name = nameToken
	}

	lookups := make([]interface{}, 0)
	commandFlags := 0

	for i, token := range tokens[1:] {
		if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
			expr, _ := ParseExpression(token[1:len(token)-1], ss, cache)
			lookups = append(lookups, expr)
		} else {
			lookups = append(lookups, token)
			for _, cm := range CommandMethods {
				if token == cm {
					commandFlags |= 1 << i
					break
				}
			}
		}
	}

	return &VariableLookup{
		Name:         name,
		Lookups:      lookups,
		CommandFlags: commandFlags,
	}
}

func parseVariableMarkup(markup string) []string {
	var res []string
	var current strings.Builder
	bracketDepth := 0
	for i := 0; i < len(markup); i++ {
		c := markup[i]
		switch c {
		case '[':
			bracketDepth++
			current.WriteByte(c)
		case ']':
			bracketDepth--
			current.WriteByte(c)
		case '.':
			if bracketDepth == 0 {
				if current.Len() > 0 {
					res = append(res, strings.TrimSpace(current.String()))
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		res = append(res, strings.TrimSpace(current.String()))
	}
	return res
}

func (vl *VariableLookup) LookupCommand(index int) bool {
	return (vl.CommandFlags & (1 << index)) != 0
}

func (vl *VariableLookup) Evaluate(ctx *Context) interface{} {
	name := ctx.Evaluate(vl.Name)
	nameStr, ok := name.(string)
	if !ok {
		if ctx.StrictVariables {
			return fmt.Errorf("variable name must be a string, got %T", name)
		}
		return nil
	}

	// D8: forloop fast path — ctx.Forloop is non-nil only inside the for-tag body
	var obj interface{}
	if nameStr == "forloop" && ctx.Forloop != nil {
		obj = ctx.Forloop
	} else {
		obj = ctx.FindVariable(nameStr, true)
	}

	for i, lookupExpr := range vl.Lookups {
		key := ctx.Evaluate(lookupExpr)
		key = ToLiquidValue(key)

		obj = vl.accessProperty(ctx, obj, key, i)
		if obj == nil && ctx.StrictVariables {
			return fmt.Errorf("undefined variable %v", key)
		}
		if obj == nil {
			return nil
		}
	}

	return obj
}

func (vl *VariableLookup) accessProperty(ctx *Context, obj interface{}, key interface{}, index int) interface{} {
	if obj == nil {
		return nil
	}

	// fast path: map[string]interface{} is the dominant case — avoid reflect
	if m, ok := obj.(map[string]interface{}); ok {
		if keyStr, ok := key.(string); ok {
			if !vl.LookupCommand(index) {
				v, _ := ctx.lookupAndEvaluate(m, keyStr, false)
				return v
			}
		}
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Map {
		kv := reflect.ValueOf(key)
		val := rv.MapIndex(kv)
		if val.IsValid() {
			return val.Interface()
		}
		if keyStr, ok := key.(string); ok {
			if m, ok := obj.(map[string]interface{}); ok {
				v, _ := ctx.lookupAndEvaluate(m, keyStr, false)
				return v
			}
		}
	}

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		if idx, err := UtilsToInteger(key); err == nil {
			if idx >= 0 && idx < rv.Len() {
				return rv.Index(idx).Interface()
			}
		}
	}

	if vl.LookupCommand(index) {
		if keyStr, ok := key.(string); ok {
			switch keyStr {
			case "size":
				if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array || rv.Kind() == reflect.Map || rv.Kind() == reflect.String {
					return rv.Len()
				}
			case "first":
				if (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
					return rv.Index(0).Interface()
				}
				if rv.Kind() == reflect.String {
					s := rv.String()
					runes := []rune(s)
					if len(runes) > 0 {
						return string(runes[0])
					}
					return ""
				}
			case "last":
				if (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
					return rv.Index(rv.Len() - 1).Interface()
				}
				if rv.Kind() == reflect.String {
					s := rv.String()
					runes := []rune(s)
					if len(runes) > 0 {
						return string(runes[len(runes)-1])
					}
					return ""
				}
			}
		}
	}

	if rv.Kind() == reflect.Struct {
		if keyStr, ok := key.(string); ok {
			// D2: O(1) cached field index instead of linear FieldByName scan
			if idx, found := cachedFieldIndex(rv.Type(), keyStr); found {
				field := rv.Field(idx)
				if field.CanInterface() {
					return field.Interface()
				}
			}
			method := rv.MethodByName(keyStr)
			if method.IsValid() && method.Kind() == reflect.Func {
				methodType := method.Type()
				if methodType.NumIn() == 0 {
					results := method.Call(nil)
					if len(results) > 0 {
						return results[0].Interface()
					}
				} else if methodType.NumIn() == 1 && methodType.In(0) == reflect.TypeOf((*Context)(nil)) {
					results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
					if len(results) > 0 {
						return results[0].Interface()
					}
				}
			}
		}
	}

	return nil
}

func ToLiquidValue(obj interface{}) interface{} {
	if obj == nil {
		return nil
	}
	if l, ok := obj.(interface{ ToLiquidValue() interface{} }); ok {
		return l.ToLiquidValue()
	}
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Map {
		newMap := make(map[string]interface{})
		for _, key := range rv.MapKeys() {
			k := fmt.Sprint(key.Interface())
			newMap[k] = ToLiquidValue(rv.MapIndex(key).Interface())
		}
		return newMap
	}
	return obj
}
