package tags

import (
	"strings"

	"github.com/lumm2509/go-liquid/internal/engine"
)

type Echo struct {
	engine.TagBase
	Variable *engine.Variable
}

func NewEcho(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	e := &Echo{
		TagBase:  engine.NewTagBase(tagName, markup, parseContext),
		Variable: engine.NewVariable(markup, parseContext.(*engine.ParseContext)),
	}
	return e, nil
}

func (e *Echo) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	return e.Variable.RenderToOutputBuffer(ctx, output)
}
