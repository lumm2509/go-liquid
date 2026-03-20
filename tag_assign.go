package liquid

import (
	"regexp"
	"strings"
)

var AssignSyntax = regexp.MustCompile(`(?s)^([\w\-]+)\s*=\s*(.*)$`)

type Assign struct {
	TagBase
	To   string
	From *Variable
}

func NewAssign(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	a := &Assign{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}

	matches := AssignSyntax.FindStringSubmatch(markup)
	if matches != nil {
		a.To = matches[1]
		a.From = NewVariable(matches[2], parseContext.(*ParseContext))
	} else {
		return nil, SyntaxError{BaseError: BaseError{Message: "Syntax Error in 'assign' - Valid syntax: assign [var] = [source]"}}
	}

	return a, nil
}

func (a *Assign) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	val := a.From.Render(context)
	context.Scopes[0][a.To] = val
	context.ResourceLimits.IncrementAssignScore(a.assignScoreOf(val))
	return nil
}

func (a *Assign) assignScoreOf(val interface{}) int {
	if s, ok := val.(string); ok {
		return len(s)
	}
	return 1
}

func (a *Assign) IsBlank() bool {
	return true
}
