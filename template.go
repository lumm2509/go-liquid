package liquid

import (
	"fmt"
	"strings"

	"github.com/go-liquid/internal/runtime"
)

type Template struct {
	Root            *Document
	Name            string
	ResourceLimits  *runtime.ResourceLimits
	Warnings        []error
	Environment     *Environment
	Registers       map[string]interface{}
	Assigns         map[string]interface{}
	InstanceAssigns map[string]interface{}
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
func Parse(source string, options map[string]interface{}) (*Template, error) {
	return ParseWithEnv(source, NewEnvironment(), options)
}

// ParseWithEnv parsea source usando el Environment proporcionado.
// El Environment permite registrar tags y filtros custom, configurar el FileSystem, etc.
func ParseWithEnv(source string, env *Environment, options map[string]interface{}) (*Template, error) {
	t := newTemplateWithEnv(env)
	return t.Parse(source, options)
}

func (t *Template) Parse(source string, options map[string]interface{}) (templateResult *Template, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("liquid parse error: %v", r)
			templateResult = nil
		}
	}()
	parseContext := NewParseContext(options)
	parseContext.Environment = t.Environment

	tokenizer := parseContext.NewTokenizer(source, 1, false)
	doc, err := ParseDocument(tokenizer, parseContext)
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
	return t.renderInternal(assigns, opts)
}

// RenderWithMap es la API legacy que acepta opciones como map[string]interface{}.
//
// Deprecated: usa Render con *RenderOptions.
func (t *Template) RenderWithMap(assigns map[string]interface{}, options map[string]interface{}) (string, error) {
	return t.renderInternal(assigns, renderOptionsFromMap(options))
}

func (t *Template) renderInternal(assigns map[string]interface{}, opts *RenderOptions) (renderResult string, err error) {
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
	environments = append(environments, t.Assigns)

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

	ctx := NewContext(
		environments,
		outerScope,
		registers,
		rethrowErrors,
		t.ResourceLimits.Fork(), // fresh counters per render, same configured limits
		[]map[string]interface{}{},
		t.Environment,
	)

	if opts != nil {
		ctx.StrictVariables = opts.StrictVariables
		ctx.StrictFilters = opts.StrictFilters
	}

	ctx.TemplateName = t.Name

	var sb strings.Builder
	err = t.Root.RenderToOutputBuffer(ctx, &sb)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}
