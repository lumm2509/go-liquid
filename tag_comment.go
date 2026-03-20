package liquid

import (
	"strings"
)

type Comment struct {
	Block
}

func NewComment(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &Comment{
		Block: NewBlock(tagName, markup, parseContext),
	}, nil
}

func (c *Comment) Parse(tokenizer *Tokenizer) error {
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}

		// Search for endcomment
		if strings.HasPrefix(token, TagStartStr) {
			matches := FullToken.FindStringSubmatch(token)
			if matches != nil && matches[2] == "endcomment" {
				return nil
			}
		}
	}
	return &SyntaxError{BaseError: BaseError{Message: "'comment' tag was never closed"}}
}

func (c *Comment) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return nil
}

func (c *Comment) IsBlank() bool {
	return true
}
