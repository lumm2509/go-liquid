package liquid

import (
	"strings"

	"github.com/go-liquid/internal/runtime"
)

type Break struct {
	TagBase
}

func NewBreak(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &Break{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}, nil
}

func (b *Break) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	context.PushInterrupt(runtime.NewBreakInterrupt(""))
	return nil
}

type Continue struct {
	TagBase
}

func NewContinue(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &Continue{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}, nil
}

func (c *Continue) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	context.PushInterrupt(runtime.NewContinueInterrupt(""))
	return nil
}
