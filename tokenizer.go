package liquid

import "github.com/go-liquid/internal/parser"

// NewTokenizer creates a Tokenizer for the given source.
// Kept as a ParseContext method so tag implementations don't need to
// import internal/parser directly.
func (p *ParseContext) NewTokenizer(source string, startLineNumber int, forLiquidTag bool) *Tokenizer {
	return parser.NewTokenizer(source, startLineNumber, forLiquidTag)
}
