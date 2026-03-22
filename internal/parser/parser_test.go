package parser_test

import (
	"testing"

	"github.com/lumm2509/go-liquid/internal/parser"
)

// --- StringScanner ---

func TestStringScanner_BasicScan(t *testing.T) {
	ss := parser.NewStringScanner("hello world")
	if ss.EOS() {
		t.Fatal("expected not EOS")
	}

	b, ok := ss.PeekByte()
	if !ok || b != 'h' {
		t.Fatalf("expected 'h', got %q", b)
	}
}

func TestStringScanner_SetString(t *testing.T) {
	ss := parser.NewStringScanner("abc")
	ss.SetString("xyz")
	b, _ := ss.PeekByte()
	if b != 'x' {
		t.Fatalf("expected 'x' after SetString, got %q", b)
	}
}

func TestStringScanner_EOS(t *testing.T) {
	ss := parser.NewStringScanner("")
	if !ss.EOS() {
		t.Fatal("empty scanner should be EOS")
	}
}

// --- Tokenize ---

func TestTokenize_SimpleIdentifier(t *testing.T) {
	tokens, err := parser.Tokenize("foo")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 1 || tokens[0].Type != parser.IdToken || tokens[0].Value != "foo" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestTokenize_FilterExpression(t *testing.T) {
	tokens, err := parser.Tokenize("name | upcase")
	if err != nil {
		t.Fatal(err)
	}
	types := make([]parser.TokenType, 0, len(tokens))
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}
	// Expect: id, pipe, id, EOS
	expected := []parser.TokenType{parser.IdToken, parser.PipeToken, parser.IdToken, parser.EOSToken}
	if len(types) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(types), types)
	}
	for i, tt := range expected {
		if types[i] != tt {
			t.Errorf("token[%d]: want %q got %q", i, tt, types[i])
		}
	}
}

func TestTokenize_StringLiteral(t *testing.T) {
	tokens, err := parser.Tokenize(`"hello world"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 1 || tokens[0].Type != parser.StringToken {
		t.Fatalf("expected StringToken, got %v", tokens)
	}
}

func TestTokenize_Number(t *testing.T) {
	tokens, err := parser.Tokenize("42")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Type != parser.NumberToken || tokens[0].Value != "42" {
		t.Fatalf("expected NumberToken 42, got %v", tokens[0])
	}
}

func TestTokenize_Comparison(t *testing.T) {
	ops := []string{"==", "!=", "<=", ">=", "<", ">", "contains"}
	for _, op := range ops {
		tokens, err := parser.Tokenize(op)
		if err != nil {
			t.Fatalf("op %q: %v", op, err)
		}
		if tokens[0].Type != parser.ComparisonToken {
			t.Errorf("op %q: expected ComparisonToken, got %q", op, tokens[0].Type)
		}
	}
}

func TestTokenize_UnexpectedChar(t *testing.T) {
	_, err := parser.Tokenize("@invalid")
	if err == nil {
		t.Fatal("expected error for unexpected character")
	}
}

// --- Tokenizer ---

func TestTokenizer_SimpleText(t *testing.T) {
	tok := parser.NewTokenizer("hello", 1, false)
	text, ok := tok.Next()
	if !ok || text != "hello" {
		t.Fatalf("expected 'hello', got %q ok=%v", text, ok)
	}
	_, ok = tok.Next()
	if ok {
		t.Fatal("expected no more tokens")
	}
}

func TestTokenizer_TagToken(t *testing.T) {
	tok := parser.NewTokenizer("{% if true %}", 1, false)
	token, ok := tok.Next()
	if !ok {
		t.Fatal("expected a token")
	}
	if token != "{% if true %}" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestTokenizer_VariableToken(t *testing.T) {
	tok := parser.NewTokenizer("{{ name }}", 1, false)
	token, ok := tok.Next()
	if !ok {
		t.Fatal("expected a token")
	}
	if token != "{{ name }}" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestTokenizer_MixedContent(t *testing.T) {
	tok := parser.NewTokenizer("Hello {{ name }}!", 1, false)
	tokens := []string{}
	for {
		token, ok := tok.Next()
		if !ok {
			break
		}
		tokens = append(tokens, token)
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "Hello " || tokens[1] != "{{ name }}" || tokens[2] != "!" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestTokenizer_LineNumbers(t *testing.T) {
	tok := parser.NewTokenizer("line1\nline2\n{% tag %}", 1, false)
	for {
		_, ok := tok.Next()
		if !ok {
			break
		}
	}
	if tok.LineNumber != 3 {
		t.Fatalf("expected LineNumber=3, got %d", tok.LineNumber)
	}
}

// --- Parser ---

func TestParser_SimpleExpression(t *testing.T) {
	ss := parser.NewStringScanner("foo")
	p := parser.NewParser(ss)
	expr, err := p.Expression()
	if err != nil {
		t.Fatal(err)
	}
	if expr != "foo" {
		t.Fatalf("expected 'foo', got %q", expr)
	}
}

func TestParser_DotLookup(t *testing.T) {
	ss := parser.NewStringScanner("user.name")
	p := parser.NewParser(ss)
	expr, err := p.Expression()
	if err != nil {
		t.Fatal(err)
	}
	if expr != "user.name" {
		t.Fatalf("expected 'user.name', got %q", expr)
	}
}

func TestParser_BracketLookup(t *testing.T) {
	ss := parser.NewStringScanner(`items[0]`)
	p := parser.NewParser(ss)
	expr, err := p.Expression()
	if err != nil {
		t.Fatal(err)
	}
	if expr != "items[0]" {
		t.Fatalf("expected 'items[0]', got %q", expr)
	}
}
