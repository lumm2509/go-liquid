package tags

import (
	"fmt"
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Case struct {
	engine.Block
	Left   interface{}
	Blocks []*engine.Condition
}

func NewCase(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	c := &Case{
		Block:  engine.NewBlock(tagName, markup, parseContext),
		Blocks: make([]*engine.Condition, 0),
	}
	c.Left, _ = parseContext.ParseExpression(markup)
	return c, nil
}

func (c *Case) UnknownTag(tag string, markup string) (bool, error) {
	switch tag {
	case "when":
		c.recordWhenCondition(markup)
		return true, nil
	case "else":
		c.recordElseCondition()
		return true, nil
	case "endcase":
		return false, nil
	}
	return c.Block.UnknownTag(tag, markup)
}

func (c *Case) recordWhenCondition(markup string) {
	expr, _ := c.Block.ParseContext().ParseExpression(markup)
	block := engine.NewCondition(c.Left, "==", expr)
	block.Attachment = engine.NewBlockBody()
	c.Blocks = append(c.Blocks, block)
}

func (c *Case) recordElseCondition() {
	block := engine.NewElseCondition()
	block.Attachment = engine.NewBlockBody()
	c.Blocks = append(c.Blocks, block)
}

func (c *Case) Parse(tokenizer *engine.Tokenizer) error {
	concretePC, ok := c.Block.ParseContext().(*engine.ParseContext)
	if !ok {
		return fmt.Errorf("case: expected concrete *engine.ParseContext for sub-parsing")
	}
	for {
		targetBody := c.Block.Body
		if len(c.Blocks) > 0 {
			targetBody = c.Blocks[len(c.Blocks)-1].Attachment
		}
		continued, err := targetBody.Parse(tokenizer, concretePC, c.UnknownTag)
		if err != nil {
			return err
		}
		if !continued {
			break
		}
	}
	return nil
}

func (c *Case) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	concreteCtx := asContext(ctx)
	executeElseBlock := true
	for _, block := range c.Blocks {
		if block.IsElse() {
			if executeElseBlock {
				return block.Attachment.RenderToOutputBuffer(ctx, output)
			}
			continue
		}
		if block.Evaluate(concreteCtx) {
			executeElseBlock = false
			if err := block.Attachment.RenderToOutputBuffer(ctx, output); err != nil {
				return err
			}
		}
	}
	return nil
}
