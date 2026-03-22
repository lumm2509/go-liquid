package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Comment struct {
	engine.Block
}

func NewComment(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Comment{Block: engine.NewBlock(tagName, markup, parseContext)}, nil
}

func (c *Comment) Parse(tokenizer *engine.Tokenizer) error {
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}
		if strings.HasPrefix(token, engine.TagStartStr) {
			matches := engine.FullToken.FindStringSubmatch(token)
			if matches != nil && matches[2] == "endcomment" {
				return nil
			}
		}
	}
	return &engine.SyntaxError{BaseError: engine.BaseError{Message: "'comment' tag was never closed"}}
}

func (c *Comment) RenderToOutputBuffer(_ engine.RenderContext, _ *strings.Builder) error { return nil }
func (c *Comment) IsBlank() bool                                                          { return true }
