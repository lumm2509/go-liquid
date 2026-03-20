package liquid

import (
	"strings"
)

type Capture struct {
	Block
	To string
}

func NewCapture(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	c := &Capture{
		Block: NewBlock(tagName, markup, parseContext),
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

func (c *Capture) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	var sb strings.Builder
	context.ResourceLimits.WithCapture(func() {
		c.Block.RenderToOutputBuffer(context, &sb)
	})
	context.Scopes[0][c.To] = sb.String()
	return nil
}

func (c *Capture) IsBlank() bool {
	return true
}
