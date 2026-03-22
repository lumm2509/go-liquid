package tags

import (
	"strings"

	"github.com/lumm2509/go-liquid/internal/engine"
	"github.com/lumm2509/go-liquid/internal/runtime"
)

type Break struct {
	engine.TagBase
}

func NewBreak(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Break{TagBase: engine.NewTagBase(tagName, markup, parseContext)}, nil
}

func (b *Break) RenderToOutputBuffer(ctx engine.RenderContext, _ *strings.Builder) error {
	ctx.PushInterrupt(runtime.NewBreakInterrupt(""))
	return nil
}

type Continue struct {
	engine.TagBase
}

func NewContinue(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Continue{TagBase: engine.NewTagBase(tagName, markup, parseContext)}, nil
}

func (c *Continue) RenderToOutputBuffer(ctx engine.RenderContext, _ *strings.Builder) error {
	ctx.PushInterrupt(runtime.NewContinueInterrupt(""))
	return nil
}
