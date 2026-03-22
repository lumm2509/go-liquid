package engine

import "sync"

// FilterFunc is the signature for a built-in filter with the Value type.
// Using Value instead of interface{} eliminates boxing for primitive inputs.
type FilterFunc func(ctx *Context, input Value, args []Value) Value

// BuiltinFilters is the direct-dispatch table for standard filters.
// Populated via RegisterBuiltin from filters/standard_filters.go init().
var BuiltinFilters = map[string]FilterFunc{}

// RegisterBuiltin must be called from init() before any render occurs
func RegisterBuiltin(name string, fn FilterFunc) {
	BuiltinFilters[name] = fn
}

// valueSlicePool reuses []Value slices for built-in filter arg conversion,
// eliminating one heap allocation per built-in filter call.
var valueSlicePool = sync.Pool{
	New: func() interface{} {
		s := make([]Value, 0, 8)
		return &s
	},
}

// GetValueSlice returns a pooled []Value; caller MUST call PutValueSlice when done
func GetValueSlice() *[]Value {
	return valueSlicePool.Get().(*[]Value)
}

// PutValueSlice resets and returns a []Value slice to the pool.
func PutValueSlice(s *[]Value) {
	*s = (*s)[:0]
	valueSlicePool.Put(s)
}
