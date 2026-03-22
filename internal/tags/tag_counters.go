package tags

import (
	"fmt"
	"strings"

	"github.com/go-liquid/internal/engine"
)

const countersKey = "counters"

func getCounters(ctx engine.RenderContext) map[string]int {
	if c, ok := ctx.RegisterGet(countersKey).(map[string]int); ok {
		return c
	}
	c := make(map[string]int)
	ctx.RegisterSet(countersKey, c)
	return c
}

type Increment struct {
	engine.TagBase
	VariableName string
}

func NewIncrement(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Increment{
		TagBase:      engine.NewTagBase(tagName, markup, parseContext),
		VariableName: strings.TrimSpace(markup),
	}, nil
}

func (i *Increment) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	counters := getCounters(ctx)
	val := counters[i.VariableName]
	output.WriteString(fmt.Sprintf("%d", val))
	counters[i.VariableName] = val + 1
	return nil
}

type Decrement struct {
	engine.TagBase
	VariableName string
}

func NewDecrement(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	return &Decrement{
		TagBase:      engine.NewTagBase(tagName, markup, parseContext),
		VariableName: strings.TrimSpace(markup),
	}, nil
}

func (d *Decrement) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	counters := getCounters(ctx)
	val := counters[d.VariableName] - 1
	output.WriteString(fmt.Sprintf("%d", val))
	counters[d.VariableName] = val
	return nil
}
