package liquid

import (
	"strings"
)

// Include supports simple template inclusion: {% include 'name' %}.
// The 'with' and 'for' sub-template syntaxes are not implemented.
type Include struct {
	TagBase
	parseContext *ParseContext
	TemplateName interface{}
	Attributes   map[string]interface{}
}

func NewInclude(tagName string, markup string, parseContext *ParseContext) (Tag, error) {
	i := &Include{
		TagBase:      NewTagBase(tagName, markup, parseContext),
		parseContext: parseContext,
		Attributes:   make(map[string]interface{}),
	}

	parts := strings.Split(markup, " ")
	if len(parts) > 0 {
		i.TemplateName, _ = ParseExpression(parts[0], parseContext.stringScanner, parseContext.expressionCache)
	}

	return i, nil
}

func (i *Include) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	templateNameVal := context.Evaluate(i.TemplateName)
	templateName, ok := templateNameVal.(string)
	if !ok {
		return &ArgumentError{BaseError: BaseError{Message: "Illegal template name"}}
	}

	partial, err := LoadPartial(templateName, context, i.parseContext)
	if err != nil {
		return err
	}

	oldTemplateName := context.TemplateName
	oldPartial := context.Partial

	context.TemplateName = partial.Name
	context.Partial = true
	defer func() {
		context.TemplateName = oldTemplateName
		context.Partial = oldPartial
	}()

	return context.Stack(nil, func() error {
		for k, v := range i.Attributes {
			context.Scopes[0][k] = context.Evaluate(v)
		}

		return partial.Root.RenderToOutputBuffer(context, output)
	})
}
