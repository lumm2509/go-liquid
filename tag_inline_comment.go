package liquid

import (
	"strings"
)

type InlineComment struct {
	TagBase
}

func NewInlineComment(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &InlineComment{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}, nil
}

func (i *InlineComment) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return nil
}

func (i *InlineComment) IsBlank() bool {
	return true
}
