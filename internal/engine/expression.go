package engine

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	Literals = map[string]interface{}{
		"nil":   nil,
		"null":  nil,
		"true":  true,
		"false": false,
	}

	RangesRegex  = regexp.MustCompile(`^\(\s*(\S+)\s*\.\.\s*(\S+)\s*\)$`)
	IntegerRegex = regexp.MustCompile(`^(-?\d+)$`)
	FloatRegex   = regexp.MustCompile(`^(-?\d+)\.\d+$`)
)

func ParseExpression(markup string, ss *StringScanner, cache map[string]interface{}) (interface{}, error) {
	if markup == "" {
		return nil, nil
	}

	markup = strings.TrimSpace(markup)

	if (strings.HasPrefix(markup, "\"") && strings.HasSuffix(markup, "\"")) ||
		(strings.HasPrefix(markup, "'") && strings.HasSuffix(markup, "'")) {
		if len(markup) < 2 {
			return markup, nil
		}
		return markup[1 : len(markup)-1], nil
	}

	if val, ok := Literals[markup]; ok {
		return val, nil
	}

	if cache != nil {
		if val, ok := cache[markup]; ok {
			return val, nil
		}
		res, err := innerParse(markup, ss, cache)
		if err == nil {
			cache[markup] = res
		}
		return res, err
	}

	return innerParse(markup, ss, nil)
}

func ParseExpressionSafe(p *Parser, ss *StringScanner, cache map[string]interface{}) (interface{}, error) {
	expr, err := p.Expression()
	if err != nil {
		return nil, err
	}
	return ParseExpression(expr, ss, cache)
}

func innerParse(markup string, ss *StringScanner, cache map[string]interface{}) (interface{}, error) {
	if strings.HasPrefix(markup, "(") && strings.HasSuffix(markup, ")") {
		matches := RangesRegex.FindStringSubmatch(markup)
		if matches != nil {
			return ParseRangeLookup(matches[1], matches[2], ss, cache)
		}
	}

	if num, ok := parseNumber(markup); ok {
		return num, nil
	}

	if markup == "blank" || markup == "empty" {
		return markup, nil
	}

	return NewVariableLookup(markup, ss, cache), nil
}

func parseNumber(markup string) (interface{}, bool) {
	if IntegerRegex.MatchString(markup) {
		i, err := strconv.Atoi(markup)
		if err == nil {
			return i, true
		}
	}
	if FloatRegex.MatchString(markup) {
		f, err := strconv.ParseFloat(markup, 64)
		if err == nil {
			return f, true
		}
	}
	return nil, false
}
