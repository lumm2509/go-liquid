package engine

import (
	"fmt"
	"reflect"

	"github.com/go-liquid/internal/runtime"
)

// Context is the runtime execution state for a single render.
type Context struct {
	Environment        EnvironmentIface
	Environments       []map[string]interface{}
	StaticEnvironments []map[string]interface{}
	Scopes             []map[string]interface{}
	Registers          *runtime.Registers
	Errors             []error
	Warnings           []error
	ResourceLimits     *runtime.ResourceLimits
	ExceptionRenderer  ExceptionRenderer
	TemplateName       string
	Partial            bool
	StrictVariables    bool
	StrictFilters      bool
	GlobalFilter       func(interface{}) interface{}

	interrupts     []interface{}
	filters        []interface{}
	strainer       *Strainer
	baseScopeDepth int
}

func BuildContext(
	env EnvironmentIface,
	environments map[string]interface{},
	outerScope map[string]interface{},
	registers map[string]interface{},
	rethrowErrors bool,
	resourceLimits *runtime.ResourceLimits,
	staticEnvironments map[string]interface{},
) *Context {
	return NewContext(
		[]map[string]interface{}{environments},
		outerScope,
		registers,
		rethrowErrors,
		resourceLimits,
		[]map[string]interface{}{staticEnvironments},
		env,
	)
}

func NewContext(
	environments []map[string]interface{},
	outerScope map[string]interface{},
	registers map[string]interface{},
	rethrowErrors bool,
	resourceLimits *runtime.ResourceLimits,
	staticEnvironments []map[string]interface{},
	environment EnvironmentIface,
) *Context {
	if outerScope == nil {
		outerScope = make(map[string]interface{})
	}

	ctx := &Context{
		Environment:        environment,
		Environments:       environments,
		StaticEnvironments: staticEnvironments,
		Scopes:             []map[string]interface{}{outerScope},
		Registers:          runtime.NewRegisters(registers),
		Errors:             []error{},
		Warnings:           []error{},
		Partial:            false,
		StrictVariables:    false,
		baseScopeDepth:     0,
		interrupts:         []interface{}{},
		filters:            []interface{}{},
	}

	ctx.ResourceLimits = resourceLimits
	if ctx.ResourceLimits == nil && environment != nil {
		ctx.ResourceLimits = runtime.NewResourceLimits(environment.GetDefaultResourceLimits())
	}

	if environment != nil {
		ctx.Registers.SetStatic("file_system", environment.GetFileSystem())
		ctx.ExceptionRenderer = environment.GetExceptionRenderer()
	}
	if rethrowErrors {
		ctx.ExceptionRenderer = func(err error) error { return err }
	}

	ctx.squashInstanceAssignsWithEnvironments()
	return ctx
}

// --- RenderContext interface implementation ---

func (c *Context) Get(expression string) interface{} {
	expr, _ := ParseExpression(expression, NewStringScanner(expression), nil)
	return c.Evaluate(expr)
}

func (c *Context) Set(key string, value interface{}) {
	c.Scopes[0][key] = value
}

func (c *Context) RegisterGet(key string) interface{} { return c.Registers.Get(key) }
func (c *Context) RegisterSet(key string, value interface{}) { c.Registers.Set(key, value) }
func (c *Context) IsPartial() bool                          { return c.Partial }
func (c *Context) SetPartial(partial bool)                   { c.Partial = partial }
func (c *Context) GetTemplateName() string                   { return c.TemplateName }
func (c *Context) SetTemplateName(name string)               { c.TemplateName = name }

func (c *Context) Strainer() *Strainer {
	if c.strainer == nil {
		c.strainer = c.Environment.CreateStrainer(c, c.filters)
	}
	return c.strainer
}

func (c *Context) InvokeFilter(method string, obj interface{}, args ...interface{}) interface{} {
	allArgs := append([]interface{}{obj}, args...)
	return c.Strainer().Invoke(method, allArgs...)
}

func (c *Context) invoke(method string, obj interface{}, args ...interface{}) interface{} {
	return c.InvokeFilter(method, obj, args...)
}

func (c *Context) ApplyGlobalFilter(obj interface{}) interface{} {
	if c.GlobalFilter == nil {
		return obj
	}
	return c.GlobalFilter(obj)
}

func (c *Context) Evaluate(object interface{}) interface{} {
	if evaluatable, ok := object.(interface{ Evaluate(*Context) interface{} }); ok {
		return evaluatable.Evaluate(c)
	}
	return object
}

func (c *Context) FindVariable(key string, raiseOnNotFound bool) interface{} {
	var variable interface{}
	found := false

	for _, scope := range c.Scopes {
		if _, ok := scope[key]; ok {
			val, err := c.lookupAndEvaluate(scope, key, raiseOnNotFound)
			if err != nil {
				c.Errors = append(c.Errors, err)
				return nil
			}
			variable = val
			found = true
			break
		}
	}

	if !found {
		val, ok, err := c.tryVariableFindInEnvironments(key, raiseOnNotFound)
		if err != nil {
			c.Errors = append(c.Errors, err)
			return nil
		}
		if ok {
			variable = val
		}
	}

	return c.toLiquid(variable)
}

func (c *Context) lookupAndEvaluate(obj map[string]interface{}, key string, raiseOnNotFound bool) (interface{}, error) {
	value, exists := obj[key]
	if c.StrictVariables && raiseOnNotFound && !exists {
		return nil, fmt.Errorf("undefined variable %s", key)
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Func {
		if rv.Type().NumIn() == 0 {
			results := rv.Call(nil)
			value = results[0].Interface()
		} else {
			results := rv.Call([]reflect.Value{reflect.ValueOf(c)})
			value = results[0].Interface()
		}
	}

	return value, nil
}

func (c *Context) HandleError(err error, lineNumber int) string {
	c.Errors = append(c.Errors, err)
	return c.ExceptionRenderer(err).Error()
}

func (c *Context) Interrupt() bool {
	return len(c.interrupts) > 0
}

func (c *Context) PushInterrupt(i interface{}) {
	c.interrupts = append(c.interrupts, i)
}

func (c *Context) PopInterrupt() interface{} {
	if len(c.interrupts) == 0 {
		return nil
	}
	idx := len(c.interrupts) - 1
	i := c.interrupts[idx]
	c.interrupts = c.interrupts[:idx]
	return i
}

func (c *Context) Push(newScope map[string]interface{}) {
	if newScope == nil {
		newScope = make(map[string]interface{})
	}
	c.Scopes = append([]map[string]interface{}{newScope}, c.Scopes...)
	c.checkOverflow()
}

func (c *Context) Pop() (map[string]interface{}, error) {
	if len(c.Scopes) <= 1 {
		return nil, fmt.Errorf("ContextError: stack underflow")
	}
	popped := c.Scopes[0]
	c.Scopes = c.Scopes[1:]
	return popped, nil
}

func (c *Context) Stack(newScope map[string]interface{}, block func() error) error {
	c.Push(newScope)
	defer c.Pop()
	return block()
}

func (c *Context) NewIsolatedSubcontext() RenderContext {
	c.checkOverflow()
	sub := NewContext(
		c.Environments,
		make(map[string]interface{}),
		c.Registers.Static(),
		false,
		c.ResourceLimits,
		c.StaticEnvironments,
		c.Environment,
	)
	sub.baseScopeDepth = c.baseScopeDepth + 1
	sub.ExceptionRenderer = c.ExceptionRenderer
	sub.filters = c.filters
	sub.Errors = make([]error, 0)
	sub.Warnings = make([]error, 0)
	return sub
}

func (c *Context) MergeSubcontext(sub RenderContext) {
	if s, ok := sub.(*Context); ok {
		c.Errors = append(c.Errors, s.Errors...)
		c.Warnings = append(c.Warnings, s.Warnings...)
	}
}

func (c *Context) toLiquid(obj interface{}) interface{} {
	if obj == nil {
		return nil
	}
	if l, ok := obj.(interface{ ToLiquid() interface{} }); ok {
		res := l.ToLiquid()
		if d, ok := res.(interface{ SetContext(*Context) }); ok {
			d.SetContext(c)
		}
		return res
	}
	return obj
}

func (c *Context) checkOverflow() {
	if c.baseScopeDepth+len(c.Scopes) > 100 {
		if c.Environment != nil {
			if logger := c.Environment.GetLogger(); logger != nil {
				logger.Log(DebugEvent{
					Event: "context.overflow",
					Data:  map[string]interface{}{"depth": c.baseScopeDepth + len(c.Scopes)},
				})
			}
		}
		c.HandleError(fmt.Errorf("StackLevelError: Nesting too deep"), 0)
	}
}

func (c *Context) tryVariableFindInEnvironments(key string, raiseOnNotFound bool) (interface{}, bool, error) {
	for _, env := range c.Environments {
		val, err := c.lookupAndEvaluate(env, key, raiseOnNotFound)
		if err != nil {
			return nil, false, err
		}
		if val != nil {
			return val, true, nil
		}
	}
	for _, env := range c.StaticEnvironments {
		val, err := c.lookupAndEvaluate(env, key, raiseOnNotFound)
		if err != nil {
			return nil, false, err
		}
		if val != nil {
			return val, true, nil
		}
	}
	return nil, false, nil
}

func (c *Context) squashInstanceAssignsWithEnvironments() {
	lastScope := c.Scopes[len(c.Scopes)-1]
	for k := range lastScope {
		for _, env := range c.Environments {
			if _, ok := env[k]; ok {
				val, _ := c.lookupAndEvaluate(env, k, false)
				lastScope[k] = val
				break
			}
		}
	}
}
