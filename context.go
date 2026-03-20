package liquid

import (
	"fmt"
	"reflect"
)

// Context representa el estado de ejecución y resolución de variables.
type Context struct {
	Environment        *Environment
	Environments       []map[string]interface{}
	StaticEnvironments []map[string]interface{}
	Scopes             []map[string]interface{}
	Registers          *Registers
	Errors             []error
	Warnings           []error
	ResourceLimits     *ResourceLimits
	ExceptionRenderer  ExceptionRenderer
	TemplateName       string
	Partial            bool
	StrictVariables    bool
	StrictFilters      bool
	GlobalFilter       func(interface{}) interface{}

	// Internos
	interrupts     []interface{}
	filters        []interface{}
	strainer       *Strainer
	baseScopeDepth int
}

// BuildContext equivale a Context.build
func BuildContext(
	env *Environment,
	environments map[string]interface{},
	outerScope map[string]interface{},
	registers map[string]interface{},
	rethrowErrors bool,
	resourceLimits *ResourceLimits,
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

// NewContext equivale a initialize
func NewContext(
	environments []map[string]interface{},
	outerScope map[string]interface{},
	registers map[string]interface{},
	rethrowErrors bool,
	resourceLimits *ResourceLimits,
	staticEnvironments []map[string]interface{},
	environment *Environment,
) *Context {
	if environment == nil {
		environment = DefaultEnvironment()
	}
	if outerScope == nil {
		outerScope = make(map[string]interface{})
	}

	ctx := &Context{
		Environment:        environment,
		Environments:       environments,
		StaticEnvironments: staticEnvironments,
		Scopes:             []map[string]interface{}{outerScope},
		Registers:          NewRegisters(registers),
		Errors:             []error{},
		Warnings:           []error{},
		Partial:            false,
		StrictVariables:    false,
		ResourceLimits:     resourceLimits,
		baseScopeDepth: 0,
		interrupts:     []interface{}{},
		filters:        []interface{}{},
	}

	if ctx.ResourceLimits == nil {
		ctx.ResourceLimits = NewResourceLimits(environment.DefaultResourceLimits)
	}

	// Configuración de registros estáticos obligatorios
	ctx.Registers.SetStatic("cached_partials", make(map[string]interface{}))
	ctx.Registers.SetStatic("file_system", environment.FileSystem)
	ctx.Registers.SetStatic("template_factory", NewTemplateFactory())

	ctx.ExceptionRenderer = environment.ExceptionRenderer
	if rethrowErrors {
		ctx.ExceptionRenderer = func(err error) error { return err }
	}

	ctx.squashInstanceAssignsWithEnvironments()
	return ctx
}

// --- Gestión de Scopes (Stack) ---

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

// --- Resolución de Variables ---

func (c *Context) Get(expression string) interface{} {
	// En Ruby: evaluate(Expression.parse(expression, @string_scanner))
	expr, _ := ParseExpression(expression, NewStringScanner(""), nil)
	return c.Evaluate(expr)
}

func (c *Context) Set(key string, value interface{}) {
	c.Scopes[0][key] = value
}

func (c *Context) Strainer() *Strainer {
	if c.strainer == nil {
		c.strainer = c.Environment.CreateStrainer(c, c.filters)
	}
	return c.strainer
}

func (c *Context) invoke(method string, obj interface{}, args ...interface{}) interface{} {
	allArgs := append([]interface{}{obj}, args...)
	return c.Strainer().Invoke(method, allArgs...)
}

func (c *Context) ApplyGlobalFilter(obj interface{}) interface{} {
	if c.GlobalFilter == nil {
		return obj
	}
	return c.GlobalFilter(obj)
}

func (c *Context) Evaluate(object interface{}) interface{} {
	// Si el objeto implementa una interfaz Evaluatable (como Variable)
	if evaluatable, ok := object.(interface{ Evaluate(*Context) interface{} }); ok {
		return evaluatable.Evaluate(c)
	}
	return object
}

func (c *Context) FindVariable(key string, raiseOnNotFound bool) interface{} {
	var variable interface{}
	found := false

	// Buscar en los scopes (de arriba hacia abajo)
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

	// Si no se encuentra, buscar en environments
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

	// Simular .to_liquid y asignación de contexto si el objeto es un Drop
	return c.toLiquid(variable)
}

func (c *Context) lookupAndEvaluate(obj map[string]interface{}, key string, raiseOnNotFound bool) (interface{}, error) {
	value, exists := obj[key]
	if c.StrictVariables && raiseOnNotFound && !exists {
		return nil, fmt.Errorf("undefined variable %s", key)
	}

	// Manejo de Procs (funciones en Go)
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Func {
		// Simular aridad de Ruby: si acepta Context o no
		if rv.Type().NumIn() == 0 {
			results := rv.Call(nil)
			value = results[0].Interface()
		} else {
			results := rv.Call([]reflect.Value{reflect.ValueOf(c)})
			value = results[0].Interface()
		}
		obj[key] = value // Cachear resultado
	}

	return value, nil
}

// --- Manejo de Errores e Interrupts ---

func (c *Context) HandleError(err error, lineNumber int) string {
	// Aquí se integraría la lógica de Liquid::Error
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

// --- Helpers internos ---

func (c *Context) toLiquid(obj interface{}) interface{} {
	if obj == nil {
		return nil
	}
	// Si el objeto tiene un método ToLiquid() (Interfaz Liquidizable)
	if l, ok := obj.(interface{ ToLiquid() interface{} }); ok {
		res := l.ToLiquid()
		// Asignar contexto si es un Drop
		if d, ok := res.(interface{ SetContext(*Context) }); ok {
			d.SetContext(c)
		}
		return res
	}
	return obj
}

func (c *Context) NewIsolatedSubcontext() *Context {
	c.checkOverflow()

	sub := NewContext(
		c.Environments,
		make(map[string]interface{}),
		c.Registers.static,
		false,
		c.ResourceLimits,
		c.StaticEnvironments,
		c.Environment,
	)
	sub.baseScopeDepth = c.baseScopeDepth + 1
	sub.ExceptionRenderer = c.ExceptionRenderer
	sub.filters = c.filters
	// Own error/warning slices — not shared with parent to avoid aliasing bugs.
	// Caller merges sub.Errors into parent after the subrender completes.
	sub.Errors = make([]error, 0)
	sub.Warnings = make([]error, 0)
	return sub
}

// MergeSubcontext copies errors and warnings from a completed subcontext back
// into this context. Call after NewIsolatedSubcontext render is done.
func (c *Context) MergeSubcontext(sub *Context) {
	c.Errors = append(c.Errors, sub.Errors...)
	c.Warnings = append(c.Warnings, sub.Warnings...)
}

func (c *Context) checkOverflow() {
	if c.baseScopeDepth+len(c.Scopes) > 100 {
		if c.Environment != nil && c.Environment.Logger != nil {
			c.Environment.Logger.Log(DebugEvent{
				Event: "context.overflow",
				Data:  map[string]interface{}{"depth": c.baseScopeDepth + len(c.Scopes)},
			})
		}
		c.HandleError(fmt.Errorf("StackLevelError: Nesting too deep"), 0)
		return
	}
}

func (c *Context) tryVariableFindInEnvironments(key string, raiseOnNotFound bool) (interface{}, bool, error) {
	allEnvs := make([]map[string]interface{}, len(c.Environments)+len(c.StaticEnvironments))
	copy(allEnvs, c.Environments)
	copy(allEnvs[len(c.Environments):], c.StaticEnvironments)
	for _, env := range allEnvs {
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
