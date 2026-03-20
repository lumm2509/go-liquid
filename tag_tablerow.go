package liquid

import (
	"fmt"
	"regexp"
	"strings"
)

var tablerowColsRe = regexp.MustCompile(`\bcols:(\S+)`)
var tablerowAttrStripRe = regexp.MustCompile(`\s+cols:\S+`)

type TableRow struct {
	Block
	VariableName   string
	CollectionName interface{}
	Cols           interface{} // expression or nil
}

func NewTableRow(tagName string, markup string, parseContext *ParseContext) (Tag, error) {
	t := &TableRow{
		Block: NewBlock(tagName, markup, parseContext),
	}

	matches := ForSyntax.FindStringSubmatch(markup)
	if matches == nil {
		return nil, SyntaxError{BaseError: BaseError{Message: "Syntax Error in 'tablerow' - Valid syntax: tablerow [item] in [collection]"}}
	}

	t.VariableName = matches[1]
	rest := matches[2]

	// Extraer cols:N del markup
	if m := tablerowColsRe.FindStringSubmatch(rest); m != nil {
		t.Cols, _ = ParseExpression(m[1], parseContext.stringScanner, parseContext.expressionCache)
	}

	collectionMarkup := strings.TrimSpace(tablerowAttrStripRe.ReplaceAllString(rest, ""))
	t.CollectionName, _ = ParseExpression(collectionMarkup, parseContext.stringScanner, parseContext.expressionCache)

	return t, nil
}

func (t *TableRow) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "endtablerow" {
		return false, nil
	}
	return t.Block.UnknownTag(tag, markup)
}

func (t *TableRow) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	collection := context.Evaluate(t.CollectionName)
	if collection == nil {
		return nil
	}

	segment := SliceCollection(collection, nil, nil)
	length := len(segment)

	cols := length // por defecto una sola fila
	if t.Cols != nil {
		if n, err := UtilsToInteger(context.Evaluate(t.Cols)); err == nil && n > 0 {
			cols = n
		}
	}

	output.WriteString("<tr class=\"row1\">\n")

	err := context.Stack(nil, func() error {
		for i, item := range segment {
			col := (i % cols) + 1  // 1-based
			row := (i / cols) + 1  // 1-based

			context.Scopes[0][t.VariableName] = item
			context.Scopes[0]["tablerow"] = map[string]interface{}{
				"col":    col,
				"col0":   col - 1,
				"row":    row,
				"first":  i == 0,
				"last":   i == length-1,
				"length": length,
				"index":  i + 1,
				"index0": i,
			}

			output.WriteString(fmt.Sprintf("<td class=\"col%d\">", col))

			err := t.Block.RenderToOutputBuffer(context, output)
			if err != nil {
				return err
			}

			output.WriteString("</td>")

			if col == cols && i+1 < length {
				output.WriteString(fmt.Sprintf("</tr>\n<tr class=\"row%d\">\n", row+1))
			}
		}
		return nil
	})

	output.WriteString("</tr>\n")
	return err
}
