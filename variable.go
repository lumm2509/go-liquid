package liquid

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Variable struct {
	Markup       string
	Name         interface{}
	Filters      [][]interface{}
	lineNumber   int
	ParseContext *ParseContext
}

type NamedArguments map[string]interface{}

func (n NamedArguments) Evaluate(ctx *Context) interface{} {
	result := make(map[string]interface{})
	for k, v := range n {
		result[k] = ctx.Evaluate(v)
	}
	return result
}

func NewVariable(markup string, parseContext *ParseContext) *Variable {
	v := &Variable{
		Markup:       markup,
		ParseContext: parseContext,
		lineNumber:   parseContext.LineNumber,
	}
	v.Parse(markup)
	return v
}

func (v *Variable) Parse(markup string) {
	// Simple parser for variables with filters
	// In Ruby: strict_parse_with_error_mode_fallback(markup)

	// Split markup by pipe '|' respecting quotes
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range markup {
		if inQuote {
			current.WriteRune(r)
			if r == quoteChar {
				inQuote = false
			}
		} else {
			if r == '"' || r == '\'' {
				inQuote = true
				quoteChar = r
				current.WriteRune(r)
			} else if r == '|' {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		}
	}
	if current.Len() > 0 || len(parts) > 0 {
		parts = append(parts, current.String())
	}

	namePart := strings.TrimSpace(parts[0])

	name, _ := ParseExpression(namePart, v.ParseContext.stringScanner, v.ParseContext.expressionCache)
	v.Name = name

	v.Filters = make([][]interface{}, 0)
	for _, filterPart := range parts[1:] {
		v.Filters = append(v.Filters, v.parseFilter(filterPart))
	}
}

var NamedArgRegex = regexp.MustCompile(`^(\w+):\s*(.*)$`)

func (v *Variable) parseFilter(markup string) []interface{} {
	markup = strings.TrimSpace(markup)

	// Buscar el primer ":" que NO esté dentro de comillas
	// Similar a QuotedFragment en Ruby que ignora separadores dentro de comillas
	var filterNameEnd int = -1
	inQuote := false
	quoteChar := rune(0)

	for i, r := range markup {
		if inQuote {
			if r == quoteChar {
				inQuote = false
			}
		} else {
			if r == '"' || r == '\'' {
				inQuote = true
				quoteChar = r
			} else if r == ':' {
				filterNameEnd = i
				break
			}
		}
	}

	var filterName string
	var argMarkup string

	if filterNameEnd >= 0 {
		filterName = strings.TrimSpace(markup[:filterNameEnd])
		argMarkup = markup[filterNameEnd+1:]
	} else {
		filterName = markup
		argMarkup = ""
	}

	args := []interface{}{}
	if argMarkup != "" {
		// Handle comma-separated arguments respecting quotes
		var argParts []string
		var current strings.Builder
		inQuote = false
		quoteChar = rune(0)

		for _, r := range argMarkup {
			if inQuote {
				current.WriteRune(r)
				if r == quoteChar {
					inQuote = false
				}
			} else {
				if r == '"' || r == '\'' {
					inQuote = true
					quoteChar = r
					current.WriteRune(r)
				} else if r == ',' {
					argParts = append(argParts, current.String())
					current.Reset()
				} else {
					current.WriteRune(r)
				}
			}
		}
		if current.Len() > 0 || len(argParts) > 0 { // Append last part if exists or if we had separators
			argParts = append(argParts, current.String())
		}

		namedArgs := make(NamedArguments)
		for _, argPart := range argParts {
			argPart = strings.TrimSpace(argPart)

			// Check for named argument
			if matches := NamedArgRegex.FindStringSubmatch(argPart); matches != nil {
				key := matches[1]
				valMarkup := matches[2]
				expr, _ := ParseExpression(valMarkup, v.ParseContext.stringScanner, v.ParseContext.expressionCache)
				namedArgs[key] = expr
			} else {
				expr, _ := ParseExpression(argPart, v.ParseContext.stringScanner, v.ParseContext.expressionCache)
				args = append(args, expr)
			}
		}

		if len(namedArgs) > 0 {
			args = append(args, namedArgs)
		}
	}

	return []interface{}{filterName, args}
}

func (v *Variable) Render(context *Context) interface{} {
	obj := context.Evaluate(v.Name)

	for _, filter := range v.Filters {
		filterName := filter[0].(string)
		filterArgs := filter[1].([]interface{})

		evaluatedArgs := make([]interface{}, len(filterArgs))
		for i, arg := range filterArgs {
			evaluatedArgs[i] = context.Evaluate(arg)
		}

		obj = context.invoke(filterName, obj, evaluatedArgs...)
	}

	return context.ApplyGlobalFilter(obj)
}

func (v *Variable) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	obj := v.Render(context)
	v.renderObjToOutput(obj, output)
	return nil
}

func (v *Variable) renderObjToOutput(obj interface{}, output *strings.Builder) {
	if obj == nil {
		return
	}

	// Fast path for common primitive types — avoids fmt.Sprintf allocation
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

	// Usar reflexión para slices/arrays
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			v.renderObjToOutput(rv.Index(i).Interface(), output)
		}
		return
	}

	output.WriteString(fmt.Sprintf("%v", obj))
}

func (v *Variable) IsBlank() bool {
	return false
}

func (v *Variable) LineNumber() int {
	return v.lineNumber
}
