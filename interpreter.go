package codecrafters_interpreter_go

import (
	"fmt"
	"log"
)

type RuntimeError struct {
	token   Token
	message string
}

func (r *RuntimeError) Error() string {
	return fmt.Sprintf("%d at '%s' %s", r.token.LineNumber, r.token.Lexeme, r.message)
}

func NewRuntimeError(token Token, message string) error {
	return &RuntimeError{token, message}
}

var _ StmtVisitor = (*Interpreter)(nil)
var _ ExprVisitor = (*Interpreter)(nil)

type Interpreter struct {
	env              *Environment
	currentLoop      Stmt
	breakCurrentLoop bool
	isReturningValue bool
}

func NewInterpreter() *Interpreter {
	env := NewEnvironment(nil)
	env.Define("clock", &Clock{})

	return &Interpreter{
		env: env,
	}
}

func (i *Interpreter) Interpret(expr []Stmt) (value interface{}, err error) {
	for _, stmt := range expr {
		var _err error
		value, _err = i.execute(stmt)

		if _err != nil {
			err = _err
			log.Printf("%s\n[line %d]", err.Error(), err.(*RuntimeError).token.LineNumber)
			return nil, err
		}
	}

	return value, err
}

func (i *Interpreter) execute(stmt Stmt) (interface{}, error) {
	if i.breakCurrentLoop {
		return nil, nil
	}

	value, err := stmt.Accept(i)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (i *Interpreter) executeBlock(statements []Stmt, env *Environment) (value interface{}, err error) {
	previous := i.env
	defer func() {
		i.env = previous
	}()

	i.env = env

	for _, statement := range statements {
		value, err = i.execute(statement)
		if err != nil {
			return nil, err
		}

		if i.isReturningValue {
			return value, nil
		}
	}

	return value, nil
}

// VisitVarStmt is function for variable statement. such as `var a = 1;`
func (i *Interpreter) VisitVarStmt(expr *Var) (v interface{}, err error) {
	var value interface{} = nil

	if expr.initializer != nil {
		value, err = i.Evaluate(expr.initializer)
		if err != nil {
			return nil, err
		}
	}

	i.env.Define(expr.name.Lexeme, value)

	return nil, nil // TODO: Find out why not returning the value.
}

func (i *Interpreter) VisitFunStmt(expr *Fun) (interface{}, error) {
	function := NewFunction(expr, i.env)
	i.env.Define(expr.name.Lexeme, function)
	return nil, nil
}

func (i *Interpreter) VisitExpressionStmt(expr *Expression) (interface{}, error) {
	_, err := i.Evaluate(expr.expression)
	return nil, err
}

func (i *Interpreter) VisitPrintStmt(expr *Print) (interface{}, error) {
	value, err := i.Evaluate(expr.expression)
	if err != nil {
		return nil, err
	}

	fmt.Println(Stringify(value))
	return nil, nil
}

func (i *Interpreter) VisitIfStmt(expr *If) (interface{}, error) {
	condition, err := i.Evaluate(expr.condition)
	if err != nil {
		return nil, err
	}

	if i.isTruthy(condition) {
		return i.execute(expr.thenBranch)
	} else if expr.elseBranch != nil {
		return i.execute(expr.elseBranch)
	}

	return nil, nil
}

func (i *Interpreter) VisitWhileStmt(expr *While) (interface{}, error) {
	i.currentLoop = expr
	defer func() {
		i.currentLoop = nil
		i.breakCurrentLoop = false
	}()

	condition, err := i.Evaluate(expr.condition)
	if err != nil {
		return nil, err
	}

	for i.isTruthy(condition) {
		_, err = i.execute(expr.body)
		if err != nil {
			return nil, err
		}

		condition, err = i.Evaluate(expr.condition)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (i *Interpreter) VisitBreakStmt(expr *Break) (interface{}, error) {
	i.breakCurrentLoop = true
	return nil, nil
}

func (i *Interpreter) VisitReturnStmt(expr *Return) (v interface{}, err error) {
	var value interface{} = nil

	if expr.value != nil {
		value, err = i.Evaluate(expr.value)
		if err != nil {
			return nil, err
		}
	}

	i.isReturningValue = true
	return value, nil
}

func (i *Interpreter) VisitBlockStmt(expr *Block) (interface{}, error) {
	return i.executeBlock(expr.statements, NewEnvironment(i.env))
}

func (i *Interpreter) Evaluate(expr Expr) (interface{}, error) {
	if i.breakCurrentLoop {
		return nil, nil
	}

	return expr.Accept(i)
}

func (i *Interpreter) VisitAssignExpr(expr *Assign) (interface{}, error) {
	value, err := i.Evaluate(expr.value)
	if err != nil {
		return nil, err
	}

	err = i.env.Assign(expr.name, value)
	return value, err
}

func (i *Interpreter) VisitLogicalExpr(expr *Logical) (interface{}, error) {
	left, err := i.Evaluate(expr.left)
	if err != nil {
		return nil, err
	}

	switch expr.operator.Type {
	case OR:
		if i.isTruthy(left) {
			return left, nil
		}
	case AND:
		if !i.isTruthy(left) {
			return left, nil
		}
	}

	return i.Evaluate(expr.right)
}

func (i *Interpreter) VisitTernaryExpr(expr *Ternary) (interface{}, error) {
	condition, err := i.Evaluate(expr.condition)
	if err != nil {
		return nil, err
	}

	if i.isTruthy(condition) {
		return i.Evaluate(expr.left)
	}

	return i.Evaluate(expr.right)
}

func (i *Interpreter) VisitUnaryExpr(expr *Unary) (interface{}, error) {
	right, err := i.Evaluate(expr.right)
	if err != nil {
		return nil, err
	}

	switch expr.operator.Type {
	case MINUS:
		isNumber := i.isAllNumber(right)
		if !isNumber {
			return nil, NewRuntimeError(expr.operator, "Operand must be a number.") // TODO: return error
		}

		return -right.(float64), nil
	case BANG:
		return !i.isTruthy(right), nil
	}

	return nil, nil // TODO: return error
}

func (i *Interpreter) VisitCallExpr(expr *Call) (interface{}, error) {
	defer func() {
		i.isReturningValue = false
	}()

	callee, err := i.Evaluate(expr.callee)
	if err != nil {
		return nil, err
	}

	var arguments []interface{}
	for _, argument := range expr.arguments {
		value, err := i.Evaluate(argument)
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, value)
	}

	function, ok := callee.(Callable)
	if !ok {
		return nil, NewRuntimeError(expr.paren, "Can only call functions and classes.")
	}

	if len(arguments) != function.Arity() {
		return nil, NewRuntimeError(expr.paren, fmt.Sprintf("Expected %d arguments but got %d.", function.Arity(), len(arguments)))
	}

	return function.Call(i, arguments)
}

func (i *Interpreter) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	if value == false {
		return false
	}

	return true
}

func (i *Interpreter) VisitBinaryExpr(expr *Binary) (interface{}, error) {
	left, err := i.Evaluate(expr.left)
	if err != nil {
		return nil, err
	}

	right, err := i.Evaluate(expr.right)
	if err != nil {
		return nil, err
	}

	switch expr.operator.Type {
	case MINUS:
		if !i.isAllNumber(left, right) {
			return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
		}

		return left.(float64) - right.(float64), nil
	case SLASH:
		if !i.isAllNumber(left, right) {
			return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
		}

		if right.(float64) == 0 {
			return nil, NewRuntimeError(expr.operator, "Division by zero.")
		}

		return left.(float64) / right.(float64), nil
	case STAR:
		if !i.isAllNumber(left, right) {
			return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
		}

		return left.(float64) * right.(float64), nil
	case PLUS:
		if i.isAllNumber(left, right) {
			return left.(float64) + right.(float64), nil
		}
		if i.isAllString(left, right) {
			return left.(string) + right.(string), nil
		}

		if useCrossAddition && i.isAllStringOrNumber(left, right) {
			return Stringify(left) + Stringify(right), nil
		}

		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
	case GREATER:
		if i.isAllNumber(left, right) {
			return left.(float64) > right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) > right.(string), nil
		}

		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
	case GREATER_EQUAL:
		if i.isAllNumber(left, right) {
			return left.(float64) >= right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) >= right.(string), nil
		}

		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
	case LESS:
		if i.isAllNumber(left, right) {
			return left.(float64) < right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) < right.(string), nil
		}
		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
	case LESS_EQUAL:
		if i.isAllNumber(left, right) {
			return left.(float64) <= right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) <= right.(string), nil
		}
		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.")
	case EQUAL_EQUAL:
		return left == right, nil
	case BANG_EQUAL:
		return left != right, nil
	}

	return nil, nil // TODO: return error
}

func (i *Interpreter) VisitLiteralExpr(expr *Literal) (interface{}, error) {
	return expr.value, nil
}

func (i *Interpreter) VisitGroupingExpr(expr *Grouping) (interface{}, error) {
	return i.Evaluate(expr.expression)
}

// VisitVariableExpr is function for variable expression. such as `a` when `a` is a identifier of variable.
func (i *Interpreter) VisitVariableExpr(expr *Variable) (interface{}, error) {
	value, err := i.env.Get(expr.name)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (i *Interpreter) isAllNumber(possibles ...interface{}) bool {
	for _, possible := range possibles {
		if _, ok := possible.(float64); !ok {
			return false
		}
	}

	return true
}

func (i *Interpreter) isAllString(possibles ...interface{}) bool {
	for _, possible := range possibles {
		if _, ok := possible.(string); !ok {
			return false
		}
	}

	return true
}

func (i *Interpreter) isAllStringOrNumber(possibles ...interface{}) bool {
	for _, possible := range possibles {
		_, isString := possible.(string)
		_, isNumber := possible.(float64)

		if !isString && !isNumber {
			return false
		}
	}

	return true
}

func Stringify(d interface{}) string {
	switch d.(type) {
	case float64:
		if d.(float64) == float64(int(d.(float64))) {
			return fmt.Sprintf("%.0f", d)
		}
		return fmt.Sprintf("%g", d.(float64))
	default:
		return toString(d)
	}
}
