package lox_interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanner_ScanTokens(t *testing.T) {
	src := "var a = 123;"
	scanner := NewScanner(src)
	tokens, err := scanner.ScanTokens()
	require.NoError(t, err)
	require.Len(t, tokens, 6)

	expectedTypes := []TokenType{VAR, IDENTIFIER, EQUAL, NUMBER, SEMICOLON, EOF}
	for i, typ := range expectedTypes {
		assert.Equal(t, typ, tokens[i].Type)
	}

	assert.Equal(t, "var", tokens[0].Lexeme)
	assert.Equal(t, "a", tokens[1].Lexeme)
	assert.Equal(t, "123", tokens[3].Lexeme)
	assert.Equal(t, 123.0, tokens[3].Literal)
}
