package engine

// Value kinds — all primitive kinds are stored inline without heap allocation.
const (
	KindNil    uint8 = iota
	KindBool         // ival: 0=false, 1=true
	KindInt          // ival
	KindFloat        // fval
	KindString       // sval
	KindObject       // pval — map, slice, struct; lives on the heap
)

// Value is the universal Liquid value type.
// Primitives (nil, bool, int, float, string) live inline without heap allocation.
// Complex objects (maps, slices, structs) are stored via pval.
type Value struct {
	kind uint8
	ival int64
	fval float64
	sval string
	pval interface{}
}

// Kind returns the KindXxx constant for this value.
func (v Value) Kind() uint8 { return v.kind }

// Constructors

func ValueNil() Value                 { return Value{kind: KindNil} }
func ValueBool(b bool) Value          { v := Value{kind: KindBool}; if b { v.ival = 1 }; return v }
func ValueInt(i int64) Value          { return Value{kind: KindInt, ival: i} }
func ValueFloat(f float64) Value      { return Value{kind: KindFloat, fval: f} }
func ValueString(s string) Value      { return Value{kind: KindString, sval: s} }
func ValueObject(o interface{}) Value { return Value{kind: KindObject, pval: o} }

// Accessors

func (v Value) IsNil() bool { return v.kind == KindNil }

func (v Value) IsTruthy() bool {
	switch v.kind {
	case KindNil:
		return false
	case KindBool:
		return v.ival != 0
	case KindInt:
		return v.ival != 0
	case KindFloat:
		return v.fval != 0
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
func (v Value) Int() int64 { return v.ival }

// Float returns the float64 payload; only valid when Kind() == KindFloat.
func (v Value) Float() float64 { return v.fval }

// Bool returns the bool payload; only valid when Kind() == KindBool.
func (v Value) Bool() bool { return v.ival != 0 }

// Object returns the object payload; only valid when Kind() == KindObject.
func (v Value) Object() interface{} { return v.pval }

// ToInterface converts the Value to interface{} for interoperability with
// legacy interface{}-based code during the migration to the Value type.
func (v Value) ToInterface() interface{} {
	switch v.kind {
	case KindNil:
		return nil
	case KindBool:
		return v.ival != 0
	case KindInt:
		return int(v.ival)
	case KindFloat:
		return v.fval
	case KindString:
		return v.sval
	case KindObject:
		return v.pval
	}
	return nil
}

// ValueFrom converts an interface{} to a Value.
// This is the bridge from the legacy interface{} world to the Value world.
func ValueFrom(obj interface{}) Value {
	if obj == nil {
		return ValueNil()
	}
	switch v := obj.(type) {
	case bool:
		return ValueBool(v)
	case int:
		return ValueInt(int64(v))
	case int64:
		return ValueInt(v)
	case float64:
		return ValueFloat(v)
	case string:
		return ValueString(v)
	default:
		return ValueObject(v)
	}
}
