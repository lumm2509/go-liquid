package tags

import (
	"regexp"
	"strings"

	"github.com/go-liquid/internal/engine"
	"github.com/go-liquid/internal/runtime"
)

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

	segment := engine.SliceCollection(collection, fromPtr, toPtr)

	if f.Reversed {
		for i, j := 0, len(segment)-1; i < j; i, j = i+1, j-1 {
			segment[i], segment[j] = segment[j], segment[i]
		}
	}

	length := len(segment)
	forloopMap := map[string]interface{}{
		"index": 0, "index0": 0, "rindex": 0, "rindex0": 0,
		"first": false, "last": false, "length": length,
	}

	return ctx.Stack(nil, func() error {
		ctx.Set("forloop", forloopMap)
		for i, item := range segment {
			idx := i + 1
			forloopMap["index"] = idx
			forloopMap["index0"] = i
			forloopMap["rindex"] = length - i
			forloopMap["rindex0"] = length - i - 1
			forloopMap["first"] = i == 0
			forloopMap["last"] = i == length-1

			ctx.Set(f.VariableName, item)

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
