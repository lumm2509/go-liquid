package liquid

import (
	"fmt"
	"reflect"
	"strings"
)

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

	obj := ctx.FindVariable(nameStr, true)

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

		if d, ok := obj.(interface{ SetContext(*Context) }); ok {
			d.SetContext(ctx)
		}
	}

	return obj
}

func (vl *VariableLookup) accessProperty(ctx *Context, obj interface{}, key interface{}, index int) interface{} {
	if obj == nil {
		return nil
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	// 1. Try maps
	if rv.Kind() == reflect.Map {
		kv := reflect.ValueOf(key)
		// Try to coerce key to map key type if possible
		val := rv.MapIndex(kv)
		if val.IsValid() {
			return val.Interface()
		}
		// If key is string, try to find in map[string]interface{}
		if keyStr, ok := key.(string); ok {
			if m, ok := obj.(map[string]interface{}); ok {
				v, _ := ctx.lookupAndEvaluate(m, keyStr, false)
				return v
			}
		}
	}

	// 2. Try slices/arrays
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		if idx, err := UtilsToInteger(key); err == nil {
			if idx >= 0 && idx < rv.Len() {
				return rv.Index(idx).Interface()
			}
		}
	}

	// 3. Handle commands
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
					if len(s) > 0 {
						// Manejar UTF-8 correctamente: obtener el primer rune
						runes := []rune(s)
						if len(runes) > 0 {
							return string(runes[0])
						}
					}
					return ""
				}
			case "last":
				if (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) && rv.Len() > 0 {
					return rv.Index(rv.Len() - 1).Interface()
				}
				if rv.Kind() == reflect.String {
					s := rv.String()
					if len(s) > 0 {
						// Manejar UTF-8 correctamente: obtener el último rune
						runes := []rune(s)
						if len(runes) > 0 {
							return string(runes[len(runes)-1])
						}
					}
					return ""
				}
			}
		}
	}

	// 4. Try struct fields and methods (Drops support)
	if rv.Kind() == reflect.Struct {
		if keyStr, ok := key.(string); ok {
			// Primero intentar campo
			field := rv.FieldByName(keyStr)
			if field.IsValid() && field.CanInterface() {
				return field.Interface()
			}

			// Luego intentar método (como respond_to? y send en Ruby)
			method := rv.MethodByName(keyStr)
			if method.IsValid() && method.Kind() == reflect.Func {
				// Verificar si el método acepta Context como parámetro
				methodType := method.Type()
				if methodType.NumIn() == 0 {
					// Método sin parámetros
					results := method.Call(nil)
					if len(results) > 0 {
						return results[0].Interface()
					}
				} else if methodType.NumIn() == 1 && methodType.In(0) == reflect.TypeOf((*Context)(nil)).Elem() {
					// Método que acepta Context
					results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
					if len(results) > 0 {
						return results[0].Interface()
					}
				}
			}
		}
	}

	// 6. Special handling for color strings (e.g., #FFFFFF)
	if s, ok := obj.(string); ok && strings.HasPrefix(s, "#") {
		if keyStr, ok := key.(string); ok {
			var r, g, b int
			fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
			switch keyStr {
			case "red":
				return r
			case "green":
				return g
			case "blue":
				return b
			case "rgb":
				return fmt.Sprintf("%d, %d, %d", r, g, b)
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

	// Si es un mapa, convertir sus valores también
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
