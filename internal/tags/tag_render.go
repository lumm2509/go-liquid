package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-liquid/internal/engine"
)

var renderSyntax = regexp.MustCompile(`\s*(?:(['"])([^'"]+)(['"])|([^\s,]+))(?:\s*,\s*(.*))?`)
var attrRegex = regexp.MustCompile(`([\w-]+):\s*([^,]+)`)

type Render struct {
	engine.TagBase
	parseContext     engine.TagParseContext
	TemplateName     string
	VariableTemplate interface{}
	VariableName     interface{}
	AliasName        string
	IsForLoop        bool
	Attributes       map[string]interface{}
}

func NewRender(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	r := &Render{
		TagBase:      engine.NewTagBase(tagName, markup, parseContext),
		parseContext: parseContext,
		Attributes:   make(map[string]interface{}),
	}

	matches := renderSyntax.FindStringSubmatch(markup)
	if matches == nil {
		return nil, fmt.Errorf("Syntax Error in 'render' - Invalid markup: %s", markup)
	}

	if matches[1] != "" {
		r.TemplateName = matches[2]
	} else {
		r.VariableTemplate, _ = parseContext.ParseExpression(matches[4])
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
			r.VariableName, _ = parseContext.ParseExpression(strings.TrimSpace(varPart))
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

	return r, nil
}

func (r *Render) parseAttributes(markup string, ctx engine.TagParseContext) {
	for _, m := range attrRegex.FindAllStringSubmatch(markup, -1) {
		expr, _ := ctx.ParseExpression(strings.TrimSpace(m[2]))
		r.Attributes[m[1]] = expr
	}
}

// PreloadPartial implements engine.StaticPartialLoader.
func (r *Render) PreloadPartial(pc engine.TagParseContext) error {
	if r.TemplateName == "" {
		return nil // dynamic template name — cannot preload at parse time
	}
	_, err := loadPartialAtParseTime(r.TemplateName, pc)
	return err
}

func (r *Render) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	templateName := r.TemplateName
	if r.VariableTemplate != nil {
		val := ctx.Evaluate(r.VariableTemplate)
		s, ok := val.(string)
		if !ok {
			return nil
		}
		templateName = s
	}
	if templateName == "" {
		return nil
	}

	parsed, err := loadPartial(templateName, ctx, r.parseContext)
	if err != nil {
		return err
	}

	innerCtx := ctx.NewIsolatedSubcontext()
	innerCtx.SetTemplateName(parsed.Name)
	innerCtx.SetPartial(true)

	for k, v := range r.Attributes {
		innerCtx.Set(k, ctx.Evaluate(v))
	}

	varName := r.AliasName
	if varName == "" {
		parts := strings.Split(templateName, "/")
		varName = strings.ReplaceAll(parts[len(parts)-1], "-", "_")
	}

	variable := ctx.Evaluate(r.VariableName)
	defer ctx.MergeSubcontext(innerCtx)

	if r.IsForLoop {
		segment := engine.SliceCollection(variable, nil, nil)
		for _, item := range segment {
			innerCtx.Set(varName, item)
			if err := parsed.Root.RenderToOutputBuffer(innerCtx, output); err != nil {
				return err
			}
		}
		return nil
	}

	if variable != nil {
		innerCtx.Set(varName, variable)
	}
	return parsed.Root.RenderToOutputBuffer(innerCtx, output)
}
