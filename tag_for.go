package liquid

import (
	"regexp"
	"strings"
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

func NewFor(tagName string, markup string, parseContext *ParseContext) (Tag, error) {
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
		f.Limit, _ = ParseExpression(m[1], parseContext.stringScanner, parseContext.expressionCache)
	}
	if m := forOffsetRe.FindStringSubmatch(rest); m != nil {
		f.Offset, _ = ParseExpression(m[1], parseContext.stringScanner, parseContext.expressionCache)
	}
	f.Reversed = forReversedRe.MatchString(rest)

	// Limpiar atributos para obtener solo la expresión de la colección
	collectionMarkup := strings.TrimSpace(forAttrStripRe.ReplaceAllString(rest, ""))
	f.CollectionName, _ = ParseExpression(collectionMarkup, parseContext.stringScanner, parseContext.expressionCache)

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

	return context.Stack(nil, func() error {
		for i, item := range segment {
			idx := i + 1 // 1-based
			context.Scopes[0][f.VariableName] = item
			context.Scopes[0]["forloop"] = map[string]interface{}{
				"index":   idx,
				"index0":  i,
				"rindex":  length - i,
				"rindex0": length - i - 1,
				"first":   i == 0,
				"last":    i == length-1,
				"length":  length,
			}

			err := f.Block.RenderToOutputBuffer(context, output)
			if err != nil {
				return err
			}
			if context.Interrupt() {
				interrupt := context.PopInterrupt()
				if _, ok := interrupt.(*BreakInterrupt); ok {
					return nil
				}
				// ContinueInterrupt: descartar y continuar siguiente iteración
			}
		}
		return nil
	})
}
