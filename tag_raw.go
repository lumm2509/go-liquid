package liquid

import (
	"strings"
)

type Raw struct {
	TagBase
	Content string
}

func NewRaw(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &Raw{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}, nil
}

func (r *Raw) Parse(tokenizer *Tokenizer) error {
	var sb strings.Builder
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}

		if strings.HasPrefix(token, TagStartStr) {
			matches := FullToken.FindStringSubmatch(token)
			if matches != nil && matches[2] == "endraw" {
				r.Content = sb.String()
				return nil
			}
		}
		sb.WriteString(token)
	}
	return &SyntaxError{BaseError: BaseError{Message: "'raw' tag was never closed"}}
}

func (r *Raw) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	output.WriteString(r.Content)
	return nil
}

func (r *Raw) IsBlank() bool {
	return r.Content == ""
}
