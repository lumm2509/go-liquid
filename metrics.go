package liquid

import (
	"context"
	"strings"
)

// RenderMetrics holds performance data collected during a single template render.
// Obtain via Template.RenderWithMetrics.
type RenderMetrics struct {
	Errors   int
	Warnings int
}

func (t *Template) RenderWithMetrics(assigns map[string]interface{}, opts *RenderOptions) (string, RenderMetrics, error) {
	return t.renderWithMetricsInternal(context.Background(), assigns, opts)
}

func (t *Template) RenderWithContextAndMetrics(ctx context.Context, assigns map[string]interface{}, opts *RenderOptions) (string, RenderMetrics, error) {
	return t.renderWithMetricsInternal(ctx, assigns, opts)
}

func (t *Template) renderWithMetricsInternal(goCtx context.Context, assigns map[string]interface{}, opts *RenderOptions) (string, RenderMetrics, error) {
	if t.Root == nil {
		return "", RenderMetrics{}, nil
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

	outerScope := make(map[string]interface{}, len(t.InstanceAssigns))
	for k, v := range t.InstanceAssigns {
		outerScope[k] = v
	}

	ctx := NewContext(ContextConfig{
		Environments:       environments,
		OuterScope:         outerScope,
		Registers:          registers,
		RethrowErrors:      rethrowErrors,
		ResourceLimits:     t.ResourceLimits.Fork(),
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
	err := t.Root.RenderToOutputBuffer(ctx, &sb)

	m := RenderMetrics{
		Errors:   len(ctx.Errors),
		Warnings: len(ctx.Warnings),
	}

	if err != nil {
		return "", m, err
	}
	return sb.String(), m, nil
}
