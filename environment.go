package liquid

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// ExceptionRenderer define cómo se procesan las excepciones al renderizar
type ExceptionRenderer func(error) error

// DebugEvent es el tipo que se pasa al DebugLogger en cada evento.
type DebugEvent struct {
	Event string
	Data  map[string]interface{}
}

// DebugLogger es la interfaz que puede implementar el consumer para observar
// eventos internos del engine (filtros no encontrados, overflow, etc.).
// nil = no-op, cero overhead.
type DebugLogger interface {
	Log(event DebugEvent)
}

// Environment contiene toda la configuración global
type Environment struct {
	ErrorMode             string
	ExceptionRenderer     ExceptionRenderer
	FileSystem            FileSystem
	DefaultResourceLimits map[string]interface{}
	Logger                DebugLogger // nil = no-op

	// Privados: usar RegisterTag / RegisterFilter / TagForName / FilterMethodNames
	tags             map[string]TagFactory
	strainerTemplate *StrainerTemplate

	// Caché para combinaciones de filtros específicos
	strainerTemplateClassCache map[string]*StrainerTemplate
	mu                         sync.RWMutex
	frozen                     bool
}

// log emite un evento al Logger si está configurado.
// El caller debe construir el map solo dentro del bloque if, para evitar
// allocations cuando Logger es nil.
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

// Default devuelve la instancia por defecto del entorno (Singleton)
func DefaultEnvironment() *Environment {
	defaultEnvOnce.Do(func() {
		defaultEnv = NewEnvironment()
	})
	return defaultEnv
}

// NewEnvironment equivale a initialize
func NewEnvironment() *Environment {
	env := &Environment{
		ErrorMode:                  "lax",
		tags:                       make(map[string]TagFactory),
		ExceptionRenderer:          func(err error) error { return err },
		FileSystem:                 &BlankFileSystem{},
		DefaultResourceLimits:      make(map[string]interface{}),
		strainerTemplateClassCache: make(map[string]*StrainerTemplate),
	}

	// Copiar tags estándar (asumiendo que Tags.StandardTags está definido en otro archivo)
	for k, v := range StandardTags {
		env.tags[k] = v
	}

	// Inicializar el StrainerTemplate con filtros estándar
	env.strainerTemplate = NewStrainerTemplate()
	env.strainerTemplate.AddFilter(StandardFilters{})

	return env
}

// Build equivale al método self.build de Ruby con soporte para callbacks
func BuildEnvironment(fn func(*Environment)) *Environment {
	env := NewEnvironment()
	if fn != nil {
		fn(env)
	}
	env.Freeze()
	return env
}

// RegisterTag registra una nueva etiqueta
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

// RegisterFilter registra un nuevo módulo de filtros
func (e *Environment) RegisterFilter(filter interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strainerTemplateClassCache = make(map[string]*StrainerTemplate) // Clear cache
	e.strainerTemplate.AddFilter(filter)
}

// RegisterFilters registra múltiples filtros
func (e *Environment) RegisterFilters(filters []interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strainerTemplateClassCache = make(map[string]*StrainerTemplate)
	for _, f := range filters {
		e.strainerTemplate.AddFilter(f)
	}
}

// CreateStrainer crea una instancia de strainer (procesador de filtros) para un contexto
func (e *Environment) CreateStrainer(context *Context, filters []interface{}) *Strainer {
	if len(filters) == 0 {
		return e.strainerTemplate.NewStrainer(context)
	}

	// Generar una llave para el cache basada en los filtros adicionales
	cacheKey := generateFilterCacheKey(filters)

	e.mu.RLock()
	template, ok := e.strainerTemplateClassCache[cacheKey]
	e.mu.RUnlock()

	if !ok {
		e.mu.Lock()
		// Double-check locking
		if template, ok = e.strainerTemplateClassCache[cacheKey]; !ok {
			// Simular la herencia de Ruby creando un nuevo template que extiende el base
			template = e.strainerTemplate.Clone()
			for _, f := range filters {
				template.AddFilter(f)
			}
			e.strainerTemplateClassCache[cacheKey] = template
		}
		e.mu.Unlock()
	}

	return template.NewStrainer(context)
}

// FilterMethodNames devuelve los nombres de métodos disponibles
func (e *Environment) FilterMethodNames() []string {
	return e.strainerTemplate.FilterMethodNames()
}

// TagForName devuelve la factoría de tags asociada a un nombre
func (e *Environment) TagForName(name string) TagFactory {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tags[name]
}

// Freeze marca el entorno como inmutable
func (e *Environment) Freeze() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.frozen = true
}

// Helper interno para generar llaves de caché basada en tipos (no en valores).
// Determinístico, sin colisiones entre tipos distintos.
func generateFilterCacheKey(filters []interface{}) string {
	var sb strings.Builder
	for _, f := range filters {
		t := reflect.TypeOf(f)
		sb.WriteString(t.PkgPath())
		sb.WriteByte('/')
		sb.WriteString(t.Name())
		sb.WriteByte('|')
	}
	return sb.String()
}
