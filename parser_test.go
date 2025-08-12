package lox_interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseVarDeclaration(t *testing.T) {
	src := "var a = 123;"
	scanner := NewScanner(src)
	tokens, err := scanner.ScanTokens()
	require.NoError(t, err)

	parser := NewParser(tokens)
	stmts, err := parser.Parse()
	require.NoError(t, err)
	require.Len(t, stmts, 1)

	varStmt, ok := stmts[0].(*VarStmt)
	require.True(t, ok)
	assert.Equal(t, "a", varStmt.name.Lexeme)

	lit, ok := varStmt.initializer.(*LiteralExpr)
	require.True(t, ok)
	assert.Equal(t, 123.0, lit.value)
}
