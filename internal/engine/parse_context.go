package engine

import "github.com/go-liquid/internal/parser"

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

// Get returns an option value by key.
func (pc *ParseContext) Get(key string) interface{} { return pc.options[key] }

// NewParser resets the shared StringScanner to input and returns a Parser.
func (pc *ParseContext) NewParser(input string) *Parser {
	pc.stringScanner.SetString(input)
	return parser.NewParser(pc.stringScanner)
}

// NewTokenizer constructs a Tokenizer from source. Implements TagParseContext.
func (pc *ParseContext) NewTokenizer(source string, startLine int, forLiquidTag bool) *Tokenizer {
	return parser.NewTokenizer(source, startLine, forLiquidTag)
}

// ParseExpression parses a markup string into an evaluatable expression.
// Implements TagParseContext.
func (pc *ParseContext) ParseExpression(markup string) (interface{}, error) {
	return ParseExpression(markup, pc.stringScanner, pc.expressionCache)
}

// SafeParseExpression parses the next expression from a Parser.
func (pc *ParseContext) SafeParseExpression(p *Parser) (interface{}, error) {
	return ParseExpressionSafe(p, pc.stringScanner, pc.expressionCache)
}

// LineNo returns the current parse line number. Implements TagParseContext.
func (pc *ParseContext) LineNo() int { return pc.LineNumber }

// SetPartial switches option scoping for partial rendering.
func (pc *ParseContext) SetPartial(isPartial bool) {
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
