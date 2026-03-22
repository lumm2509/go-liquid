package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lumm2509/go-liquid/internal/engine"
	"github.com/lumm2509/go-liquid/internal/runtime"
)

// ForloopDrop is defined in internal/engine so Context can hold a direct pointer.
// Re-exported as a type alias so external code that embeds or inspects it still works.
type ForloopDrop = engine.ForloopDrop

var ForSyntax = regexp.MustCompile(`^([\w\-]+)\s+in\s+(.+)$`)
var forLimitRe = regexp.MustCompile(`\blimit:(\S+)`)
var forOffsetRe = regexp.MustCompile(`\boffset:(\S+)`)
var forReversedRe = regexp.MustCompile(`\breversed\b`)
var forAttrStripRe = regexp.MustCompile(`\s+(?:limit:\S+|offset:\S+|reversed)`)

type For struct {
	engine.Block
	VariableName   string
	CollectionName interface{}
	Reversed       bool
	Limit          interface{}
	Offset         interface{}
}

func NewFor(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	f := &For{Block: engine.NewBlock(tagName, markup, parseContext)}

	matches := ForSyntax.FindStringSubmatch(markup)
	if matches == nil {
		return nil, engine.SyntaxError{BaseError: engine.BaseError{Message: "Syntax Error in 'for' - Valid syntax: for [item] in [collection]"}}
	}

	f.VariableName = matches[1]
	rest := matches[2]

	if m := forLimitRe.FindStringSubmatch(rest); m != nil {
		f.Limit, _ = parseContext.ParseExpression(m[1])
	}
	if m := forOffsetRe.FindStringSubmatch(rest); m != nil {
		f.Offset, _ = parseContext.ParseExpression(m[1])
	}
	f.Reversed = forReversedRe.MatchString(rest)

	collectionMarkup := strings.TrimSpace(forAttrStripRe.ReplaceAllString(rest, ""))
	f.CollectionName, _ = parseContext.ParseExpression(collectionMarkup)

	return f, nil
}

func (f *For) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "endfor" {
		return false, nil
	}
	return f.Block.UnknownTag(tag, markup)
}

func (f *For) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	c, ok := ctx.(*engine.Context)
	if !ok {
		return fmt.Errorf("tag_for: expected *engine.Context")
	}

	collection := ctx.Evaluate(f.CollectionName)
	if collection == nil {
		return nil
	}

	var fromPtr, toPtr *int
	offset := 0
	if f.Offset != nil {
		if n, err := engine.UtilsToInteger(ctx.Evaluate(f.Offset)); err == nil {
			offset = n
			fromPtr = &offset
		}
	}
	if f.Limit != nil {
		if n, err := engine.UtilsToInteger(ctx.Evaluate(f.Limit)); err == nil {
			to := offset + n
			toPtr = &to
		}
	}

	iter := engine.ToIterable(collection, fromPtr, toPtr)
	length := iter.Len()
	drop := engine.ForloopDrop{Length: length}

	// D8: store forloop in a dedicated Context field instead of the scope map.
	// This lets VariableLookup.Evaluate skip the scope scan for {{ forloop.* }}.
	prevForloop := c.Forloop
	c.Forloop = &drop
	defer func() { c.Forloop = prevForloop }()

	return ctx.Stack(nil, func() error {
		for i := 0; i < length; i++ {
			if err := c.ResourceLimits.IncrementRenderScore(1); err != nil {
				return engine.MemoryError{BaseError: engine.BaseError{Message: err.Error(), Cause: err}}
			}
			drop.Index = i + 1
			drop.Index0 = i
			drop.Rindex = length - i
			drop.Rindex0 = length - i - 1
			drop.First = i == 0
			drop.Last = i == length-1

			dataIdx := i
			if f.Reversed {
				dataIdx = length - 1 - i
			}
			ctx.Set(f.VariableName, iter.At(dataIdx))

			if err := f.Block.RenderToOutputBuffer(ctx, output); err != nil {
				return err
			}
			if ctx.Interrupt() {
				interrupt := ctx.PopInterrupt()
				if _, ok := interrupt.(*runtime.BreakInterrupt); ok {
					return nil
				}
			}
		}
		return nil
	})
}
