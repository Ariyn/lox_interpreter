package lox_interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpreter_Evaluate(t *testing.T) {
	src := "var a = 1 + 2;"
	scanner := NewScanner(src)
	tokens, err := scanner.ScanTokens()
	require.NoError(t, err)

	parser := NewParser(tokens)
	stmts, err := parser.Parse()
	require.NoError(t, err)

	interpreter := NewInterpreter(nil)
	_, err = interpreter.Interpret(stmts)
	require.NoError(t, err)

	val, ok := interpreter.Env.Values["a"]
	require.True(t, ok)
	assert.Equal(t, 3.0, val)
}
