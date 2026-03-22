package engine

import (
	"math"
	"unsafe"
)

// Value kinds — all primitive kinds are stored inline without heap allocation.
const (
	KindNil    uint8 = iota
	KindBool         // num: 0=false, 1=true
	KindInt          // num: uint64(int64)
	KindFloat        // num: math.Float64bits
	KindString       // sval
	KindObject       // pval — map, slice, struct; lives on the heap
)

// Value is the universal Liquid value type.
// Primitives (nil, bool, int, float, string) live inline without heap allocation.
// ival and fval are mutually exclusive — collapsed into a single uint64 field (num)
// to save 8 bytes per Value: 48 bytes vs 56 bytes in the previous layout.
type Value struct {
	kind uint8
	_    [7]byte    // explicit alignment padding — documents the intent
	num  uint64     // KindBool: 0/1; KindInt: uint64(int64); KindFloat: math.Float64bits
	sval string
	pval interface{}
}

// Compile-time size assertion: Value must be exactly 48 bytes.
var _ [48]byte = [unsafe.Sizeof(Value{})]byte{}

// Kind returns the KindXxx constant for this value.
func (v Value) Kind() uint8 { return v.kind }

// Constructors

func ValueNil() Value   { return Value{kind: KindNil} }
func ValueBool(b bool) Value {
	if b {
		return Value{kind: KindBool, num: 1}
	}
	return Value{kind: KindBool}
}
func ValueInt(i int64) Value     { return Value{kind: KindInt, num: uint64(i)} }
func ValueFloat(f float64) Value { return Value{kind: KindFloat, num: math.Float64bits(f)} }
func ValueString(s string) Value { return Value{kind: KindString, sval: s} }
func ValueObject(o interface{}) Value { return Value{kind: KindObject, pval: o} }

// Accessors

func (v Value) IsNil() bool { return v.kind == KindNil }

func (v Value) IsTruthy() bool {
	switch v.kind {
	case KindNil:
		return false
	case KindBool:
		return v.num != 0
	case KindInt:
		return v.num != 0
	case KindFloat:
		return v.num != 0 // math.Float64bits(0.0) == 0
	case KindString:
		return v.sval != ""
	case KindObject:
		return v.pval != nil
	}
	return false
}

// String returns the string payload; only valid when Kind() == KindString.
func (v Value) String() string { return v.sval }

// Int returns the int64 payload; only valid when Kind() == KindInt.
func (v Value) Int() int64 { return int64(v.num) }

// Float returns the float64 payload; only valid when Kind() == KindFloat.
func (v Value) Float() float64 { return math.Float64frombits(v.num) }

// Bool returns the bool payload; only valid when Kind() == KindBool.
func (v Value) Bool() bool { return v.num != 0 }

// Object returns the object payload; only valid when Kind() == KindObject.
func (v Value) Object() interface{} { return v.pval }

// ToInterface converts the Value to interface{} for interoperability with
// legacy interface{}-based code during the migration to the Value type.
func (v Value) ToInterface() interface{} {
	switch v.kind {
	case KindNil:
		return nil
	case KindBool:
		return v.num != 0
	case KindInt:
		return int(v.num)
	case KindFloat:
		return math.Float64frombits(v.num)
	case KindString:
		return v.sval
	case KindObject:
		return v.pval
	}
	return nil
}

// ValueFrom converts an interface{} to a Value.
// Inlined fast path for string (the dominant case in Liquid filter chains).
// nil and numeric types go through valueFromTyped.
func ValueFrom(obj interface{}) Value {
	if s, ok := obj.(string); ok {
		return Value{kind: KindString, sval: s}
	}
	return valueFromTyped(obj)
}

// valueFromTyped handles nil and non-string types.
// Kept separate so ValueFrom can be inlined by the compiler.
func valueFromTyped(obj interface{}) Value {
	if obj == nil {
		return Value{}
	}
	switch v := obj.(type) {
	case bool:
		if v {
			return Value{kind: KindBool, num: 1}
		}
		return Value{kind: KindBool}
	case int:
		return Value{kind: KindInt, num: uint64(v)}
	case int64:
		return Value{kind: KindInt, num: uint64(v)}
	case float64:
		return Value{kind: KindFloat, num: math.Float64bits(v)}
	default:
		return Value{kind: KindObject, pval: obj}
	}
}
