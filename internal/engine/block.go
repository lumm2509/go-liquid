package engine

import (
	"fmt"
	"strings"
)

// Block is the base for block-level tags (if, for, case, etc.) that contain
// a body of child nodes delimited by an end tag.
type Block struct {
	TagBase
	Body         *BlockBody
	parseContext TagParseContext
}

func NewBlock(tagName string, markup string, parseContext TagParseContext) Block {
	return Block{
		TagBase:      NewTagBase(tagName, markup, parseContext),
		Body:         NewBlockBody(),
		parseContext: parseContext,
	}
}

func (b *Block) Parse(tokenizer *Tokenizer) error {
	b.Body = NewBlockBody()
	concretePC, ok := b.parseContext.(*ParseContext)
	if !ok {
		return fmt.Errorf("block: expected concrete *ParseContext for sub-parsing")
	}
	for {
		continued, err := b.Body.Parse(tokenizer, concretePC, b.UnknownTag)
		if err != nil {
			return err
		}
		if !continued {
			break
		}
	}
	return nil
}

func (b *Block) ParseContext() TagParseContext { return b.parseContext }

// UnknownTag handles end-tag and errors; subclasses may override
func (b *Block) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "end"+b.TagBase.name {
		return false, nil
	}
	return true, &SyntaxError{BaseError: BaseError{Message: "Unknown tag " + tag}}
}

func (b *Block) RenderToOutputBuffer(ctx RenderContext, output *strings.Builder) error {
	return b.Body.RenderToOutputBuffer(ctx, output)
}
