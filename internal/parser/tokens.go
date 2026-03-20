package parser

type TokenType string

const (
	IdToken          TokenType = "id"
	StringToken      TokenType = "string"
	NumberToken      TokenType = "number"
	DotToken         TokenType = "dot"
	DotDotToken      TokenType = "dotdot"
	ColonToken       TokenType = "colon"
	CommaToken       TokenType = "comma"
	OpenSquareToken  TokenType = "open_square"
	CloseSquareToken TokenType = "close_square"
	OpenRoundToken   TokenType = "open_round"
	CloseRoundToken  TokenType = "close_round"
	PipeToken        TokenType = "pipe"
	DashToken        TokenType = "dash"
	ComparisonToken  TokenType = "comparison"
	EOSToken         TokenType = "end_of_string"
)

type Token struct {
	Type  TokenType
	Value string
}
