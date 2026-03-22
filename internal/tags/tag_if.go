package tags

import (
	"fmt"
	"strings"

	"github.com/lumm2509/go-liquid/internal/engine"
)

type If struct {
	engine.Block
	Conditions []*engine.Condition
}

func NewIf(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	i := &If{Block: engine.NewBlock(tagName, markup, parseContext)}
	i.Conditions = make([]*engine.Condition, 0)
	i.pushBlock("if", markup)
	return i, nil
}

func (i *If) pushBlock(tag string, markup string) {
	var condition *engine.Condition
	if tag == "else" {
		condition = engine.NewElseCondition()
	} else {
		var err error
		condition, err = engine.ParseCondition(markup, i.Block.ParseContext())
		if err != nil {
			condition = engine.NewCondition(markup, "", nil)
		}
	}
	condition.Attachment = engine.NewBlockBody()
	i.Conditions = append(i.Conditions, condition)
}

func (i *If) Parse(tokenizer *engine.Tokenizer) error {
	concretePC, ok := i.Block.ParseContext().(*engine.ParseContext)
	if !ok {
		return fmt.Errorf("if: expected concrete *engine.ParseContext for sub-parsing")
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

func (i *If) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	c := asContext(ctx)
	for _, condition := range i.Conditions {
		if condition.IsElse() || condition.Evaluate(c) {
			return condition.Attachment.RenderToOutputBuffer(ctx, output)
		}
	}
	return nil
}
