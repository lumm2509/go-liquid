package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Doc struct {
	engine.Block
}

func NewDoc(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Doc{Block: engine.NewBlock(tagName, markup, parseContext)}, nil
}

func (d *Doc) Parse(tokenizer *engine.Tokenizer) error {
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}
		if strings.HasPrefix(token, engine.TagStartStr) {
			matches := engine.FullToken.FindStringSubmatch(token)
			if matches != nil && matches[2] == "enddoc" {
				return nil
			}
		}
	}
	return &engine.SyntaxError{BaseError: engine.BaseError{Message: "'doc' tag was never closed"}}
}

func (d *Doc) RenderToOutputBuffer(_ engine.RenderContext, _ *strings.Builder) error { return nil }
func (d *Doc) IsBlank() bool                                                          { return true }
