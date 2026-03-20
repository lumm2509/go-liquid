package liquid

import "fmt"

type Drop interface {
	ToLiquid() interface{}
	LiquidMethodMissing(method string) interface{}
	InvokeDrop(method string) interface{}
	SetContext(ctx *Context)
}

// DropBase is the base implementation for Drop. Embed it in your own struct
// and override InvokeDrop to expose methods to Liquid templates.
//
// Concurrency note: a DropBase stores a *Context via SetContext. A Drop
// instance must not be shared across concurrent renders; create a new Drop
// per render or avoid storing context-dependent state in the Drop.
type DropBase struct {
	Context *Context
}

func (d *DropBase) ToLiquid() interface{} {
	return d
}

// LiquidMethodMissing is called when a property is accessed that has no
// concrete implementation. Override InvokeDrop to handle named methods
// before falling through to this default.
func (d *DropBase) LiquidMethodMissing(method string) interface{} {
	if d.Context != nil && d.Context.StrictVariables {
		return fmt.Errorf("undefined method %s", method)
	}
	return nil
}

// InvokeDrop is called for every property access on this Drop. Override this
// in your concrete type to dispatch specific methods; call LiquidMethodMissing
// for unknown ones.
func (d *DropBase) InvokeDrop(method string) interface{} {
	return d.LiquidMethodMissing(method)
}

func (d *DropBase) SetContext(ctx *Context) {
	d.Context = ctx
}

// ToS returns a string representation. Override in concrete types.
func (d *DropBase) ToS() string {
	return "Drop"
}
