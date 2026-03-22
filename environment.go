package liquid

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-liquid/internal/engine"
	"github.com/go-liquid/internal/filters"
	"github.com/go-liquid/internal/tags"
)

// Environment holds all global configuration for a Liquid template engine.
type Environment struct {
	ErrorMode             string
	ExceptionRenderer     ExceptionRenderer
	FileSystem            FileSystem
	DefaultResourceLimits map[string]interface{}
	Logger                DebugLogger // nil = no-op

	tags                       map[string]TagFactory
	strainerTemplate           *StrainerTemplate
	strainerTemplateClassCache map[string]*StrainerTemplate
	mu                         sync.RWMutex
	frozen                     bool
}

func (e *Environment) log(event string, data map[string]interface{}) {
	if e.Logger == nil {
		return
	}
	e.Logger.Log(DebugEvent{Event: event, Data: data})
}

var (
	defaultEnv     *Environment
	defaultEnvOnce sync.Once
)

func DefaultEnvironment() *Environment {
	defaultEnvOnce.Do(func() { defaultEnv = NewEnvironment() })
	return defaultEnv
}

func NewEnvironment() *Environment {
	env := &Environment{
		ErrorMode:                  "lax",
		tags:                       make(map[string]TagFactory),
		ExceptionRenderer:          func(err error) error { return err },
		FileSystem:                 &BlankFileSystem{},
		DefaultResourceLimits:      make(map[string]interface{}),
		strainerTemplateClassCache: make(map[string]*StrainerTemplate),
	}
	for k, v := range tags.StandardTags {
		env.tags[k] = v
	}
	env.strainerTemplate = NewStrainerTemplate()
	env.strainerTemplate.AddFilter(filters.StandardFilters{})
	return env
}

func BuildEnvironment(fn func(*Environment)) *Environment {
	env := NewEnvironment()
	if fn != nil {
		fn(env)
	}
	env.Freeze()
	return env
}

func (e *Environment) RegisterTag(name string, factory TagFactory) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.frozen {
		if e.Logger != nil {
			e.Logger.Log(DebugEvent{
				Event: "environment.frozen_tag_skipped",
				Data:  map[string]interface{}{"tag": name},
			})
		}
		return fmt.Errorf("can't modify frozen environment, skipping tag %s", name)
	}
	e.tags[name] = factory
	return nil
}

func (e *Environment) RegisterFilter(filter interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strainerTemplateClassCache = make(map[string]*StrainerTemplate)
	e.strainerTemplate.AddFilter(filter)
}

func (e *Environment) RegisterFilters(filterList []interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strainerTemplateClassCache = make(map[string]*StrainerTemplate)
	for _, f := range filterList {
		e.strainerTemplate.AddFilter(f)
	}
}

func (e *Environment) CreateStrainer(context *engine.Context, filterList []interface{}) *engine.Strainer {
	if len(filterList) == 0 {
		return e.strainerTemplate.NewStrainer(context)
	}
	cacheKey := generateFilterCacheKey(filterList)

	e.mu.RLock()
	tmpl, ok := e.strainerTemplateClassCache[cacheKey]
	e.mu.RUnlock()

	if !ok {
		e.mu.Lock()
		if tmpl, ok = e.strainerTemplateClassCache[cacheKey]; !ok {
			tmpl = e.strainerTemplate.Clone()
			for _, f := range filterList {
				tmpl.AddFilter(f)
			}
			e.strainerTemplateClassCache[cacheKey] = tmpl
		}
		e.mu.Unlock()
	}
	return tmpl.NewStrainer(context)
}

func (e *Environment) FilterMethodNames() []string {
	return e.strainerTemplate.FilterMethodNames()
}

func (e *Environment) TagForName(name string) engine.TagFactory {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tags[name]
}

func (e *Environment) Freeze() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.frozen = true
}

// --- EnvironmentIface implementation ---

func (e *Environment) GetExceptionRenderer() engine.ExceptionRenderer {
	return e.ExceptionRenderer
}
func (e *Environment) GetFileSystem() engine.FileSystem    { return e.FileSystem }
func (e *Environment) GetDefaultResourceLimits() map[string]interface{} {
	return e.DefaultResourceLimits
}
func (e *Environment) GetLogger() engine.DebugLogger { return e.Logger }
func (e *Environment) GetErrorMode() string          { return e.ErrorMode }

func generateFilterCacheKey(filterList []interface{}) string {
	var sb strings.Builder
	for _, f := range filterList {
		t := reflect.TypeOf(f)
		sb.WriteString(t.PkgPath())
		sb.WriteByte('/')
		sb.WriteString(t.Name())
		sb.WriteByte('|')
	}
	return sb.String()
}
