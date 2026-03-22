package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type InlineComment struct {
	engine.TagBase
}

func NewInlineComment(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &InlineComment{TagBase: engine.NewTagBase(tagName, markup, parseContext)}, nil
}

func (i *InlineComment) RenderToOutputBuffer(_ engine.RenderContext, _ *strings.Builder) error {
	return nil
}

func (i *InlineComment) IsBlank() bool { return true }
