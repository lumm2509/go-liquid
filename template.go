package liquid

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-liquid/internal/engine"
	"github.com/go-liquid/internal/runtime"
)

type Template struct {
	Root            *engine.Document
	Name            string
	ResourceLimits  *runtime.ResourceLimits
	Warnings        []error
	Environment     *Environment
	Registers       map[string]interface{}
	Assigns         map[string]interface{}
	InstanceAssigns map[string]interface{}
	mu              sync.RWMutex // protege Assigns e InstanceAssigns
}

// NewTemplate crea un Template con el Environment por defecto (singleton).
// Prefer ParseWithEnv para control explícito del entorno.
func NewTemplate() *Template {
	env := DefaultEnvironment()
	return newTemplateWithEnv(env)
}

func newTemplateWithEnv(env *Environment) *Template {
	return &Template{
		Environment:     env,
		ResourceLimits:  runtime.NewResourceLimits(env.DefaultResourceLimits),
		Registers:       make(map[string]interface{}),
		Assigns:         make(map[string]interface{}),
		InstanceAssigns: make(map[string]interface{}),
	}
}

// Parse parsea source usando un Environment fresco con la configuración por defecto.
// Para usar un Environment customizado, usa ParseWithEnv.
//
// Parse es costoso: tokeniza el source y construye el AST completo en cada llamada.
// En servidores web que renderizan el mismo template repetidamente, usa TemplateCache
// para parsear una sola vez y renderizar muchas veces.
func Parse(source string, options map[string]interface{}) (*Template, error) {
	return ParseWithEnv(source, NewEnvironment(), options)
}

// ParseWithEnv parsea source usando el Environment proporcionado.
// El Environment permite registrar tags y filtros custom, configurar el FileSystem, etc.
func ParseWithEnv(source string, env *Environment, options map[string]interface{}) (*Template, error) {
	t := newTemplateWithEnv(env)
	return t.Parse(source, options)
}

// ParseWithOptions parsea source usando un Environment y opciones tipadas.
// Es la forma idiomática de pasar opciones de parseo; equivalente a
// ParseWithEnv con opciones en mapa pero con tipos seguros.
func ParseWithOptions(source string, env *Environment, opts *ParseOptions) (*Template, error) {
	if env == nil {
		env = NewEnvironment()
	}
	var m map[string]interface{}
	if opts != nil {
		m = opts.toMap()
	}
	return ParseWithEnv(source, env, m)
}

func (t *Template) Parse(source string, options map[string]interface{}) (templateResult *Template, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("liquid parse error: %v", r)
			templateResult = nil
		}
	}()
	parseContext := engine.NewParseContext(options)
	parseContext.Environment = t.Environment

	tokenizer := parseContext.NewTokenizer(source, 1, false)
	doc, err := engine.ParseDocument(tokenizer, parseContext)
	if err != nil {
		return nil, err
	}
	t.Root = doc
	t.Warnings = parseContext.Warnings
	return t, nil
}

// Render renderiza el template con los datos y opciones proporcionados.
// assigns contiene las variables disponibles en el template.
// opts puede ser nil para usar los valores por defecto.
func (t *Template) Render(assigns map[string]interface{}, opts *RenderOptions) (string, error) {
	return t.renderInternal(context.Background(), assigns, opts)
}

// RenderWithContext renderiza el template propagando un context.Context de Go.
// El contexto permite cancelación del render y propagación de trace IDs
// (OpenTelemetry, slog, etc.) hasta los tags y filtros custom.
// Si ctx se cancela durante el render, la operación retorna ctx.Err().
func (t *Template) RenderWithContext(ctx context.Context, assigns map[string]interface{}, opts *RenderOptions) (string, error) {
	return t.renderInternal(ctx, assigns, opts)
}

// RenderWithMap es la API legacy que acepta opciones como map[string]interface{}.
//
// Deprecated: usa Render con *RenderOptions.
func (t *Template) RenderWithMap(assigns map[string]interface{}, options map[string]interface{}) (string, error) {
	return t.renderInternal(context.Background(), assigns, renderOptionsFromMap(options))
}

func (t *Template) renderInternal(goCtx context.Context, assigns map[string]interface{}, opts *RenderOptions) (renderResult string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("liquid render error: %v", r)
			renderResult = ""
		}
	}()
	if t.Root == nil {
		return "", nil
	}

	environments := []map[string]interface{}{}
	if assigns != nil {
		environments = append(environments, assigns)
	}
	t.mu.RLock()
	assignsCopy := make(map[string]interface{}, len(t.Assigns))
	for k, v := range t.Assigns {
		assignsCopy[k] = v
	}
	t.mu.RUnlock()
	environments = append(environments, assignsCopy)

	registers := make(map[string]interface{})
	for k, v := range t.Registers {
		registers[k] = v
	}

	rethrowErrors := false
	if opts != nil {
		if opts.Registers != nil {
			for k, v := range opts.Registers {
				registers[k] = v
			}
		}
		rethrowErrors = opts.RethrowErrors
	}

	// Copy InstanceAssigns to prevent mutation across renders (assign tag writes to Scopes[0])
	outerScope := make(map[string]interface{}, len(t.InstanceAssigns))
	for k, v := range t.InstanceAssigns {
		outerScope[k] = v
	}

	ctx := engine.NewContext(engine.ContextConfig{
		Environments:       environments,
		OuterScope:         outerScope,
		Registers:          registers,
		RethrowErrors:      rethrowErrors,
		ResourceLimits:     t.ResourceLimits.Fork(), // fresh counters per render, same configured limits
		StaticEnvironments: []map[string]interface{}{},
		Environment:        t.Environment,
	})

	ctx.GoCtx = goCtx

	ctx.AutoEscape = true // secure default: HTML-escape all output
	if opts != nil {
		ctx.StrictVariables = opts.StrictVariables
		ctx.StrictFilters = opts.StrictFilters
		ctx.AutoEscape = !opts.DisableAutoEscape
	}

	ctx.TemplateName = t.Name

	var sb strings.Builder
	err = t.Root.RenderToOutputBuffer(ctx, &sb)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

// SetAssign sets a template-level assign variable in a thread-safe manner.
func (t *Template) SetAssign(key string, value interface{}) {
	t.mu.Lock()
	t.Assigns[key] = value
	t.mu.Unlock()
}
