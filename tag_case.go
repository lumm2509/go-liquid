package liquid

import (
	"fmt"
	"strings"
)

type Case struct {
	Block
	Left   interface{}
	Blocks []*Condition
}

func NewCase(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	c := &Case{
		Block: NewBlock(tagName, markup, parseContext),
	}
	c.Left, _ = parseContext.ParseExpression(markup)
	c.Blocks = make([]*Condition, 0)
	return c, nil
}

func (c *Case) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "when" {
		c.recordWhenCondition(markup)
		return true, nil
	}
	if tag == "else" {
		c.recordElseCondition(markup)
		return true, nil
	}
	if tag == "endcase" {
		return false, nil
	}
	return c.Block.UnknownTag(tag, markup)
}

func (c *Case) recordWhenCondition(markup string) {
	// Simplified when parsing
	expr, _ := c.Block.parseContext.ParseExpression(markup)
	block := NewCondition(c.Left, "==", expr)
	block.Attachment = NewBlockBody()
	c.Blocks = append(c.Blocks, block)
}

func (c *Case) recordElseCondition(markup string) {
	block := NewElseCondition()
	block.Attachment = NewBlockBody()
	c.Blocks = append(c.Blocks, block)
}

func (c *Case) Parse(tokenizer *Tokenizer) error {
	concretePC, ok := c.Block.parseContext.(*ParseContext)
	if !ok {
		return fmt.Errorf("case: expected concrete *ParseContext for sub-parsing")
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

func (c *Case) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	executeElseBlock := true
	for _, block := range c.Blocks {
		if block.IsElse() {
			if executeElseBlock {
				return block.Attachment.RenderToOutputBuffer(context, output)
			}
			continue
		}

		if block.Evaluate(context) {
			executeElseBlock = false
			err := block.Attachment.RenderToOutputBuffer(context, output)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
