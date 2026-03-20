package liquid

import (
	"strings"
)

// Node es la interfaz mínima que debe cumplir cualquier objeto en el árbol de parseo
type Node interface {
	RenderToOutputBuffer(context *Context, output *strings.Builder) error
	IsBlank() bool
	LineNumber() int
}

// StringNode represents a raw text literal in the parsed template tree.
// Replaces the bare string entries in BlockBody.NodeList, making NodeList
// fully typed as []Node and eliminating the type assertion in Render.
type StringNode struct {
	content string
	line    int
}

func NewStringNode(content string, line int) *StringNode {
	return &StringNode{content: content, line: line}
}

func (s *StringNode) RenderToOutputBuffer(_ *Context, output *strings.Builder) error {
	output.WriteString(s.content)
	return nil
}

func (s *StringNode) IsBlank() bool {
	for _, r := range s.content {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

func (s *StringNode) LineNumber() int { return s.line }

func (s *StringNode) TrimRight() {
	n := len(s.content)
	for n > 0 {
		c := s.content[n-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		n--
	}
	s.content = s.content[:n]
}

// TagParseContext is the interface passed to TagFactory during tag construction.
// It exposes only what a custom tag implementation needs at parse time.
// The concrete *ParseContext implements this interface; external tag authors
// should type their factory functions against TagParseContext, not *ParseContext.
type TagParseContext interface {
	ParseExpression(markup string) (interface{}, error)
	NewTokenizer(source string, startLine int, forLiquidTag bool) *Tokenizer
	LineNo() int
}

// TagFactory es el tipo de función necesaria para registrar tags en el Environment
type TagFactory func(tagName string, markup string, parseContext TagParseContext) (Tag, error)

type Tag interface {
	Node
	Parse(tokenizer *Tokenizer) error
	TagName() string
	Raw() string
}

type TagBase struct {
	name   string
	markup string
	line   int
}

// NewTagBase equivale al initialize de Ruby
func NewTagBase(tagName string, markup string, parseContext TagParseContext) TagBase {
	return TagBase{
		name:   tagName,
		markup: markup,
		line:   parseContext.LineNo(),
	}
}

// Parse es un método por defecto (no-op en Ruby)
func (t *TagBase) Parse(tokenizer *Tokenizer) error {
	return nil
}

// Raw devuelve la representación original del tag
func (t *TagBase) Raw() string {
	return t.name + " " + t.markup
}

// TagName devuelve el nombre del tag registrado
func (t *TagBase) TagName() string {
	return t.name
}

func (t *TagBase) LineNumber() int {
	return t.line
}

// IsBlank equivale a blank? (por defecto false en Ruby)
func (t *TagBase) IsBlank() bool {
	return false
}

// RenderToOutputBuffer maneja la escritura en el buffer.
func (t *TagBase) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return nil
}

func WrapWithDisabler(originalFactory TagFactory) TagFactory {
	return func(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
		return &DisabledTag{TagBase: NewTagBase(tagName, markup, parseContext), tagName: tagName}, nil
	}
}

type DisabledTag struct {
	TagBase
	tagName string
}

func (d *DisabledTag) Parse(tokenizer *Tokenizer) error {
	return &DisabledError{TagName: d.tagName}
}

func (d *DisabledTag) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return nil
}
