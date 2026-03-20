package parser

import (
	"fmt"
	"regexp"
)

var (
	identifierRegex     = regexp.MustCompile(`^[a-zA-Z_][\w-]*\??`)
	numberRegex         = regexp.MustCompile(`^-?\d+(\.\d+)?`)
	stringRegexSingle   = regexp.MustCompile(`^'([^']*)'`)
	stringRegexDouble   = regexp.MustCompile(`^"([^"]*)"`)
	skipWhitespaceRegex = regexp.MustCompile(`^\s+`)
)

func Tokenize(input string) ([]Token, error) {
	ss := NewStringScanner(input)
	tokens := []Token{}

	for !ss.EOS() {
		ss.Skip(skipWhitespaceRegex)
		if ss.EOS() {
			break
		}

		b, _ := ss.PeekByte()
		switch b {
		case '|':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: PipeToken, Value: "|"})
		case '.':
			ss.ScanByte()
			if nextB, ok := ss.PeekByte(); ok && nextB == '.' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: DotDotToken, Value: ".."})
			} else {
				tokens = append(tokens, Token{Type: DotToken, Value: "."})
			}
		case ':':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: ColonToken, Value: ":"})
		case ',':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: CommaToken, Value: ","})
		case '[':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: OpenSquareToken, Value: "["})
		case ']':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: CloseSquareToken, Value: "]"})
		case '(':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: OpenRoundToken, Value: "("})
		case ')':
			ss.ScanByte()
			tokens = append(tokens, Token{Type: CloseRoundToken, Value: ")"})
		case '=':
			ss.ScanByte()
			if nextB, ok := ss.PeekByte(); ok && nextB == '=' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: ComparisonToken, Value: "=="})
			} else {
				return nil, fmt.Errorf("Liquid syntax error: unexpected character %c", b)
			}
		case '!':
			ss.ScanByte()
			if nextB, ok := ss.PeekByte(); ok && nextB == '=' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: ComparisonToken, Value: "!="})
			} else {
				return nil, fmt.Errorf("Liquid syntax error: unexpected character %c", b)
			}
		case '<':
			ss.ScanByte()
			if nextB, ok := ss.PeekByte(); ok && nextB == '=' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: ComparisonToken, Value: "<="})
			} else if nextB == '>' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: ComparisonToken, Value: "<>"})
			} else {
				tokens = append(tokens, Token{Type: ComparisonToken, Value: "<"})
			}
		case '>':
			ss.ScanByte()
			if nextB, ok := ss.PeekByte(); ok && nextB == '=' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: ComparisonToken, Value: ">="})
			} else {
				tokens = append(tokens, Token{Type: ComparisonToken, Value: ">"})
			}
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			val := ss.Scan(numberRegex)
			if val != "" {
				tokens = append(tokens, Token{Type: NumberToken, Value: val})
			} else if b == '-' {
				ss.ScanByte()
				tokens = append(tokens, Token{Type: DashToken, Value: "-"})
			}
		case '\'', '"':
			var val string
			if b == '\'' {
				val = ss.Scan(stringRegexSingle)
			} else {
				val = ss.Scan(stringRegexDouble)
			}
			if val != "" {
				tokens = append(tokens, Token{Type: StringToken, Value: val})
			} else {
				return nil, fmt.Errorf("Liquid syntax error: unexpected character %c", b)
			}
		default:
			val := ss.Scan(identifierRegex)
			if val != "" {
				if val == "contains" {
					tokens = append(tokens, Token{Type: ComparisonToken, Value: "contains"})
				} else {
					tokens = append(tokens, Token{Type: IdToken, Value: val})
				}
			} else {
				return nil, fmt.Errorf("Liquid syntax error: unexpected character %s", ss.Getch())
			}
		}
	}

	tokens = append(tokens, Token{Type: EOSToken, Value: ""})
	return tokens, nil
}
