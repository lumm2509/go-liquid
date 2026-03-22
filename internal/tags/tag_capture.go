package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Capture struct {
	engine.Block
	To string
}

func NewCapture(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	c := &Capture{
		Block: engine.NewBlock(tagName, markup, parseContext),
	}
	c.To = strings.TrimSpace(markup)
	return c, nil
}

func (c *Capture) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "endcapture" {
		return false, nil
	}
	return c.Block.UnknownTag(tag, markup)
}

func (c *Capture) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	var sb strings.Builder
	concreteCtx := asContext(ctx)
	if concreteCtx != nil {
		concreteCtx.ResourceLimits.WithCapture(func() {
			c.Block.RenderToOutputBuffer(ctx, &sb)
		})
	} else {
		c.Block.RenderToOutputBuffer(ctx, &sb)
	}
	ctx.Set(c.To, sb.String())
	return nil
}

func (c *Capture) IsBlank() bool { return true }
