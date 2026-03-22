package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Include struct {
	engine.TagBase
	parseContext engine.TagParseContext
	TemplateName interface{}
	Attributes   map[string]interface{}
}

func NewInclude(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	i := &Include{
		TagBase:      engine.NewTagBase(tagName, markup, parseContext),
		parseContext: parseContext,
		Attributes:   make(map[string]interface{}),
	}

	parts := strings.Split(markup, " ")
	if len(parts) > 0 {
		i.TemplateName, _ = parseContext.ParseExpression(parts[0])
	}

	return i, nil
}

func (i *Include) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	templateNameVal := ctx.Evaluate(i.TemplateName)
	templateName, ok := templateNameVal.(string)
	if !ok {
		return &engine.ArgumentError{BaseError: engine.BaseError{Message: "Illegal template name"}}
	}

	partial, err := loadPartial(templateName, ctx, i.parseContext.(*engine.ParseContext))
	if err != nil {
		return err
	}

	oldTemplateName := ctx.GetTemplateName()
	oldPartial := ctx.IsPartial()

	ctx.SetTemplateName(partial.Name)
	ctx.SetPartial(true)
	defer func() {
		ctx.SetTemplateName(oldTemplateName)
		ctx.SetPartial(oldPartial)
	}()

	return ctx.Stack(nil, func() error {
		for k, v := range i.Attributes {
			ctx.Set(k, ctx.Evaluate(v))
		}
		return partial.Root.RenderToOutputBuffer(ctx, output)
	})
}
