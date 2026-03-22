package engine

import (
	"fmt"
	"strings"
)

// Document is the root AST node produced by parsing a Liquid template.
type Document struct {
	Body *BlockBody
}

func NewDocument() *Document {
	return &Document{Body: NewBlockBody()}
}

// ParseDocument parses source and preloads static partials so missing files
// fail at parse time rather than first render
func ParseDocument(tokenizer *Tokenizer, parseContext *ParseContext) (*Document, error) {
	doc := NewDocument()
	if err := doc.Parse(tokenizer, parseContext); err != nil {
		return nil, err
	}
	// skip when parsing a partial — re-entrant preloads would recurse infinitely on mutual references
	if !parseContext.Partial {
		for _, node := range doc.NodeList() {
			if spl, ok := node.(StaticPartialLoader); ok {
				if err := spl.PreloadPartial(parseContext); err != nil {
					return nil, err
				}
			}
		}
	}
	return doc, nil
}

func (d *Document) NodeList() []Node { return d.Body.NodeList }

func (d *Document) Parse(tokenizer *Tokenizer, parseContext *ParseContext) error {
	for {
		continued, err := d.parseBody(tokenizer, parseContext)
		if err != nil {
			return fmt.Errorf("line %d: %w", parseContext.LineNumber, err)
		}
		if !continued {
			break
		}
	}
	return nil
}

func (d *Document) UnknownTag(tag string, markup string, tokenizer *Tokenizer) error {
	switch tag {
	case "else", "end":
		return fmt.Errorf("syntax error: unexpected outer tag '%s'", tag)
	default:
		return fmt.Errorf("syntax error: unknown tag '%s'", tag)
	}
}

func (d *Document) RenderToOutputBuffer(ctx RenderContext, output *strings.Builder) error {
	return d.Body.RenderToOutputBuffer(ctx, output)
}

func (d *Document) Render(ctx *Context) (string, error) {
	var sb strings.Builder
	if err := d.RenderToOutputBuffer(ctx, &sb); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (d *Document) parseBody(tokenizer *Tokenizer, parseContext *ParseContext) (bool, error) {
	return d.Body.Parse(tokenizer, parseContext, func(tagName string, tagMarkup string) (bool, error) {
		if tagName != "" {
			return true, d.UnknownTag(tagName, tagMarkup, tokenizer)
		}
		return false, nil
	})
}
