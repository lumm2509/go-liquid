package tags

import (
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Ifchanged struct {
	engine.Block
}

func NewIfchanged(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Ifchanged{Block: engine.NewBlock(tagName, markup, parseContext)}, nil
}

func (i *Ifchanged) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	var sb strings.Builder
	if err := i.Block.RenderToOutputBuffer(ctx, &sb); err != nil {
		return err
	}

	newContent := sb.String()
	lastContent, _ := ctx.RegisterGet("ifchanged").(string)

	if newContent != lastContent {
		ctx.RegisterSet("ifchanged", newContent)
		output.WriteString(newContent)
	}
	return nil
}
