package liquid

// Drop is the interface for objects that expose custom methods to Liquid templates.
// Implement ToLiquid to control the value this object presents to the template
// engine. Override InvokeDrop to dispatch named property accesses; call
// LiquidMethodMissing for any method your type does not handle explicitly.
//
// Drop implementations are safe for concurrent use across multiple renders as
// long as they do not mutate shared state during InvokeDrop or LiquidMethodMissing.
type Drop interface {
	ToLiquid() interface{}
	LiquidMethodMissing(method string) interface{}
	InvokeDrop(method string) interface{}
}

// DropBase is the base implementation for Drop. Embed it in your own struct
// and override InvokeDrop to expose methods to Liquid templates.
type DropBase struct{}

func (d *DropBase) ToLiquid() interface{} {
	return d
}

// LiquidMethodMissing is called when a property is accessed that has no
// concrete implementation. Override InvokeDrop to handle named methods
// before falling through to this default.
func (d *DropBase) LiquidMethodMissing(method string) interface{} {
	return nil
}

// InvokeDrop is called for every property access on this Drop. Override this
// in your concrete type to dispatch specific methods; call LiquidMethodMissing
// for unknown ones.
func (d *DropBase) InvokeDrop(method string) interface{} {
	return d.LiquidMethodMissing(method)
}

// ToS returns a string representation. Override in concrete types.
func (d *DropBase) ToS() string {
	return "Drop"
}
