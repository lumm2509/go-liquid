package engine

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Variable is an AST node that evaluates a Liquid variable expression,
// optionally applying a chain of filters.
type Variable struct {
	Markup     string
	Name       interface{}
	Filters    [][]interface{}
	lineNumber int
}

// NamedArguments holds keyword arguments passed to a filter.
type NamedArguments map[string]interface{}

func (n NamedArguments) Evaluate(ctx RenderContext) interface{} {
	result := make(map[string]interface{}, len(n))
	for k, v := range n {
		result[k] = ctx.Evaluate(v)
	}
	return result
}

// NewVariable parses markup into a Variable node.
func NewVariable(markup string, parseContext *ParseContext) *Variable {
	v := &Variable{
		Markup:     markup,
		lineNumber: parseContext.LineNumber,
	}
	v.parse(markup, parseContext)
	return v
}

func (v *Variable) parse(markup string, parseContext *ParseContext) {
	parts := splitByPipeRespectingQuotes(markup)

	namePart := strings.TrimSpace(parts[0])
	name, _ := ParseExpression(namePart, parseContext.stringScanner, parseContext.expressionCache)
	v.Name = name

	v.Filters = make([][]interface{}, 0, len(parts)-1)
	for _, filterPart := range parts[1:] {
		v.Filters = append(v.Filters, v.parseFilter(filterPart, parseContext))
	}
}

// splitByPipeRespectingQuotes splits s on '|' while ignoring pipes inside quotes.
func splitByPipeRespectingQuotes(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		if inQuote {
			current.WriteRune(r)
			if r == quoteChar {
				inQuote = false
			}
		} else {
			switch r {
			case '"', '\'':
				inQuote = true
				quoteChar = r
				current.WriteRune(r)
			case '|':
				parts = append(parts, current.String())
				current.Reset()
			default:
				current.WriteRune(r)
			}
		}
	}
	parts = append(parts, current.String())
	return parts
}

var namedArgRegex = regexp.MustCompile(`^(\w+):\s*(.*)$`)

func (v *Variable) parseFilter(markup string, parseContext *ParseContext) []interface{} {
	markup = strings.TrimSpace(markup)

	filterNameEnd := -1
	inQuote := false
	quoteChar := rune(0)
	for i, r := range markup {
		if inQuote {
			if r == quoteChar {
				inQuote = false
			}
		} else {
			switch r {
			case '"', '\'':
				inQuote = true
				quoteChar = r
			case ':':
				filterNameEnd = i
			}
		}
		if filterNameEnd >= 0 {
			break
		}
	}

	var filterName, argMarkup string
	if filterNameEnd >= 0 {
		filterName = strings.TrimSpace(markup[:filterNameEnd])
		argMarkup = markup[filterNameEnd+1:]
	} else {
		filterName = markup
	}

	args := []interface{}{}
	if argMarkup != "" {
		argParts := splitByCommaRespectingQuotes(argMarkup)
		namedArgs := make(NamedArguments)
		for _, argPart := range argParts {
			argPart = strings.TrimSpace(argPart)
			if matches := namedArgRegex.FindStringSubmatch(argPart); matches != nil {
				expr, _ := ParseExpression(matches[2], parseContext.stringScanner, parseContext.expressionCache)
				namedArgs[matches[1]] = expr
			} else {
				expr, _ := ParseExpression(argPart, parseContext.stringScanner, parseContext.expressionCache)
				args = append(args, expr)
			}
		}
		if len(namedArgs) > 0 {
			args = append(args, namedArgs)
		}
	}

	return []interface{}{filterName, args}
}

func splitByCommaRespectingQuotes(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		if inQuote {
			current.WriteRune(r)
			if r == quoteChar {
				inQuote = false
			}
		} else {
			switch r {
			case '"', '\'':
				inQuote = true
				quoteChar = r
				current.WriteRune(r)
			case ',':
				parts = append(parts, current.String())
				current.Reset()
			default:
				current.WriteRune(r)
			}
		}
	}
	parts = append(parts, current.String())
	return parts
}

// Render evaluates the variable and all its filters, returning the final value.
func (v *Variable) Render(ctx RenderContext) interface{} {
	obj := ctx.Evaluate(v.Name)

	for _, filter := range v.Filters {
		filterName := filter[0].(string)
		filterArgs := filter[1].([]interface{})

		evaluated := make([]interface{}, len(filterArgs))
		for i, arg := range filterArgs {
			evaluated[i] = ctx.Evaluate(arg)
		}
		obj = ctx.InvokeFilter(filterName, obj, evaluated...)
	}

	return ctx.ApplyGlobalFilter(obj)
}

// RenderToOutputBuffer implements Node.
func (v *Variable) RenderToOutputBuffer(ctx RenderContext, output *strings.Builder) error {
	obj := v.Render(ctx)
	renderObjToOutput(obj, output)
	return nil
}

func renderObjToOutput(obj interface{}, output *strings.Builder) {
	if obj == nil {
		return
	}
	switch val := obj.(type) {
	case string:
		output.WriteString(val)
		return
	case int:
		output.WriteString(strconv.Itoa(val))
		return
	case int64:
		output.WriteString(strconv.FormatInt(val, 10))
		return
	case float64:
		output.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
		return
	case bool:
		if val {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
		return
	}

	obj = ToLiquidValue(obj)
	if s, ok := obj.(string); ok {
		output.WriteString(s)
		return
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			renderObjToOutput(rv.Index(i).Interface(), output)
		}
		return
	}

	output.WriteString(fmt.Sprintf("%v", obj))
}

func (v *Variable) IsBlank() bool   { return false }
func (v *Variable) LineNumber() int { return v.lineNumber }
