package liquid

import (
	"strings"
)

type Ifchanged struct {
	Block
}

func NewIfchanged(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	return &Ifchanged{
		Block: NewBlock(tagName, markup, parseContext),
	}, nil
}

func (i *Ifchanged) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	var sb strings.Builder
	err := i.Block.RenderToOutputBuffer(context, &sb)
	if err != nil {
		return err
	}

	newContent := sb.String()
	lastContent, _ := context.Registers.Get("ifchanged").(string)

	if newContent != lastContent {
		context.Registers.Set("ifchanged", newContent)
		output.WriteString(newContent)
	}

	return nil
}
