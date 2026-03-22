package tags

import (
	"fmt"
	"strings"

	"github.com/go-liquid/internal/engine"
)

type Cycle struct {
	engine.TagBase
	Variables []interface{}
	Name      interface{}
}

func NewCycle(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	c := &Cycle{TagBase: engine.NewTagBase(tagName, markup, parseContext)}

	tokens, err := engine.Tokenize(markup)
	if err != nil {
		return nil, engine.SyntaxError{BaseError: engine.BaseError{Message: err.Error()}}
	}

	var toks []engine.Token
	for _, t := range tokens {
		if t.Type != engine.EOSToken {
			toks = append(toks, t)
		}
	}

	startIdx := 0
	if len(toks) >= 2 && toks[1].Type == engine.ColonToken {
		expr, _ := parseContext.ParseExpression(toks[0].Value)
		c.Name = expr
		startIdx = 2
	}

	for i := startIdx; i < len(toks); i++ {
		if toks[i].Type == engine.CommaToken {
			continue
		}
		expr, _ := parseContext.ParseExpression(toks[i].Value)
		c.Variables = append(c.Variables, expr)
	}

	if c.Name == nil {
		c.Name = fmt.Sprintf("%v", c.Variables)
	}

	return c, nil
}

func (c *Cycle) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	if len(c.Variables) == 0 {
		return nil
	}

	nameVal := ctx.Evaluate(c.Name)
	key, ok := nameVal.(string)
	if !ok {
		key = fmt.Sprint(nameVal)
	}

	cycleRegistry, ok := ctx.RegisterGet("cycle").(map[string]int)
	if !ok {
		cycleRegistry = make(map[string]int)
		ctx.RegisterSet("cycle", cycleRegistry)
	}

	iteration := cycleRegistry[key]
	val := ctx.Evaluate(c.Variables[iteration])
	output.WriteString(engine.UtilsToString(val))

	cycleRegistry[key] = (iteration + 1) % len(c.Variables)
	return nil
}
