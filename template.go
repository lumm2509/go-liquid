package liquid

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lumm2509/go-liquid/internal/engine"
	"github.com/lumm2509/go-liquid/internal/runtime"
)

// pre-grown to 4 KiB; builders that grew beyond 512 KiB are not returned to avoid retaining large buffers
var renderBuilderPool = sync.Pool{
	New: func() interface{} {
		sb := &strings.Builder{}
		sb.Grow(4096)
		return sb
	},
}

type Template struct {
	Root            *engine.Document
	Name            string
	ResourceLimits  *runtime.ResourceLimits
	Warnings        []error
	Environment     *Environment
	Registers       map[string]interface{}
	Assigns         map[string]interface{}
	InstanceAssigns map[string]interface{}
	mu              sync.RWMutex
}

// NewTemplate creates a Template with the default (singleton) Environment.
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

// Parse parses source with a fresh default Environment.
// Parsing is expensive; use TemplateCache when rendering the same template repeatedly.
func Parse(source string, options map[string]interface{}) (*Template, error) {
	return ParseWithEnv(source, NewEnvironment(), options)
}

func ParseWithEnv(source string, env *Environment, options map[string]interface{}) (*Template, error) {
	t := newTemplateWithEnv(env)
	return t.Parse(source, options)
}

// ParseWithOptions is the type-safe alternative to ParseWithEnv.
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

func (t *Template) Render(assigns map[string]interface{}, opts *RenderOptions) (string, error) {
	return t.renderInternal(context.Background(), assigns, opts)
}

// RenderWithContext propagates a Go context; returns ctx.Err() on cancellation.
func (t *Template) RenderWithContext(ctx context.Context, assigns map[string]interface{}, opts *RenderOptions) (string, error) {
	return t.renderInternal(ctx, assigns, opts)
}

// Deprecated: use Render with *RenderOptions.
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
	// D9: skip copy when maps are empty (common case)
	t.mu.RLock()
	if len(t.Assigns) > 0 {
		assignsCopy := make(map[string]interface{}, len(t.Assigns))
		for k, v := range t.Assigns {
			assignsCopy[k] = v
		}
		environments = append(environments, assignsCopy)
	}
	t.mu.RUnlock()

	var registers map[string]interface{}
	if len(t.Registers) > 0 {
		registers = make(map[string]interface{}, len(t.Registers))
		for k, v := range t.Registers {
			registers[k] = v
		}
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

	// copy InstanceAssigns — assign tag writes to Scopes[0], must not mutate across renders
	outerScope := make(map[string]interface{}, len(t.InstanceAssigns))
	for k, v := range t.InstanceAssigns {
		outerScope[k] = v
	}

	ctx := engine.NewContext(engine.ContextConfig{
		Environments:       environments,
		OuterScope:         outerScope,
		Registers:          registers,
		RethrowErrors:      rethrowErrors,
		ResourceLimits:     t.ResourceLimits.Fork(), // fresh counters, same limits
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

	// D3: reuse builder backing buffer; strings.Clone makes an independent copy
	sb := renderBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	err = t.Root.RenderToOutputBuffer(ctx, sb)
	result := strings.Clone(sb.String())
	if sb.Cap() <= 512*1024 {
		renderBuilderPool.Put(sb)
	}
	if err != nil {
		return "", err
	}
	return result, nil
}

func (t *Template) SetAssign(key string, value interface{}) {
	t.mu.Lock()
	t.Assigns[key] = value
	t.mu.Unlock()
}
