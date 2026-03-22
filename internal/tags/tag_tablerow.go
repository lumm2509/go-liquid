package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lumm2509/go-liquid/internal/engine"
)

var tablerowColsRe = regexp.MustCompile(`\bcols:(\S+)`)
var tablerowAttrStripRe = regexp.MustCompile(`\s+cols:\S+`)

type TableRow struct {
	engine.Block
	VariableName   string
	CollectionName interface{}
	Cols           interface{}
}

func NewTableRow(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	t := &TableRow{Block: engine.NewBlock(tagName, markup, parseContext)}

	matches := ForSyntax.FindStringSubmatch(markup)
	if matches == nil {
		return nil, engine.SyntaxError{BaseError: engine.BaseError{Message: "Syntax Error in 'tablerow' - Valid syntax: tablerow [item] in [collection]"}}
	}

	t.VariableName = matches[1]
	rest := matches[2]

	if m := tablerowColsRe.FindStringSubmatch(rest); m != nil {
		t.Cols, _ = parseContext.ParseExpression(m[1])
	}

	collectionMarkup := strings.TrimSpace(tablerowAttrStripRe.ReplaceAllString(rest, ""))
	t.CollectionName, _ = parseContext.ParseExpression(collectionMarkup)

	return t, nil
}

func (t *TableRow) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "endtablerow" {
		return false, nil
	}
	return t.Block.UnknownTag(tag, markup)
}

func (t *TableRow) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	collection := ctx.Evaluate(t.CollectionName)
	if collection == nil {
		return nil
	}

	segment := engine.SliceCollection(collection, nil, nil)
	length := len(segment)

	cols := length
	if t.Cols != nil {
		if n, err := engine.UtilsToInteger(ctx.Evaluate(t.Cols)); err == nil && n > 0 {
			cols = n
		}
	}

	output.WriteString("<tr class=\"row1\">\n")

	tablerowMap := map[string]interface{}{
		"col": 0, "col0": 0, "row": 0,
		"first": false, "last": false, "length": length,
		"index": 0, "index0": 0,
	}

	err := ctx.Stack(nil, func() error {
		ctx.Set("tablerow", tablerowMap)
		for i, item := range segment {
			col := (i % cols) + 1
			row := (i / cols) + 1

			tablerowMap["col"] = col
			tablerowMap["col0"] = col - 1
			tablerowMap["row"] = row
			tablerowMap["first"] = i == 0
			tablerowMap["last"] = i == length-1
			tablerowMap["index"] = i + 1
			tablerowMap["index0"] = i

			ctx.Set(t.VariableName, item)
			output.WriteString(fmt.Sprintf("<td class=\"col%d\">", col))

			if err := t.Block.RenderToOutputBuffer(ctx, output); err != nil {
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
