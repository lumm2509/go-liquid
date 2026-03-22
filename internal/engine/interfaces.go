package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-liquid/internal/parser"
	"github.com/go-liquid/internal/runtime"
)

// re-exported parser primitives for internal use
type Tokenizer = parser.Tokenizer
type Token = parser.Token
type TokenType = parser.TokenType
type StringScanner = parser.StringScanner
type Parser = parser.Parser

func NewStringScanner(source string) *StringScanner { return parser.NewStringScanner(source) }
func NewParser(ss *StringScanner) *Parser            { return parser.NewParser(ss) }
func Tokenize(input string) ([]Token, error)         { return parser.Tokenize(input) }
func NewTokenizer(source string, startLine int, forLiquidTag bool) *Tokenizer {
	return parser.NewTokenizer(source, startLine, forLiquidTag)
}

const (
	IdToken          = parser.IdToken
	StringToken      = parser.StringToken
	NumberToken      = parser.NumberToken
	DotToken         = parser.DotToken
	DotDotToken      = parser.DotDotToken
	ColonToken       = parser.ColonToken
	CommaToken       = parser.CommaToken
	OpenSquareToken  = parser.OpenSquareToken
	CloseSquareToken = parser.CloseSquareToken
	OpenRoundToken   = parser.OpenRoundToken
	CloseRoundToken  = parser.CloseRoundToken
	PipeToken        = parser.PipeToken
	DashToken        = parser.DashToken
	ComparisonToken  = parser.ComparisonToken
	EOSToken         = parser.EOSToken
)

// DebugEventType is the type for structured debug event identifiers.
// Use the Event* constants instead of raw strings to enable filtering and alerting.
type DebugEventType string

const (
	EventFilterNotFound            DebugEventType = "filter.not_found"
	EventEnvironmentFrozenTag      DebugEventType = "environment.frozen_tag_skipped"
	EventContextOverflow           DebugEventType = "context.overflow"
	EventRenderNodeError           DebugEventType = "render.node_error"
	EventConditionUnknownOperator  DebugEventType = "condition.unknown_operator"
)

// DebugEvent is passed to DebugLogger on each internal event.
type DebugEvent struct {
	Event DebugEventType
	Data  map[string]interface{}
}

// DebugLogger is the observer interface for internal engine events.
type DebugLogger interface {
	Log(event DebugEvent)
}

// FileSystem abstracts template file loading.
type FileSystem interface {
	ReadTemplateFile(templatePath string) (string, error)
}

// ExceptionRenderer transforms render errors before returning them.
type ExceptionRenderer func(error) error

// EnvironmentIface is what engine types need from the environment.
// root.Environment implements this.
type EnvironmentIface interface {
	TagForName(name string) TagFactory
	CreateFilterDispatcher(ctx *Context, filters []interface{}) *FilterDispatcher
	GetExceptionRenderer() ExceptionRenderer
	GetFileSystem() FileSystem
	GetDefaultResourceLimits() runtime.ResourceLimitsConfig
	GetLogger() DebugLogger
	GetErrorMode() string
}

// RenderContext is the interface tags use during rendering.
// The concrete *Context implements this; external custom tags
// should program against RenderContext, not *Context.
type RenderContext interface {
	Get(expression string) interface{}
	Set(key string, value interface{})
	Evaluate(object interface{}) interface{}
	FindVariable(key string, raiseOnNotFound bool) interface{}
	Stack(scope map[string]interface{}, block func() error) error
	Interrupt() bool
	PushInterrupt(i interface{})
	PopInterrupt() interface{}
	HandleError(err error, lineNumber int) string
	RegisterGet(key string) interface{}
	RegisterSet(key string, value interface{})
	IsPartial() bool
	SetPartial(partial bool)
	GetTemplateName() string
	SetTemplateName(name string)
	InvokeFilter(method string, obj interface{}, args ...interface{}) interface{}
	ApplyGlobalFilter(obj interface{}) interface{}
	NewIsolatedSubcontext() RenderContext
	MergeSubcontext(sub RenderContext)
	// Context returns the Go context; returns context.Background() if none was set
	Context() context.Context
}

// TagParseContext is the interface passed to TagFactory during tag construction.
type TagParseContext interface {
	ParseExpression(markup string) (interface{}, error)
	// ParseExpressionFromTokens parses an already-tokenized sub-slice directly,
	// avoiding the tokensToMarkup → ParseExpression round-trip for common cases.
	ParseExpressionFromTokens(tokens []Token) (interface{}, error)
	NewTokenizer(source string, startLine int, forLiquidTag bool) *Tokenizer
	LineNo() int
	GetParsedPartial(key string) (*ParsedPartial, bool)
	SetParsedPartial(key string, p *ParsedPartial)
	SetPartial(isPartial bool)
	GetErrorMode() string
}

// StaticPartialLoader is implemented by tags that reference a static (literal)
// partial name and can therefore be preloaded at parse time.
// ParseDocument calls PreloadPartial on every top-level node that implements
// this interface so that missing or broken partials are caught at parse time,
// not at first render.
type StaticPartialLoader interface {
	PreloadPartial(pc TagParseContext) error
}

type TagFactory func(tagName string, markup string, parseContext TagParseContext) (Tag, error)

// UnknownTagHandler is called during parsing when an unrecognised tag is encountered
type UnknownTagHandler func(tagName string, tagMarkup string) (bool, error)

type Node interface {
	RenderToOutputBuffer(ctx RenderContext, output *strings.Builder) error
	IsBlank() bool
	LineNumber() int
}

// Tag extends Node with parse-time behaviour.
type Tag interface {
	Node
	Parse(tokenizer *Tokenizer) error
	TagName() string
	Raw() string
}

// TagBase is the base implementation for Tag. Embed it in custom tags.
type TagBase struct {
	name   string
	markup string
	line   int
}

func NewTagBase(tagName string, markup string, parseContext TagParseContext) TagBase {
	line := 0
	if parseContext != nil {
		line = parseContext.LineNo()
	}
	return TagBase{name: tagName, markup: markup, line: line}
}

func (t TagBase) TagName() string { return t.name }
func (t TagBase) Raw() string     { return t.markup }
func (t TagBase) LineNumber() int { return t.line }
func (t TagBase) IsBlank() bool   { return false }
func (t TagBase) Parse(_ *Tokenizer) error { return nil }
func (t TagBase) RenderToOutputBuffer(_ RenderContext, _ *strings.Builder) error { return nil }

// StringNode holds a raw text node.
type StringNode struct {
	Value      string
	lineNumber int
}

func NewStringNode(value string, lineNumber int) *StringNode {
	return &StringNode{Value: value, lineNumber: lineNumber}
}

func (s *StringNode) RenderToOutputBuffer(_ RenderContext, output *strings.Builder) error {
	output.WriteString(s.Value)
	return nil
}

func (s *StringNode) IsBlank() bool   { return strings.TrimSpace(s.Value) == "" }
func (s *StringNode) LineNumber() int { return s.lineNumber }
func (s *StringNode) TrimRight()      { s.Value = strings.TrimRight(s.Value, " \t\n\r") }

// DisabledTag always returns a DisabledError when rendered.
type DisabledTag struct {
	TagBase
}

func (d DisabledTag) RenderToOutputBuffer(_ RenderContext, _ *strings.Builder) error {
	return DisabledError{BaseError: BaseError{Message: fmt.Sprintf("Tag '%s' is disabled", d.name)}, TagName: d.name}
}
