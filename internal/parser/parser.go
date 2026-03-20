package parser

import (
	"fmt"
	"strings"
)

type Parser struct {
	tokens []Token
	p      int
}

func NewParser(ss *StringScanner) *Parser {
	tokens, _ := Tokenize(ss.String())
	return &Parser{tokens: tokens, p: 0}
}

func (p *Parser) Jump(point int) {
	p.p = point
}

func (p *Parser) Look(tType TokenType, ahead int) bool {
	index := p.p + ahead
	if index >= len(p.tokens) {
		return false
	}
	return p.tokens[index].Type == tType
}

func (p *Parser) Consume(tType TokenType) (string, error) {
	token := p.tokens[p.p]
	if tType != "" && token.Type != tType {
		return "", fmt.Errorf("syntax error: expected %s but found %s", tType, token.Type)
	}
	p.p++
	return token.Value, nil
}

func (p *Parser) ConsumeOptional(tType TokenType) (string, bool) {
	if p.p >= len(p.tokens) {
		return "", false
	}
	token := p.tokens[p.p]
	if token.Type != tType {
		return "", false
	}
	p.p++
	return token.Value, true
}

func (p *Parser) Id(str string) (string, bool) {
	if p.p >= len(p.tokens) {
		return "", false
	}
	token := p.tokens[p.p]
	if token.Type == IdToken && token.Value == str {
		p.p++
		return token.Value, true
	}
	return "", false
}

func (p *Parser) Expression() (string, error) {
	if p.p >= len(p.tokens) {
		return "", fmt.Errorf("unexpected end of input")
	}

	token := p.tokens[p.p]
	var sb strings.Builder

	switch token.Type {
	case IdToken:
		val, _ := p.Consume(IdToken)
		sb.WriteString(val)
		lookups, err := p.VariableLookups()
		if err != nil {
			return "", err
		}
		sb.WriteString(lookups)

	case OpenSquareToken:
		val, _ := p.Consume(OpenSquareToken)
		sb.WriteString(val)

		expr, err := p.Expression()
		if err != nil {
			return "", err
		}
		sb.WriteString(expr)

		closeVal, err := p.Consume(CloseSquareToken)
		if err != nil {
			return "", err
		}
		sb.WriteString(closeVal)

		lookups, err := p.VariableLookups()
		if err != nil {
			return "", err
		}
		sb.WriteString(lookups)

	case StringToken, NumberToken:
		val, _ := p.Consume(token.Type)
		sb.WriteString(val)

	case OpenRoundToken:
		p.Consume(OpenRoundToken)
		first, err := p.Expression()
		if err != nil {
			return "", err
		}
		p.Consume(DotDotToken)
		last, err := p.Expression()
		if err != nil {
			return "", err
		}
		p.Consume(CloseRoundToken)
		return fmt.Sprintf("(%s..%s)", first, last), nil

	default:
		return "", fmt.Errorf("%v is not a valid expression", token)
	}

	return sb.String(), nil
}

func (p *Parser) Argument() (string, error) {
	var sb strings.Builder

	if p.Look(IdToken, 0) && p.Look(ColonToken, 1) {
		key, _ := p.Consume(IdToken)
		colon, _ := p.Consume(ColonToken)
		sb.WriteString(key)
		sb.WriteString(colon)
		sb.WriteString(" ")
	}

	expr, err := p.Expression()
	if err != nil {
		return "", err
	}
	sb.WriteString(expr)

	return sb.String(), nil
}

func (p *Parser) VariableLookups() (string, error) {
	var sb strings.Builder
	for {
		if p.Look(OpenSquareToken, 0) {
			val, _ := p.Consume(OpenSquareToken)
			sb.WriteString(val)

			expr, err := p.Expression()
			if err != nil {
				return "", err
			}
			sb.WriteString(expr)

			closeVal, err := p.Consume(CloseSquareToken)
			if err != nil {
				return "", err
			}
			sb.WriteString(closeVal)

		} else if p.Look(DotToken, 0) {
			dot, _ := p.Consume(DotToken)
			sb.WriteString(dot)

			id, err := p.Consume(IdToken)
			if err != nil {
				return "", err
			}
			sb.WriteString(id)

		} else {
			break
		}
	}
	return sb.String(), nil
}
