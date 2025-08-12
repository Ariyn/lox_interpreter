package lox_interpreter

import "fmt"

type EnvironmentError struct {
	token   Token
	message string
}

func NewEnvironmentError(token Token, message string) error {
	return EnvironmentError{token, message}
}

func (e EnvironmentError) Error() string {
	return fmt.Sprintf("%d at '%s' %s", e.token.LineNumber, e.token.Lexeme, e.message)
}

type Environment struct {
	Enclosing *Environment
	Values    map[string]interface{}
}

func NewEnvironment(enclosing *Environment) *Environment {
	return &Environment{
		Enclosing: enclosing,
		Values:    make(map[string]interface{}),
	}
}

// ancestor returns the environment `distance` steps above the current one.
// If the requested depth is out of range, it returns nil.
func (e *Environment) ancestor(distance int) *Environment {
	env := e
	for i := 0; i < distance; i++ {
		if env == nil {
			return nil
		}
		env = env.Enclosing
	}

	return env
}

func (e *Environment) depth() int {
	depth := 0
	for env := e; env != nil; env = env.Enclosing {
		depth++
	}

	return depth
}

func (e *Environment) Define(name string, value interface{}) {
	e.Values[name] = value
}

func (e *Environment) Assign(name Token, value interface{}) error {
	if _, ok := e.Values[name.Lexeme]; ok {
		e.Values[name.Lexeme] = value
		return nil
	}

	if e.Enclosing != nil {
		return e.Enclosing.Assign(name, value)
	}

	return NewEnvironmentError(name, fmt.Sprintf("Undefined variable '%s'", name.Lexeme))
}

func (e *Environment) AssignAt(distance int, name Token, value interface{}) error {
	if distance >= e.depth() {
		return NewEnvironmentError(name, fmt.Sprintf("Invalid ancestor. current : %d, distance: %d", e.depth(), distance))
	}

	ancestor := e.ancestor(distance)
	if ancestor == nil {
		return NewEnvironmentError(name, fmt.Sprintf("Invalid ancestor. current : %d, distance: %d", e.depth(), distance))
	}

	ancestor.Values[name.Lexeme] = value
	return nil
}

func (e *Environment) GetAt(distance int, name Token) (v interface{}, err error) {
	if distance >= e.depth() {
		return nil, NewEnvironmentError(name, fmt.Sprintf("Invalid ancestor. current : %d, distance: %d", e.depth(), distance))
	}

	ancestor := e.ancestor(distance)
	if ancestor == nil {
		return nil, NewEnvironmentError(name, fmt.Sprintf("Invalid ancestor. current : %d, distance: %d", e.depth(), distance))
	}

	v, ok := ancestor.Values[name.Lexeme]
	if !ok {
		return nil, NewEnvironmentError(name, fmt.Sprintf("Undefined variable '%s'", name.Lexeme))
	}

	return v, nil
}

func (e *Environment) GetAtWithString(distance int, name string) (v interface{}, err error) {
	if distance >= e.depth() {
		return nil, fmt.Errorf("Invalid ancestor. current : %d, distance: %d", e.depth(), distance)
	}

	ancestor := e.ancestor(distance)
	if ancestor == nil {
		return nil, fmt.Errorf("Invalid ancestor. current : %d, distance: %d", e.depth(), distance)
	}

	v, ok := ancestor.Values[name]
	if !ok {
		return nil, fmt.Errorf("Undefined variable '%s'", name)
	}

	return v, nil
}

func (e *Environment) Get(name Token) (interface{}, error) {
	if val, ok := e.Values[name.Lexeme]; ok {
		return val, nil
	}

	if e.Enclosing != nil {
		return e.Enclosing.Get(name)
	}

	return nil, NewEnvironmentError(name, fmt.Sprintf("Undefined variable '%s'", name.Lexeme))
}
