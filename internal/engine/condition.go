package engine

import (
	"reflect"
	"strings"
)

// Operator is a comparison function used by Condition.
type Operator func(cond *Condition, left, right interface{}) bool

// Operators maps operator strings to their implementations.
var Operators = map[string]Operator{
	"==": func(cond *Condition, left, right interface{}) bool { return cond.EqualVariables(left, right) },
	"!=": func(cond *Condition, left, right interface{}) bool { return !cond.EqualVariables(left, right) },
	"<>": func(cond *Condition, left, right interface{}) bool { return !cond.EqualVariables(left, right) },
	"<":  func(cond *Condition, left, right interface{}) bool { return CompareValues(left, right) < 0 },
	">":  func(cond *Condition, left, right interface{}) bool { return CompareValues(left, right) > 0 },
	"<=": func(cond *Condition, left, right interface{}) bool { return CompareValues(left, right) <= 0 },
	">=": func(cond *Condition, left, right interface{}) bool { return CompareValues(left, right) >= 0 },
	"contains": func(cond *Condition, left, right interface{}) bool {
		if left == nil || right == nil {
			return false
		}
		rv := reflect.ValueOf(left)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				if reflect.DeepEqual(rv.Index(i).Interface(), right) {
					return true
				}
			}
			return false
		}
		if rv.Kind() == reflect.String {
			return strings.Contains(left.(string), UtilsToString(right))
		}
		return false
	},
}

// Condition is a parsed boolean expression node used by if/unless/case.
type Condition struct {
	Left           interface{}
	Operator       string
	Right          interface{}
	ChildRelation  string
	ChildCondition *Condition
	Attachment     *BlockBody
}

func NewCondition(left interface{}, operator string, right interface{}) *Condition {
	return &Condition{Left: left, Operator: operator, Right: right}
}

func NewElseCondition() *Condition { return &Condition{} }

func (c *Condition) IsElse() bool {
	return c.Operator == "" && c.Left == nil && c.Right == nil
}

func (c *Condition) Or(condition *Condition) {
	c.ChildRelation = "or"
	c.ChildCondition = condition
}

func (c *Condition) And(condition *Condition) {
	c.ChildRelation = "and"
	c.ChildCondition = condition
}

// ParseCondition tokenizes markup and builds a Condition chain.
func ParseCondition(markup string, parseContext TagParseContext) (*Condition, error) {
	if markup == "" {
		return nil, nil
	}
	tokens, err := Tokenize(markup)
	if err != nil {
		return nil, err
	}
	return parseRecursive(tokens, parseContext)
}

func parseRecursive(tokens []Token, parseContext TagParseContext) (*Condition, error) {
	if len(tokens) == 0 || tokens[0].Type == EOSToken {
		return nil, nil
	}

	var leftMarkup, op string
	var right interface{}
	var rel string
	relIdx := -1
	p := 0

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.Type == ComparisonToken {
			op = t.Value
			leftMarkup = tokensToMarkup(tokens[:i])
			p = i + 1
			for j := p; j < len(tokens); j++ {
				if tokens[j].Type == IdToken && (tokens[j].Value == "and" || tokens[j].Value == "or") {
					rm := tokensToMarkup(tokens[p:j])
					right, _ = parseContext.ParseExpression(rm)
					rel = tokens[j].Value
					relIdx = j
					break
				}
				if tokens[j].Type == EOSToken {
					rm := tokensToMarkup(tokens[p:j])
					right, _ = parseContext.ParseExpression(rm)
					break
				}
			}
			break
		}
		if t.Type == IdToken && (t.Value == "and" || t.Value == "or") {
			leftMarkup = tokensToMarkup(tokens[:i])
			rel = t.Value
			relIdx = i
			break
		}
		if t.Type == EOSToken {
			leftMarkup = tokensToMarkup(tokens[:i])
			break
		}
	}

	left, _ := parseContext.ParseExpression(leftMarkup)
	cond := NewCondition(left, op, right)

	if rel != "" {
		cond.ChildRelation = rel
		nextCond, _ := parseRecursive(tokens[relIdx+1:], parseContext)
		cond.ChildCondition = nextCond
	}

	return cond, nil
}

func tokensToMarkup(tokens []Token) string {
	var sb strings.Builder
	for i, t := range tokens {
		if i > 0 {
			prev := tokens[i-1]
			if t.Type != DotToken && prev.Type != DotToken {
				sb.WriteString(" ")
			}
		}
		sb.WriteString(t.Value)
	}
	return sb.String()
}

// Evaluate evaluates the condition chain against ctx.
func (c *Condition) Evaluate(ctx *Context) bool {
	condition := c
	result := false

	for condition != nil {
		currentResult := condition.interpretCondition(condition.Left, condition.Right, condition.Operator, ctx)

		if condition.ChildRelation == "and" {
			result = currentResult
			if !result {
				for condition != nil && condition.ChildRelation != "or" {
					condition = condition.ChildCondition
				}
				if condition == nil {
					return false
				}
				result = false
			}
		} else if condition.ChildRelation == "or" {
			result = currentResult
			if result {
				return true
			}
		} else {
			result = currentResult
			break
		}
		condition = condition.ChildCondition
	}
	return result
}

func (c *Condition) interpretCondition(left, right interface{}, op string, ctx *Context) bool {
	if op == "" {
		return IsTruthy(ctx.Evaluate(left))
	}

	leftVal := ctx.Evaluate(left)
	rightVal := ctx.Evaluate(right)

	operation, ok := Operators[op]
	if !ok {
		if ctx.Environment != nil {
			if logger := ctx.Environment.GetLogger(); logger != nil {
				logger.Log(DebugEvent{
					Event: "condition.unknown_operator",
					Data:  map[string]interface{}{"operator": op},
				})
			}
		}
		return false
	}
	return operation(c, maybeLiquidValue(leftVal), maybeLiquidValue(rightVal))
}

func (c *Condition) EqualVariables(left, right interface{}) bool {
	if left == right {
		return true
	}
	if isBlankSymbol(left) {
		return isBlankValue(right)
	}
	if isBlankSymbol(right) {
		return isBlankValue(left)
	}
	if left == nil && right == nil {
		return true
	}
	return reflect.DeepEqual(left, right)
}

// maybeLiquidValue calls ToLiquidValue only for non-primitive types.
func maybeLiquidValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch v.(type) {
	case string, int, int64, float64, bool:
		return v
	}
	return ToLiquidValue(v)
}

func isBlankSymbol(v interface{}) bool {
	s, ok := v.(string)
	return ok && (s == "blank" || s == "empty")
}

func isBlankValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return len(strings.TrimSpace(rv.String())) == 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() == 0
	case reflect.Bool:
		return !v.(bool)
	}
	return false
}
