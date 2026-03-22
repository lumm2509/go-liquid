package liquid

import (
	"regexp"

	"github.com/lumm2509/go-liquid/internal/parser"
)

const (
	ArgumentSeparator          = ","
	FilterArgumentSeparator    = ":"
	VariableAttributeSeparator = "."
	WhitespaceControl          = "-"
)

const (
	quotedString          = `(?:"[^"]*"|'[^']*')`
	quotedFragment        = `(?:` + quotedString + `|(?:[^\s,\|'"]|` + quotedString + `)+)`
	tagStart              = `\{%`
	tagEnd                = `%\}`
	variableStart         = `\{\{`
	variableEnd           = `\}\}`
	variableIncompleteEnd = `\}\}?`
)

// re-exported from internal/parser for public API
type TokenType = parser.TokenType
type Token = parser.Token
type StringScanner = parser.StringScanner
type Tokenizer = parser.Tokenizer
type Parser = parser.Parser

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

const (
	TagStartStr      = "{%"
	TagEndStr        = "%}"
	VariableStartStr = "{{"
	VariableEndStr   = "}}"
)

var (
	tagName           = regexp.MustCompile(`#|\w+`)
	variableSignature = regexp.MustCompile(`\(?[\w\-\.\[\]]\)?`)
	variableSegment   = regexp.MustCompile(`[\w\-]`)
	QuotedString      = regexp.MustCompile(quotedString)
	QuotedFragment    = regexp.MustCompile(quotedFragment)

	TagAttributes  = regexp.MustCompile(`(\w[\w-]*)\s*\:\s*(` + quotedFragment + `)`)
	AnyStartingTag = regexp.MustCompile(tagStart + `|` + variableStart)

	PartialTemplateParser = regexp.MustCompile(`(?s)` + tagStart + `.*?` + tagEnd + `|` + variableStart + `.*?` + variableIncompleteEnd)
	TemplateParser        = regexp.MustCompile(`(?s)(` + PartialTemplateParser.String() + `|` + AnyStartingTag.String() + `)`)

	VariableParser = regexp.MustCompile(`\[[^\[\]]+\]|` + variableSegment.String() + `+\??`)
)

func NewStringScanner(source string) *StringScanner { return parser.NewStringScanner(source) }
func NewParser(ss *StringScanner) *Parser           { return parser.NewParser(ss) }
func Tokenize(input string) ([]Token, error)        { return parser.Tokenize(input) }
