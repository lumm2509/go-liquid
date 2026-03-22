package liquid

// Drop is the interface for objects that expose custom methods to Liquid templates.
// use ToLiquid to control the value presented to the engine; override InvokeDrop
// to dispatch named property accesses and call LiquidMethodMissing for unhandled ones.
type Drop interface {
	ToLiquid() interface{}
	LiquidMethodMissing(method string) interface{}
	InvokeDrop(method string) interface{}
}

// DropBase is the base Drop implementation; embed it and override InvokeDrop to expose methods.
type DropBase struct{}

func (d *DropBase) ToLiquid() interface{} {
	return d
}

// LiquidMethodMissing is called when no concrete implementation handles the property
func (d *DropBase) LiquidMethodMissing(method string) interface{} {
	return nil
}

// InvokeDrop is called for every property access; override this to dispatch specific methods
func (d *DropBase) InvokeDrop(method string) interface{} {
	return d.LiquidMethodMissing(method)
}

// ToS returns a string representation; override in concrete types
func (d *DropBase) ToS() string {
	return "Drop"
}
