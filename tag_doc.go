package liquid

import (
	"strings"
)

type Doc struct {
	Block
}

func NewDoc(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &Doc{
		Block: NewBlock(tagName, markup, parseContext),
	}, nil
}

func (d *Doc) Parse(tokenizer *Tokenizer) error {
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}

		if strings.HasPrefix(token, TagStartStr) {
			matches := FullToken.FindStringSubmatch(token)
			if matches != nil && matches[2] == "enddoc" {
				return nil
			}
		}
	}
	return &SyntaxError{BaseError: BaseError{Message: "'doc' tag was never closed"}}
}

func (d *Doc) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return nil
}

func (d *Doc) IsBlank() bool {
	return true
}
