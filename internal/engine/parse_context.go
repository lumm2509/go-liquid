package engine

import (
	"sync"

	"github.com/go-liquid/internal/parser"
)

// ParsedPartial is a partial template resolved and parsed at parse-time.
// Stored in ParseContext.parsedPartials to avoid re-parsing on every render.
type ParsedPartial struct {
	Name string
	Root *Document
}

// ParseContext holds all parse-time state for a single template parse pass.
type ParseContext struct {
	Locale         *I18n
	LineNumber     int
	TrimWhitespace bool
	Depth          int
	Partial        bool
	Warnings       []error
	ErrorMode      string
	Environment    EnvironmentIface

	templateOptions map[string]interface{}
	options         map[string]interface{}
	stringScanner   *StringScanner
	expressionCache map[string]interface{}
	partialOptions  map[string]interface{}

	partialsMu    sync.RWMutex
	parsedPartials map[string]*ParsedPartial

	partialStateMu sync.Mutex // guards Partial, options, ErrorMode in SetPartial
}

// NewParseContext constructs a ParseContext from an options map.
// The "environment" key must hold an EnvironmentIface value.
func NewParseContext(options map[string]interface{}) *ParseContext {
	optsCopy := make(map[string]interface{}, len(options))
	for k, v := range options {
		optsCopy[k] = v
	}
	options = optsCopy

	var env EnvironmentIface
	if e, ok := options["environment"].(EnvironmentIface); ok {
		env = e
	}

	pc := &ParseContext{
		Environment:     env,
		templateOptions: options,
		Warnings:        []error{},
		stringScanner:   NewStringScanner(""),
		Depth:           0,
		Partial:         false,
		parsedPartials:  make(map[string]*ParsedPartial),
	}

	if loc, ok := options["locale"].(*I18n); ok {
		pc.Locale = loc
	} else {
		pc.Locale = NewI18n("")
		pc.templateOptions["locale"] = pc.Locale
	}

	pc.setupExpressionCache(options)
	pc.options = pc.templateOptions

	if env != nil {
		pc.ErrorMode = env.GetErrorMode()
	}
	if mode, ok := options["error_mode"].(string); ok {
		pc.ErrorMode = mode
	}

	return pc
}

func (pc *ParseContext) Get(key string) interface{} { return pc.options[key] }

// NewParser resets the shared StringScanner to input and returns a Parser
func (pc *ParseContext) NewParser(input string) *Parser {
	pc.stringScanner.SetString(input)
	return parser.NewParser(pc.stringScanner)
}

func (pc *ParseContext) NewTokenizer(source string, startLine int, forLiquidTag bool) *Tokenizer {
	return parser.NewTokenizer(source, startLine, forLiquidTag)
}

func (pc *ParseContext) ParseExpression(markup string) (interface{}, error) {
	return ParseExpression(markup, pc.stringScanner, pc.expressionCache)
}

// ParseExpressionFromTokens skips the tokensToMarkup round-trip; single tokens resolve without allocation
func (pc *ParseContext) ParseExpressionFromTokens(tokens []Token) (interface{}, error) {
	if len(tokens) == 1 {
		t := tokens[0]
		switch t.Type {
		case StringToken:
			if len(t.Value) >= 2 {
				return t.Value[1 : len(t.Value)-1], nil
			}
			return t.Value, nil
		case NumberToken:
			if num, ok := parseNumber(t.Value); ok {
				return num, nil
			}
		case IdToken:
			if val, ok := Literals[t.Value]; ok {
				return val, nil
			}
			if t.Value == "blank" || t.Value == "empty" {
				return t.Value, nil
			}
		}
	}
	// fall back to string reconstruction for dotted paths and complex expressions
	return pc.ParseExpression(tokensToMarkup(tokens))
}

func (pc *ParseContext) SafeParseExpression(p *Parser) (interface{}, error) {
	return ParseExpressionSafe(p, pc.stringScanner, pc.expressionCache)
}

func (pc *ParseContext) LineNo() int    { return pc.LineNumber }
func (pc *ParseContext) GetErrorMode() string { return pc.ErrorMode }

// SetPartial switches option scoping for partial rendering
func (pc *ParseContext) SetPartial(isPartial bool) {
	pc.partialStateMu.Lock()
	defer pc.partialStateMu.Unlock()
	pc.Partial = isPartial
	if isPartial {
		pc.options = pc.getPartialOptions()
	} else {
		pc.options = pc.templateOptions
	}
	if mode, ok := pc.options["error_mode"].(string); ok {
		pc.ErrorMode = mode
	} else if pc.Environment != nil {
		pc.ErrorMode = pc.Environment.GetErrorMode()
	}
}

func (pc *ParseContext) getPartialOptions() map[string]interface{} {
	if pc.partialOptions != nil {
		return pc.partialOptions
	}

	dontPass := pc.templateOptions["include_options_blacklist"]
	switch v := dontPass.(type) {
	case bool:
		if v {
			pc.partialOptions = map[string]interface{}{"locale": pc.Locale}
			return pc.partialOptions
		}
	case []string:
		pc.partialOptions = make(map[string]interface{})
		blacklist := make(map[string]bool, len(v))
		for _, key := range v {
			blacklist[key] = true
		}
		for k, val := range pc.templateOptions {
			if !blacklist[k] {
				pc.partialOptions[k] = val
			}
		}
		return pc.partialOptions
	}

	pc.partialOptions = pc.templateOptions
	return pc.partialOptions
}

func (pc *ParseContext) GetParsedPartial(key string) (*ParsedPartial, bool) {
	pc.partialsMu.RLock()
	defer pc.partialsMu.RUnlock()
	p, ok := pc.parsedPartials[key]
	return p, ok
}

func (pc *ParseContext) SetParsedPartial(key string, p *ParsedPartial) {
	pc.partialsMu.Lock()
	defer pc.partialsMu.Unlock()
	pc.parsedPartials[key] = p
}

func (pc *ParseContext) setupExpressionCache(options map[string]interface{}) {
	cacheOption := options["expression_cache"]
	if cacheOption == nil {
		pc.expressionCache = make(map[string]interface{})
		return
	}
	if cache, ok := cacheOption.(map[string]interface{}); ok {
		pc.expressionCache = cache
		return
	}
	pc.expressionCache = make(map[string]interface{})
}
