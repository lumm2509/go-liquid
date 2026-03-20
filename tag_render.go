package liquid

import (
	"fmt"
	"regexp"
	"strings"
)

var RenderSyntax = regexp.MustCompile(`\s*(?:(['"])([^'"]+)(['"])|([^\s,]+))(?:\s*,\s*(.*))?`)
var AttrRegex = regexp.MustCompile(`([\w-]+):\s*([^,]+)`)

type Render struct {
	TagBase
	parseContext     *ParseContext
	TemplateName     string
	VariableTemplate interface{}
	VariableName     interface{}
	AliasName        string
	IsForLoop        bool
	Attributes       map[string]interface{}
}

func NewRender(tagName string, markup string, parseContext *ParseContext) (Tag, error) {
	r := &Render{
		TagBase:      NewTagBase(tagName, markup, parseContext),
		parseContext: parseContext,
		Attributes:   make(map[string]interface{}),
	}

	matches := RenderSyntax.FindStringSubmatch(markup)
	if matches != nil {
		if matches[1] != "" {
			r.TemplateName = matches[2]
		} else {
			expr, _ := ParseExpression(matches[4], parseContext.stringScanner, parseContext.expressionCache)
			r.VariableTemplate = expr
		}

		rest := matches[5]
		if rest != "" {
			if strings.HasPrefix(rest, "with ") || strings.HasPrefix(rest, "for ") {
				parts := strings.Fields(rest)
				if parts[0] == "for" {
					r.IsForLoop = true
				}
				asSplit := strings.Split(rest[len(parts[0])+1:], " as ")
				varPart := asSplit[0]
				attrPart := ""
				if commaIdx := strings.Index(varPart, ","); commaIdx != -1 {
					attrPart = varPart[commaIdx+1:]
					varPart = varPart[:commaIdx]
				}
				r.VariableName, _ = ParseExpression(strings.TrimSpace(varPart), parseContext.stringScanner, parseContext.expressionCache)
				if len(asSplit) > 1 {
					aliasAndAttrs := asSplit[1]
					if commaIdx := strings.Index(aliasAndAttrs, ","); commaIdx != -1 {
						r.AliasName = strings.TrimSpace(aliasAndAttrs[:commaIdx])
						attrPart = aliasAndAttrs[commaIdx+1:]
					} else {
						r.AliasName = strings.TrimSpace(aliasAndAttrs)
					}
				}
				if attrPart != "" {
					r.parseAttributes(attrPart, parseContext)
				}
			} else {
				r.parseAttributes(rest, parseContext)
			}
		}
	} else {
		return nil, fmt.Errorf("Syntax Error in 'render' - Invalid markup: %s", markup)
	}

	return r, nil
}

func (r *Render) parseAttributes(markup string, ctx *ParseContext) {
	matches := AttrRegex.FindAllStringSubmatch(markup, -1)
	for _, m := range matches {
		key := m[1]
		valMarkup := strings.TrimSpace(m[2])
		expr, _ := ParseExpression(valMarkup, ctx.stringScanner, ctx.expressionCache)
		r.Attributes[key] = expr
	}
}

func (r *Render) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	templateName := r.TemplateName
	if r.VariableTemplate != nil {
		val := context.Evaluate(r.VariableTemplate)
		if s, ok := val.(string); ok {
			templateName = s
		} else {
			return nil
		}
	}

	if templateName == "" {
		return nil
	}

	partial, err := LoadPartial(templateName, context, r.parseContext)
	if err != nil {
		return err
	}

	innerContext := context.NewIsolatedSubcontext()
	innerContext.TemplateName = partial.Name
	innerContext.Partial = true

	for k, v := range r.Attributes {
		innerContext.Scopes[0][k] = context.Evaluate(v)
	}

	context_variable_name := r.AliasName
	if context_variable_name == "" {
		parts := strings.Split(templateName, "/")
		context_variable_name = parts[len(parts)-1]
		context_variable_name = strings.ReplaceAll(context_variable_name, "-", "_")
	}

	variable := context.Evaluate(r.VariableName)

	defer context.MergeSubcontext(innerContext)

	if r.IsForLoop {
		segment := SliceCollection(variable, nil, nil)
		for _, item := range segment {
			innerContext.Scopes[0][context_variable_name] = item
			err = partial.Root.RenderToOutputBuffer(innerContext, output)
			if err != nil {
				return err
			}
		}
	} else {
		if variable != nil {
			innerContext.Scopes[0][context_variable_name] = variable
		}
		return partial.Root.RenderToOutputBuffer(innerContext, output)
	}

	return nil
}
