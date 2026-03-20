package liquid

import (
	"reflect"
	"strconv"
	"strings"
)

type Operator func(cond *Condition, left, right interface{}) bool

var Operators = map[string]Operator{
	"==": func(cond *Condition, left, right interface{}) bool { return cond.EqualVariables(left, right) },
	"!=": func(cond *Condition, left, right interface{}) bool { return !cond.EqualVariables(left, right) },
	"<>": func(cond *Condition, left, right interface{}) bool { return !cond.EqualVariables(left, right) },
	"<": func(cond *Condition, left, right interface{}) bool {
		return CompareValues(left, right) < 0
	},
	">": func(cond *Condition, left, right interface{}) bool {
		return CompareValues(left, right) > 0
	},
	"<=": func(cond *Condition, left, right interface{}) bool {
		return CompareValues(left, right) <= 0
	},
	">=": func(cond *Condition, left, right interface{}) bool {
		return CompareValues(left, right) >= 0
	},
	"contains": func(cond *Condition, left, right interface{}) bool {
		if left == nil || right == nil {
			return false
		}
		// Check if left contains right
		rv := reflect.ValueOf(left)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				if reflect.DeepEqual(rv.Index(i).Interface(), right) {
					return true
				}
			}
		} else if rv.Kind() == reflect.String {
			return strings.Contains(left.(string), UtilsToString(right))
		}
		return false
	},
}

type Condition struct {
	Left           interface{}
	Operator       string
	Right          interface{}
	ChildRelation  string
	ChildCondition *Condition
	Attachment     *BlockBody
}

func NewCondition(left interface{}, operator string, right interface{}) *Condition {
	return &Condition{
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

func ParseCondition(markup string, parseContext *ParseContext) (*Condition, error) {
	if markup == "" {
		return nil, nil
	}

	// Tokenizar el markup
	tokens, err := Tokenize(markup)
	if err != nil {
		return nil, err
	}

	return parseRecursive(tokens, parseContext)
}

func parseRecursive(tokens []Token, parseContext *ParseContext) (*Condition, error) {
	if len(tokens) == 0 || tokens[0].Type == EOSToken {
		return nil, nil
	}

	p := 0
	// 1. Encontrar el primer operando (Left)
	leftMarkup := ""
	var op string
	var right interface{}
	var rel string
	relIdx := -1

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.Type == ComparisonToken {
			op = t.Value
			leftMarkup = tokensToMarkup(tokens[:i])
			p = i + 1
			// Encontrar el Right
			for j := p; j < len(tokens); j++ {
				if tokens[j].Type == IdToken && (tokens[j].Value == "and" || tokens[j].Value == "or") {
					rightMarkup := tokensToMarkup(tokens[p:j])
					right, _ = ParseExpression(rightMarkup, parseContext.stringScanner, parseContext.expressionCache)
					rel = tokens[j].Value
					relIdx = j
					break
				}
				if tokens[j].Type == EOSToken {
					rightMarkup := tokensToMarkup(tokens[p:j])
					right, _ = ParseExpression(rightMarkup, parseContext.stringScanner, parseContext.expressionCache)
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

	left, _ := ParseExpression(leftMarkup, parseContext.stringScanner, parseContext.expressionCache)
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
			// No añadir espacio si es un punto o si el anterior era un punto
			if t.Type != DotToken && prev.Type != DotToken {
				sb.WriteString(" ")
			}
		}
		sb.WriteString(t.Value)
	}
	return sb.String()
}

func (c *Condition) Evaluate(context *Context) bool {
	condition := c
	result := false

	for condition != nil {
		currentResult := condition.interpretCondition(condition.Left, condition.Right, condition.Operator, context)

		if condition.ChildRelation == "and" {
			result = currentResult
			if !result {
				// Saltamos hasta el próximo "or" si existe
				for condition != nil && condition.ChildRelation != "or" {
					condition = condition.ChildCondition
				}
				if condition == nil {
					return false
				}
				result = false // Empezamos de nuevo con el OR
			}
		} else if condition.ChildRelation == "or" {
			result = currentResult
			if result {
				return true // Cortocircuito para OR
			}
		} else {
			result = currentResult
			break
		}
		condition = condition.ChildCondition
	}
	return result
}

func (c *Condition) interpretCondition(left, right interface{}, op string, context *Context) bool {
	if op == "" {
		val := context.Evaluate(left)
		return IsTruthy(val)
	}

	leftVal := context.Evaluate(left)
	rightVal := context.Evaluate(right)

	operation, ok := Operators[op]
	if !ok {
		if context.Environment != nil && context.Environment.Logger != nil {
			context.Environment.Logger.Log(DebugEvent{
				Event: "condition.unknown_operator",
				Data:  map[string]interface{}{"operator": op},
			})
		}
		return false
	}

	// ToLiquidValue only does work for map types; skip it for primitives
	// to avoid unnecessary reflect.ValueOf calls on every condition evaluation.
	return operation(c, maybeLiquidValue(leftVal), maybeLiquidValue(rightVal))
}

func (c *Condition) EqualVariables(left, right interface{}) bool {
	if left == right {
		return true
	}

	// Si uno de los dos es un símbolo "blank" o "empty"
	if isBlankSymbol(left) {
		return isBlankValue(right)
	}
	if isBlankSymbol(right) {
		return isBlankValue(left)
	}

	// Si ambos son nil (aunque el primer check ya debería cubrirlo)
	if left == nil && right == nil {
		return true
	}

	return reflect.DeepEqual(left, right)
}

// maybeLiquidValue calls ToLiquidValue only for map types.
// For primitives (string, int, bool, nil), it returns the value unchanged,
// avoiding reflect.ValueOf on every condition evaluation.
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

func (c *Condition) Or(condition *Condition) {
	c.ChildRelation = "or"
	c.ChildCondition = condition
}

func (c *Condition) And(condition *Condition) {
	c.ChildRelation = "and"
	c.ChildCondition = condition
}

// toFloat64 se mantiene para uso interno en filtros numéricos (Plus, Minus, etc.)
func toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

type ElseCondition struct {
	Condition
}

func NewElseCondition() *Condition {
	return &Condition{}
}

func (c *Condition) IsElse() bool {
	return c.Operator == "" && c.Left == nil && c.Right == nil
}
