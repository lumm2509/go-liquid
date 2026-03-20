package liquid

import (
	"fmt"
	"strings"
)

// countersKey is the Registers key for increment/decrement counters.
// Counters live in Registers (per-render state), NOT in context.Environments,
// so they never mutate the user's assigns map.
const countersKey = "counters"

func getCounters(context *Context) map[string]int {
	if c, ok := context.Registers.Get(countersKey).(map[string]int); ok {
		return c
	}
	c := make(map[string]int)
	context.Registers.Set(countersKey, c)
	return c
}

type Increment struct {
	TagBase
	VariableName string
}

func NewIncrement(tagName string, markup string, parseContext *ParseContext) (Tag, error) {
	return &Increment{
		TagBase:      NewTagBase(tagName, markup, parseContext),
		VariableName: strings.TrimSpace(markup),
	}, nil
}

func (i *Increment) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	counters := getCounters(context)
	val := counters[i.VariableName]
	output.WriteString(fmt.Sprintf("%d", val))
	counters[i.VariableName] = val + 1
	return nil
}

type Decrement struct {
	TagBase
	VariableName string
}

func NewDecrement(tagName string, markup string, parseContext *ParseContext) (Tag, error) {
	return &Decrement{
		TagBase:      NewTagBase(tagName, markup, parseContext),
		VariableName: strings.TrimSpace(markup),
	}, nil
}

func (d *Decrement) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	counters := getCounters(context)
	val := counters[d.VariableName] - 1
	output.WriteString(fmt.Sprintf("%d", val))
	counters[d.VariableName] = val
	return nil
}
