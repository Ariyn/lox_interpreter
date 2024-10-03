package codecrafters_interpreter_go

import (
	"fmt"
	"strconv"
)

type Scanner struct {
	Source string
	Tokens []Token

	start   int
	current int
	line    int
}

func (s *Scanner) ScanTokens() []Token {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.Tokens = append(s.Tokens, Token{
		EOF,
		"",
		nil,
		s.line,
	})

	return s.Tokens
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.Source)
}

func (s *Scanner) scanToken() error {
	c := s.advance()
	switch c {
	case "(":
		s.addToken(LEFT_PAREN, nil)
	case ")":
		s.addToken(RIGHT_PAREN, nil)
	case "{":
		s.addToken(LEFT_BRACE, nil)
	case "}":
		s.addToken(RIGHT_BRACE, nil)
	case ",":
		s.addToken(COMMA, nil)
	case ".":
		s.addToken(DOT, nil)
	case "-":
		s.addToken(MINUS, nil)
	case "+":
		s.addToken(PLUS, nil)
	case ";":
		s.addToken(SEMICOLON, nil)
	case "*":
		s.addToken(STAR, nil)
	case "!":
		typ := BANG
		if s.match("=") {
			typ = BANG_EQUAL
		}
		s.addToken(typ, nil)
	case "=":
		typ := EQUAL
		if s.match("=") {
			typ = EQUAL_EQUAL
		}
		s.addToken(typ, nil)
	case "<":
		typ := LESS
		if s.match("=") {
			typ = LESS_EQUAL
		}
		s.addToken(typ, nil)
	case ">":
		typ := GREATER
		if s.match("=") {
			typ = GREATER_EQUAL
		}
		s.addToken(typ, nil)
	case "/":
		if s.match("/") {
			for s.peek() != "\n" && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(SLASH, nil)
		}
	case " ":
		break
	case "\r":
		break
	case "\t":
		break
	case "\n":
		s.line += 1
	case "\"":
		s.string()
	case "o":
		if s.match("r") {
			s.addToken(OR, nil)
		}
	default:
		if s.isDigit(c) {
			s.number()
		} else if s.isAlphabet(c) {
			s.identifier()
		} else {
			return fmt.Errorf("Unexpected Character. - %d: %s", s.line, c)
		}
	}

	return nil
}

func (s *Scanner) identifier() {
	for s.isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := s.Source[s.start:s.current]

	if keywordType, ok := KeywordsMap[text]; ok {
		s.addToken(keywordType, nil)
	} else {
		s.addToken(IDENTIFIER, nil)
	}
}

func (s *Scanner) isAlphaNumeric(c string) bool {
	return s.isAlphabet(c) || s.isDigit(c)
}

func (s *Scanner) isAlphabet(c string) bool {
	return ('a' <= c[0] && c[0] <= 'z') || ('A' <= c[0] && c[0] <= 'Z') || c[0] == '_'
}

func (s *Scanner) isDigit(c string) bool {
	return '0' <= c[0] && c[0] <= '9'
}

func (s *Scanner) number() (err error) {
	for s.isDigit(s.peek()) {
		s.advance()
	}

	if s.peek() == "." && s.isDigit(s.peekNext()) {
		s.advance()

		for s.isDigit(s.peek()) {
			s.advance()
		}
	}

	f, err := strconv.ParseFloat(s.Source[s.start:s.current], 64)
	if err != nil {
		return err
	}

	s.addToken(NUMBER, f)
	return nil
}

func (s *Scanner) string() (err error) {
	for s.peek() != "\"" && !s.isAtEnd() {
		if s.peek() == "\n" {
			s.line += 1
		}
		s.advance()
	}

	if s.isAtEnd() {
		return fmt.Errorf("Unterminated string - %d: %s", s.line, s.Source[s.start:s.current])
	}

	s.advance()

	s.addToken(STRING, s.Source[s.start+1:s.current-1])
	return nil
}

func (s *Scanner) peekNext() string {
	if s.current+1 >= len(s.Source) {
		return "\\0"
	}

	return s.Source[s.current+1 : s.current+2]
}

func (s *Scanner) peek() string {
	if s.isAtEnd() {
		return "\\0"
	}
	return s.Source[s.current : s.current+1]
}

func (s *Scanner) match(next string) bool {
	if s.isAtEnd() {
		return false
	}

	if s.Source[s.current:s.current+1] != next {
		return false
	}

	s.current += 1
	return true
}

func (s *Scanner) advance() (next string) {
	next = s.Source[s.current : s.current+1]
	s.current += 1
	return next
}

func (s *Scanner) addToken(tokenType TokenType, literal any) {
	text := s.Source[s.start:s.current]
	s.Tokens = append(s.Tokens, Token{
		tokenType,
		text,
		literal,
		s.line,
	})
}