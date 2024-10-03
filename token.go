package codecrafters_interpreter_go

type TokenType string

const (
	// 단일 단어 토큰
	LEFT_PAREN  TokenType = "LEFT_PAREN"
	RIGHT_PAREN TokenType = "RIGHT_PAREN"
	LEFT_BRACE  TokenType = "LEFT_BRACE"
	RIGHT_BRACE TokenType = "RIGHT_BRACE"
	COMMA       TokenType = "COMMA"
	DOT         TokenType = "DOT"
	MINUS       TokenType = "MINUS"
	PLUS        TokenType = "PLUS"
	SEMICOLON   TokenType = "SEMICOLON"
	SLASH       TokenType = "SLASH"
	STAR        TokenType = "STAR"

	// 1~2 글자 토큰
	BANG          TokenType = "BANG"
	BANG_EQUAL    TokenType = "BANG_EQUAL"
	EQUAL         TokenType = "EQUAL"
	EQUAL_EQUAL   TokenType = "EQUAL_EQUAL"
	GREATER       TokenType = "GREATER"
	GREATER_EQUAL TokenType = "GREATER_EQUAL"
	LESS          TokenType = "LESS"
	LESS_EQUAL    TokenType = "LESS_EQUAL"

	// 리터럴
	IDENTIFIER TokenType = "IDENTIFIER"
	STRING     TokenType = "STRING"
	NUMBER     TokenType = "NUMBER"

	// 키워드
	AND    TokenType = "ADD"
	CLASS  TokenType = "CLASS"
	ELSE   TokenType = "ELSE"
	FALSE  TokenType = "FALSE"
	FUN    TokenType = "FUN"
	FOR    TokenType = "FOR"
	IF     TokenType = "IF"
	NIL    TokenType = "NIL"
	OR     TokenType = "OR"
	PRINT  TokenType = "PRINT"
	RETURN TokenType = "RETURN"
	SUPER  TokenType = "SUPER"
	THIS   TokenType = "THIS"
	TRUE   TokenType = "TRUE"
	VAR    TokenType = "VAR"
	WHILE  TokenType = "WHILE"

	EOF TokenType = "EOF"
)

var KeywordsMap = map[string]TokenType{
	"AND":    AND,
	"CLASS":  CLASS,
	"ELSE":   ELSE,
	"FALSE":  FALSE,
	"FOR":    FOR,
	"FUN":    FUN,
	"IF":     IF,
	"NIL":    NIL,
	"OR":     OR,
	"PRINT":  PRINT,
	"RETURN": RETURN,
	"SUPER":  SUPER,
	"THIS":   THIS,
	"TRUE":   TRUE,
	"VAR":    VAR,
	"WHILE":  WHILE,
}

type Token struct {
	Type       TokenType
	Lexeme     string
	Literal    any
	LineNumber int
}

func (t Token) String() string {
	return string(t.Type) + " " + t.Lexeme + " " // + string(t.Literal)
}