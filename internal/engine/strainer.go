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

// FilterRegistry holds the filter configuration for an environment.
// It is built once per Environment and shared across all renders.
type FilterRegistry struct {
	Filters    []interface{}
	filterMaps []map[string]*filterMethod

	mu       sync.RWMutex
	combined map[string]*filterMethod
}

func NewFilterRegistry() *FilterRegistry {
	return &FilterRegistry{
		Filters:    []interface{}{},
		filterMaps: []map[string]*filterMethod{},
	}
}

func (fr *FilterRegistry) AddFilter(filter interface{}) {
	fr.Filters = append(fr.Filters, filter)
	fr.filterMaps = append(fr.filterMaps, buildMethodMapForFilter(filter))
	fr.mu.Lock()
	fr.combined = nil
	fr.mu.Unlock()
}

func (fr *FilterRegistry) getCombined() map[string]*filterMethod {
	fr.mu.RLock()
	c := fr.combined
	fr.mu.RUnlock()
	if c != nil {
		return c
	}

	fr.mu.Lock()
	defer fr.mu.Unlock()
	if fr.combined != nil { // another goroutine may have built while we waited
		return fr.combined
	}
	size := 0
	for _, m := range fr.filterMaps {
		size += len(m)
	}
	combined := make(map[string]*filterMethod, size)
	for _, m := range fr.filterMaps {
		for k, v := range m {
			combined[k] = v
		}
	}
	fr.combined = combined
	return combined
}

func (fr *FilterRegistry) NewFilterDispatcher(context *Context) *FilterDispatcher {
	return &FilterDispatcher{context: context, methodMap: fr.getCombined()}
}

func (fr *FilterRegistry) Clone() *FilterRegistry {
	newFr := NewFilterRegistry()
	newFr.Filters = append(newFr.Filters, fr.Filters...)
	newFr.filterMaps = append(newFr.filterMaps, fr.filterMaps...)
	return newFr
}

func (fr *FilterRegistry) FilterMethodNames() []string {
	m := fr.getCombined()
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}

// callArgsPool reuses []reflect.Value slices across filter invocations to reduce heap pressure.
var callArgsPool = sync.Pool{
	New: func() interface{} {
		s := make([]reflect.Value, 0, 8)
		return &s
	},
}

type FilterDispatcher struct {
	context   *Context
	methodMap map[string]*filterMethod
}

func NewFilterDispatcher(context *Context) *FilterDispatcher {
	return &FilterDispatcher{context: context, methodMap: make(map[string]*filterMethod)}
}

func (s *FilterDispatcher) Invoke(method string, args ...interface{}) interface{} {
	// fast path: built-in filters via direct dispatch, no reflect
	if fn, ok := BuiltinFilters[method]; ok {
		sp := GetValueSlice()
		vals := *sp
		for _, a := range args {
			vals = append(vals, ValueFrom(a))
		}
		var input Value
		var filterArgs []Value
		if len(vals) > 0 {
			input = vals[0]
			filterArgs = vals[1:]
		}
		result := fn(s.context, input, filterArgs).ToInterface()
		*sp = vals[:0]
		PutValueSlice(sp)
		return result
	}

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

		sp := callArgsPool.Get().(*[]reflect.Value)
		var callArgs []reflect.Value
		if cap(*sp) >= targetNumArgs {
			callArgs = (*sp)[:targetNumArgs]
		} else {
			callArgs = make([]reflect.Value, targetNumArgs)
		}
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

		// zero entries before returning to pool — avoid retaining references
		for i := range callArgs {
			callArgs[i] = reflect.Value{}
		}
		*sp = callArgs[:0]
		callArgsPool.Put(sp)

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
					Event: EventFilterNotFound,
					Data:  map[string]interface{}{"filter": method},
				})
			}
		}
		return args[0]
	}
	return nil
}
