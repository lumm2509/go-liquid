package tags

import (
	"regexp"
	"strings"

	"github.com/go-liquid/internal/engine"
)

var assignSyntax = regexp.MustCompile(`(?s)^([\w\-]+)\s*=\s*(.*)$`)

type Assign struct {
	engine.TagBase
	To   string
	From *engine.Variable
}

func NewAssign(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	a := &Assign{
		TagBase: engine.NewTagBase(tagName, markup, parseContext),
	}
	matches := assignSyntax.FindStringSubmatch(markup)
	if matches == nil {
		return nil, engine.SyntaxError{BaseError: engine.BaseError{Message: "Syntax Error in 'assign' - Valid syntax: assign [var] = [source]"}}
	}
	a.To = matches[1]
	a.From = engine.NewVariable(matches[2], parseContext.(*engine.ParseContext))
	return a, nil
}

func (a *Assign) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	val := a.From.Render(ctx)
	ctx.Set(a.To, val)
	if c := asContext(ctx); c != nil {
		c.ResourceLimits.IncrementAssignScore(assignScoreOf(val))
	}
	return nil
}

func assignScoreOf(val interface{}) int {
	if s, ok := val.(string); ok {
		return len(s)
	}
	return 1
}

func (a *Assign) IsBlank() bool { return true }
