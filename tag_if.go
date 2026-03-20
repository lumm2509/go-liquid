package liquid

import (
	"fmt"
	"strings"
)

type If struct {
	Block
	Conditions []*Condition
}

func NewIf(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	i := &If{
		Block: NewBlock(tagName, markup, parseContext),
	}
	i.Conditions = make([]*Condition, 0)
	i.pushBlock("if", markup)
	return i, nil
}

func (i *If) pushBlock(tag string, markup string) {
	var condition *Condition
	if tag == "else" {
		condition = NewElseCondition()
	} else {
		var err error
		condition, err = ParseCondition(markup, i.Block.parseContext)
		if err != nil {
			// In a real implementation we might want to handle this error better
			condition = NewCondition(markup, "", nil)
		}
	}
	condition.Attachment = NewBlockBody()
	i.Conditions = append(i.Conditions, condition)
}

func (i *If) Parse(tokenizer *Tokenizer) error {
	concretePC, ok := i.Block.parseContext.(*ParseContext)
	if !ok {
		return fmt.Errorf("if: expected concrete *ParseContext for sub-parsing")
	}
	for {
		continued, err := i.Conditions[len(i.Conditions)-1].Attachment.Parse(tokenizer, concretePC, i.UnknownTag)
		if err != nil {
			return err
		}
		if !continued {
			break
		}
	}
	return nil
}

func (i *If) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "elsif" || tag == "else" {
		i.pushBlock(tag, markup)
		return true, nil
	}
	if tag == "endif" {
		return false, nil
	}
	return i.Block.UnknownTag(tag, markup)
}

func (i *If) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	for _, condition := range i.Conditions {
		if condition.IsElse() || condition.Evaluate(context) {
			return condition.Attachment.RenderToOutputBuffer(context, output)
		}
	}
	return nil
}
