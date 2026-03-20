package liquid

import (
	"fmt"
	"strings"
)

type Cycle struct {
	TagBase
	Variables []interface{}
	Name      interface{}
}

func NewCycle(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	c := &Cycle{
		TagBase: NewTagBase(tagName, markup, parseContext),
	}

	tokens, err := Tokenize(markup)
	if err != nil {
		return nil, SyntaxError{BaseError: BaseError{Message: err.Error()}}
	}

	// Filter EOS token
	var toks []Token
	for _, t := range tokens {
		if t.Type != EOSToken {
			toks = append(toks, t)
		}
	}

	// Named group: first token followed by colon — {% cycle "name": "a", "b" %}
	startIdx := 0
	if len(toks) >= 2 && toks[1].Type == ColonToken {
		expr, _ := parseContext.ParseExpression(toks[0].Value)
		c.Name = expr
		startIdx = 2
	}

	// Collect variable expressions, skipping commas
	for i := startIdx; i < len(toks); i++ {
		if toks[i].Type == CommaToken {
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

func (c *Cycle) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	if len(c.Variables) == 0 {
		return nil
	}

	nameVal := context.Evaluate(c.Name)
	key, ok := nameVal.(string)
	if !ok {
		key = fmt.Sprint(nameVal)
	}

	cycleRegistry, ok := context.Registers.Get("cycle").(map[string]int)
	if !ok {
		cycleRegistry = make(map[string]int)
		context.Registers.Set("cycle", cycleRegistry)
	}

	iteration := cycleRegistry[key]
	val := context.Evaluate(c.Variables[iteration])

	output.WriteString(UtilsToString(val))

	iteration = (iteration + 1) % len(c.Variables)
	cycleRegistry[key] = iteration

	return nil
}
