package engine

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/go-liquid/internal/runtime"
)

var invokeFilterArgsPool = sync.Pool{
	New: func() interface{} {
		s := make([]interface{}, 0, 8)
		return &s
	},
}

const maxInlineScopes = 8

// ScopeStack is a stack of variable scopes. The first maxInlineScopes entries
// live in an inline array to avoid heap allocations for common templates.
type ScopeStack struct {
	inline [maxInlineScopes]map[string]interface{}
	depth  int
	extra  []map[string]interface{} // only used when depth > maxInlineScopes
}

func newScopeStack(initial map[string]interface{}) ScopeStack {
	var s ScopeStack
	s.inline[0] = initial
	s.depth = 1
	return s
}

func (s *ScopeStack) Push(scope map[string]interface{}) {
	if s.depth < maxInlineScopes {
		s.inline[s.depth] = scope
	} else {
		s.extra = append(s.extra, scope)
	}
	s.depth++
}

func (s *ScopeStack) Pop() map[string]interface{} {
	if s.depth == 0 {
		return nil
	}
	s.depth--
	if s.depth < maxInlineScopes {
		scope := s.inline[s.depth]
		s.inline[s.depth] = nil
		return scope
	}
	n := len(s.extra)
	scope := s.extra[n-1]
	s.extra = s.extra[:n-1]
	return scope
}

func (s *ScopeStack) Top() map[string]interface{} {
	if s.depth == 0 {
		return nil
	}
	if s.depth <= maxInlineScopes {
		return s.inline[s.depth-1]
	}
	return s.extra[len(s.extra)-1]
}

func (s *ScopeStack) Bottom() map[string]interface{} {
	if s.depth == 0 {
		return nil
	}
	return s.inline[0]
}

func (s *ScopeStack) At(i int) map[string]interface{} {
	if i < maxInlineScopes {
		return s.inline[i]
	}
	return s.extra[i-maxInlineScopes]
}

func (s *ScopeStack) Len() int { return s.depth }

// Context is the runtime execution state for a single render.
type Context struct {
	Environment        EnvironmentIface
	Environments       []map[string]interface{}
	StaticEnvironments []map[string]interface{}
	Scopes             ScopeStack
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
	AutoEscape         bool
	GoCtx              context.Context

	interrupts          []interface{}
	filterDispatcher    *FilterDispatcher
	baseScopeDepth      int
	runtimeExprCache    map[string]interface{} // lazy; caches Get() expression parses
	// Forloop is set by the for-tag to enable O(1) forloop variable access.
	// nil outside of a for-loop body.
	Forloop *ForloopDrop
}

// ContextConfig holds all parameters for constructing a render Context.
// Use NewContext(ContextConfig{...}) instead of positional arguments.
type ContextConfig struct {
	Environments       []map[string]interface{}
	OuterScope         map[string]interface{}
	Registers          map[string]interface{}
	RethrowErrors      bool
	ResourceLimits     *runtime.ResourceLimits
	StaticEnvironments []map[string]interface{}
	Environment        EnvironmentIface
}


func NewContext(cfg ContextConfig) *Context {
	environments := cfg.Environments
	outerScope := cfg.OuterScope
	registers := cfg.Registers
	rethrowErrors := cfg.RethrowErrors
	resourceLimits := cfg.ResourceLimits
	staticEnvironments := cfg.StaticEnvironments
	environment := cfg.Environment
	if outerScope == nil {
		outerScope = make(map[string]interface{})
	}

	ctx := &Context{
		Environment:        environment,
		Environments:       environments,
		StaticEnvironments: staticEnvironments,
		Scopes:             newScopeStack(outerScope),
		Registers:          runtime.NewRegisters(registers),
		Partial:         false,
		StrictVariables: false,
		baseScopeDepth:  0,
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

func (c *Context) Get(expression string) interface{} {
	if c.runtimeExprCache == nil {
		c.runtimeExprCache = make(map[string]interface{}, 8)
	}
	expr, ok := c.runtimeExprCache[expression]
	if !ok {
		expr, _ = ParseExpression(expression, NewStringScanner(expression), c.runtimeExprCache)
	}
	return c.Evaluate(expr)
}

func (c *Context) Set(key string, value interface{}) {
	c.Scopes.Top()[key] = value
}

// Context returns the Go context, or context.Background() if none was set
func (c *Context) Context() context.Context {
	if c.GoCtx != nil {
		return c.GoCtx
	}
	return context.Background()
}

func (c *Context) RegisterGet(key string) interface{} { return c.Registers.Get(key) }
func (c *Context) RegisterSet(key string, value interface{}) { c.Registers.Set(key, value) }
func (c *Context) IsPartial() bool                          { return c.Partial }
func (c *Context) SetPartial(partial bool)                   { c.Partial = partial }
func (c *Context) GetTemplateName() string                   { return c.TemplateName }
func (c *Context) SetTemplateName(name string)               { c.TemplateName = name }

func (c *Context) FilterDispatcher() *FilterDispatcher {
	if c.filterDispatcher == nil {
		c.filterDispatcher = c.Environment.CreateFilterDispatcher(c, nil)
	}
	return c.filterDispatcher
}

func (c *Context) InvokeFilter(method string, obj interface{}, args ...interface{}) interface{} {
	pooled := invokeFilterArgsPool.Get().(*[]interface{})
	allArgs := append((*pooled)[:0], obj)
	allArgs = append(allArgs, args...)
	result := c.FilterDispatcher().Invoke(method, allArgs...)
	*pooled = allArgs[:0]
	invokeFilterArgsPool.Put(pooled)
	return result
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

	for i := c.Scopes.Len() - 1; i >= 0; i-- {
		scope := c.Scopes.At(i)
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
	if value == nil {
		return nil, nil
	}

	// fast path: common primitives are never functions — skip reflect
	switch value.(type) {
	case string, int, int64, float64, bool, []interface{}, map[string]interface{}:
		return value, nil
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

func (c *Context) Push(newScope map[string]interface{}) error {
	if newScope == nil {
		newScope = make(map[string]interface{})
	}
	c.Scopes.Push(newScope)
	return c.checkOverflow()
}

func (c *Context) Pop() (map[string]interface{}, error) {
	if c.Scopes.Len() <= 1 {
		return nil, fmt.Errorf("ContextError: stack underflow")
	}
	popped := c.Scopes.Pop()
	return popped, nil
}

func (c *Context) Stack(newScope map[string]interface{}, block func() error) error {
	if err := c.Push(newScope); err != nil {
		return err
	}
	defer c.Pop()
	return block()
}

func (c *Context) NewIsolatedSubcontext() RenderContext {
	c.checkOverflow()
	sub := NewContext(ContextConfig{
		Environments:       c.Environments,
		OuterScope:         make(map[string]interface{}),
		Registers:          c.Registers.Static(),
		RethrowErrors:      false,
		ResourceLimits:     c.ResourceLimits,
		StaticEnvironments: c.StaticEnvironments,
		Environment:        c.Environment,
	})
	sub.baseScopeDepth = c.baseScopeDepth + 1
	sub.ExceptionRenderer = c.ExceptionRenderer
	sub.GoCtx = c.GoCtx
	sub.filterDispatcher = nil
	// errors/warnings/interrupts start nil — append/len are nil-safe
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
		return l.ToLiquid()
	}
	return obj
}

func (c *Context) checkOverflow() error {
	if c.baseScopeDepth+c.Scopes.Len() > 100 {
		if c.Environment != nil {
			if logger := c.Environment.GetLogger(); logger != nil {
				logger.Log(DebugEvent{
					Event: EventContextOverflow,
					Data:  map[string]interface{}{"depth": c.baseScopeDepth + c.Scopes.Len()},
				})
			}
		}
		err := fmt.Errorf("StackLevelError: Nesting too deep")
		c.HandleError(err, 0)
		return err
	}
	return nil
}

func (c *Context) tryVariableFindInEnvironments(key string, raiseOnNotFound bool) (interface{}, bool, error) {
	for _, env := range c.Environments {
		if _, exists := env[key]; exists {
			val, err := c.lookupAndEvaluate(env, key, raiseOnNotFound)
			if err != nil {
				return nil, false, err
			}
			return val, true, nil
		}
	}
	for _, env := range c.StaticEnvironments {
		if _, exists := env[key]; exists {
			val, err := c.lookupAndEvaluate(env, key, raiseOnNotFound)
			if err != nil {
				return nil, false, err
			}
			return val, true, nil
		}
	}
	return nil, false, nil
}

func (c *Context) squashInstanceAssignsWithEnvironments() {
	lastScope := c.Scopes.Bottom()
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
