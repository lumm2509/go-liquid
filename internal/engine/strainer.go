package engine

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

type filterMethod struct {
	fn         reflect.Value
	numIn      int
	isVariadic bool
	paramTypes []reflect.Type
}

var filterMethodCache sync.Map // map[reflect.Type]map[string]*filterMethod

func buildMethodMapForFilter(filter interface{}) map[string]*filterMethod {
	t := reflect.TypeOf(filter)
	if cached, ok := filterMethodCache.Load(t); ok {
		return cached.(map[string]*filterMethod)
	}
	m := make(map[string]*filterMethod)
	val := reflect.ValueOf(filter)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		fn := val.Method(i)
		mt := fn.Type()
		numIn := mt.NumIn()
		paramTypes := make([]reflect.Type, numIn)
		for j := 0; j < numIn; j++ {
			paramTypes[j] = mt.In(j)
		}
		fm := &filterMethod{
			fn:         fn,
			numIn:      numIn,
			isVariadic: mt.IsVariadic(),
			paramTypes: paramTypes,
		}
		goName := method.Name
		snakeName := pascalToSnake(goName)
		m[goName] = fm
		if snakeName != goName {
			m[snakeName] = fm
		}
	}
	filterMethodCache.Store(t, m)
	return m
}

func pascalToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// StrainerTemplate holds the filter configuration for an environment.
type StrainerTemplate struct {
	Filters    []interface{}
	filterMaps []map[string]*filterMethod

	mu       sync.Mutex
	combined map[string]*filterMethod
}

func NewStrainerTemplate() *StrainerTemplate {
	return &StrainerTemplate{
		Filters:    []interface{}{},
		filterMaps: []map[string]*filterMethod{},
	}
}

func (st *StrainerTemplate) AddFilter(filter interface{}) {
	st.Filters = append(st.Filters, filter)
	st.filterMaps = append(st.filterMaps, buildMethodMapForFilter(filter))
	st.mu.Lock()
	st.combined = nil
	st.mu.Unlock()
}

func (st *StrainerTemplate) getCombined() map[string]*filterMethod {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.combined != nil {
		return st.combined
	}
	size := 0
	for _, m := range st.filterMaps {
		size += len(m)
	}
	combined := make(map[string]*filterMethod, size)
	for _, m := range st.filterMaps {
		for k, v := range m {
			combined[k] = v
		}
	}
	st.combined = combined
	return combined
}

func (st *StrainerTemplate) NewStrainer(context *Context) *Strainer {
	return &Strainer{context: context, methodMap: st.getCombined()}
}

func (st *StrainerTemplate) Clone() *StrainerTemplate {
	newSt := NewStrainerTemplate()
	newSt.Filters = append(newSt.Filters, st.Filters...)
	newSt.filterMaps = append(newSt.filterMaps, st.filterMaps...)
	return newSt
}

func (st *StrainerTemplate) FilterMethodNames() []string {
	m := st.getCombined()
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}

// Strainer dispatches filter calls for a single render context.
type Strainer struct {
	context   *Context
	methodMap map[string]*filterMethod
}

func NewStrainer(context *Context) *Strainer {
	return &Strainer{context: context, methodMap: make(map[string]*filterMethod)}
}

func (s *Strainer) Invoke(method string, args ...interface{}) interface{} {
	fm, ok := s.methodMap[method]
	if !ok {
		fm, ok = s.methodMap[toPascalCase(method)]
	}

	if ok {
		numIn := fm.numIn
		isVariadic := fm.isVariadic

		targetNumArgs := numIn
		if isVariadic {
			if len(args) > numIn-1 {
				targetNumArgs = len(args)
			} else {
				targetNumArgs = numIn - 1
			}
		}

		callArgs := make([]reflect.Value, targetNumArgs)
		for i := 0; i < targetNumArgs; i++ {
			var targetType reflect.Type
			if isVariadic && i >= numIn-1 {
				targetType = fm.paramTypes[numIn-1].Elem()
			} else {
				targetType = fm.paramTypes[i]
			}
			if i < len(args) {
				arg := args[i]
				if arg == nil {
					callArgs[i] = reflect.Zero(targetType)
				} else {
					v := reflect.ValueOf(arg)
					if v.Type().ConvertibleTo(targetType) {
						callArgs[i] = v.Convert(targetType)
					} else {
						callArgs[i] = reflect.Zero(targetType)
					}
				}
			} else {
				callArgs[i] = reflect.Zero(targetType)
			}
		}

		res := fm.fn.Call(callArgs)
		if len(res) > 0 {
			return res[0].Interface()
		}
		return nil
	}

	if len(args) > 0 {
		if s.context.StrictFilters {
			return fmt.Errorf("Liquid error: Filter '%s' not found", method)
		}
		env := s.context.Environment
		if env != nil {
			if logger := env.GetLogger(); logger != nil {
				logger.Log(DebugEvent{
					Event: "filter.not_found",
					Data:  map[string]interface{}{"filter": method},
				})
			}
		}
		return args[0]
	}
	return nil
}
