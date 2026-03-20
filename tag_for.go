package liquid

import (
	"regexp"
	"strings"

	"github.com/go-liquid/internal/runtime"
)

var ForSyntax = regexp.MustCompile(`^([\w\-]+)\s+in\s+(.+)$`)
var forLimitRe = regexp.MustCompile(`\blimit:(\S+)`)
var forOffsetRe = regexp.MustCompile(`\boffset:(\S+)`)
var forReversedRe = regexp.MustCompile(`\breversed\b`)
var forAttrStripRe = regexp.MustCompile(`\s+(?:limit:\S+|offset:\S+|reversed)`)

type For struct {
	Block
	VariableName   string
	CollectionName interface{}
	Reversed       bool
	Limit          interface{} // expression or nil
	Offset         interface{} // expression or nil
}

func NewFor(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	f := &For{
		Block: NewBlock(tagName, markup, parseContext),
	}

	matches := ForSyntax.FindStringSubmatch(markup)
	if matches == nil {
		return nil, SyntaxError{BaseError: BaseError{Message: "Syntax Error in 'for' - Valid syntax: for [item] in [collection]"}}
	}

	f.VariableName = matches[1]
	rest := matches[2]

	// Extraer atributos del resto
	if m := forLimitRe.FindStringSubmatch(rest); m != nil {
		f.Limit, _ = parseContext.ParseExpression(m[1])
	}
	if m := forOffsetRe.FindStringSubmatch(rest); m != nil {
		f.Offset, _ = parseContext.ParseExpression(m[1])
	}
	f.Reversed = forReversedRe.MatchString(rest)

	// Limpiar atributos para obtener solo la expresión de la colección
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

func (f *For) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	collection := context.Evaluate(f.CollectionName)
	if collection == nil {
		return nil
	}

	// Calcular limit y offset.
	// SliceCollection recibe (from, to) donde to es índice final absoluto.
	// to = offset + limit
	var fromPtr, toPtr *int
	offset := 0
	if f.Offset != nil {
		if n, err := UtilsToInteger(context.Evaluate(f.Offset)); err == nil {
			offset = n
			fromPtr = &offset
		}
	}
	if f.Limit != nil {
		if n, err := UtilsToInteger(context.Evaluate(f.Limit)); err == nil {
			to := offset + n
			toPtr = &to
		}
	}

	segment := SliceCollection(collection, fromPtr, toPtr)

	if f.Reversed {
		for i, j := 0, len(segment)-1; i < j; i, j = i+1, j-1 {
			segment[i], segment[j] = segment[j], segment[i]
		}
	}

	length := len(segment)

	// Pre-allocate forloop map once and reuse across iterations.
	forloopMap := map[string]interface{}{
		"index": 0, "index0": 0, "rindex": 0, "rindex0": 0,
		"first": false, "last": false, "length": length,
	}

	return context.Stack(nil, func() error {
		context.Scopes[0]["forloop"] = forloopMap
		for i, item := range segment {
			idx := i + 1
			forloopMap["index"] = idx
			forloopMap["index0"] = i
			forloopMap["rindex"] = length - i
			forloopMap["rindex0"] = length - i - 1
			forloopMap["first"] = i == 0
			forloopMap["last"] = i == length-1

			context.Scopes[0][f.VariableName] = item

			err := f.Block.RenderToOutputBuffer(context, output)
			if err != nil {
				return err
			}
			if context.Interrupt() {
				interrupt := context.PopInterrupt()
				if _, ok := interrupt.(*runtime.BreakInterrupt); ok {
					return nil
				}
			}
		}
		return nil
	})
}
