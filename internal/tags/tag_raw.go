package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Raw struct {
	engine.TagBase
	Content string
}

func NewRaw(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Raw{TagBase: engine.NewTagBase(tagName, markup, parseContext)}, nil
}

func (r *Raw) Parse(tokenizer *engine.Tokenizer) error {
	var sb strings.Builder
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}
		if strings.HasPrefix(token, engine.TagStartStr) {
			matches := engine.FullToken.FindStringSubmatch(token)
			if matches != nil && matches[2] == "endraw" {
				r.Content = sb.String()
				return nil
			}
		}
		sb.WriteString(token)
	}
	return &engine.SyntaxError{BaseError: engine.BaseError{Message: "'raw' tag was never closed"}}
}

func (r *Raw) RenderToOutputBuffer(_ engine.RenderContext, output *strings.Builder) error {
	output.WriteString(r.Content)
	return nil
}

func (r *Raw) IsBlank() bool { return r.Content == "" }
