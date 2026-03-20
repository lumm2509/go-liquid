package liquid

import (
	"strings"
)

type Echo struct {
	TagBase
	Variable *Variable
}

func NewEcho(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	e := &Echo{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}
	e.Variable = NewVariable(markup, parseContext.(*ParseContext))
	return e, nil
}

func (e *Echo) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return e.Variable.RenderToOutputBuffer(context, output)
}
