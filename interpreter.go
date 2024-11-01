package lox_interpreter

import (
	"fmt"
	"strings"
)

type listType []interface{}
type dictType map[string]interface{}

type RuntimeError struct {
	token     Token
	message   string
	callstack []Callable
}

func (r *RuntimeError) Error() string {
	return fmt.Sprintf("%d at '%s' %s\n%s", r.token.LineNumber, r.token.Lexeme, r.message, r.callstackToString())
}

func (r *RuntimeError) callstackToString() string {
	var callstack string
	for i := len(r.callstack) - 1; i >= 0; i-- {
		c := r.callstack[i]
		callstack += fmt.Sprintf("%s[line %d] in %s\n", strings.Repeat(" ", i), c.(*LoxFunction).declaration.name.LineNumber, c.ToString())
	}

	return callstack
}

func NewRuntimeError(token Token, message string, callstack []Callable) error {
	return &RuntimeError{token, message, callstack}
}

var _ StmtVisitor = (*Interpreter)(nil)
var _ ExprVisitor = (*Interpreter)(nil)

type Interpreter struct {
	env              *Environment
	globals          *Environment
	currentLoop      Stmt
	breakCurrentLoop bool
	isReturningValue bool
	localsTable      map[Expr]int
	callStack        []Callable
}

func NewInterpreter(env *Environment) *Interpreter {
	if env == nil {
		env = NewEnvironment(nil)
	}

	env.Define("clock", &Clock{})

	return &Interpreter{
		env:         env,
		globals:     env,
		localsTable: make(map[Expr]int),
	}
}

func (i *Interpreter) Interpret(expr []Stmt) (value interface{}, err error) {
	for _, stmt := range expr {
		var _err error
		value, _err = i.execute(stmt)

		if _err != nil {
			err = _err
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
func (i *Interpreter) VisitVarStmt(expr *VarStmt) (v interface{}, err error) {
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

func (i *Interpreter) VisitFunStmt(expr *FunStmt) (interface{}, error) {
	function := NewFunction(expr, i.env, false)
	i.env.Define(expr.name.Lexeme, function)
	return nil, nil
}

func (i *Interpreter) VisitClassStmt(stmt *ClassStmt) (_ interface{}, err error) {
	i.env.Define(stmt.name.Lexeme, nil)

	var superclass *LoxClass = nil
	if stmt.superClass != nil {
		spc, err := i.Evaluate(stmt.superClass)
		if err != nil {
			return nil, err
		}

		if _, ok := spc.(*LoxClass); !ok {
			return nil, NewRuntimeError(stmt.superClass.name, "Superclass must be a class.", i.callStack)
		}

		superclass = spc.(*LoxClass)
	}

	i.env.Define(stmt.name.Lexeme, nil)

	if stmt.superClass != nil {
		i.env = NewEnvironment(i.env)
		i.env.Define("super", superclass)
	}

	methods := make(map[string]Callable)
	for _, method := range stmt.methods {
		function := NewFunction(method, i.env, method.name.Lexeme == "init")
		methods[method.name.Lexeme] = function
	}
	class := NewLoxClass(stmt.name.Lexeme, superclass, methods)

	if superclass != nil {
		i.env = i.env.Enclosing
	}
	err = i.env.Assign(stmt.name, class)

	return class, nil
}

func (i *Interpreter) VisitThisExpr(expr *ThisExpr) (interface{}, error) {
	return i.lookupTable(expr.keyword, expr)
}

func (i *Interpreter) VisitSuperExpr(expr *SuperExpr) (interface{}, error) {
	distance := i.localsTable[expr]
	spc, err := i.env.GetAtWithString(distance, "super")
	if err != nil {
		return nil, err
	}

	object, err := i.env.GetAtWithString(distance-1, "this")
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, NewRuntimeError(expr.keyword, "Cannot use 'super' in a class with no superclass.", i.callStack)
	}
	if _, ok := object.(*LoxInstance); !ok {
		return nil, NewRuntimeError(expr.keyword, "Cannot use 'super' in a class with no superclass.", i.callStack)
	}

	method := spc.(*LoxClass).findMethod(expr.method.Lexeme)
	if method == nil {
		return nil, NewRuntimeError(expr.method, fmt.Sprintf("Undefined property '%s'.", expr.method.Lexeme), i.callStack)
	}

	return method.(*LoxFunction).Bind(object.(*LoxInstance)), nil
}

func (i *Interpreter) VisitDictionaryExpr(expr *DictionaryExpr) (interface{}, error) {
	dict := make(dictType)
	for k, v := range expr.mapExpr {
		value, err := i.Evaluate(v)
		if err != nil {
			return nil, err
		}

		if _, ok := dict[k.Lexeme]; ok {
			return nil, NewRuntimeError(k, "Duplicate key in dictionary.", i.callStack)
		}
		dict[k.Lexeme] = value
	}

	return dict, nil
}

func (i *Interpreter) VisitSelectExpr(expr *SelectExpr) (interface{}, error) {
	object, err := i.Evaluate(expr.object)
	if err != nil {
		return nil, err
	}

	if dict, ok := object.(dictType); ok {
		name, err := i.Evaluate(expr.name)
		if err != nil {
			return nil, err
		}

		if _, ok := name.(string); !ok {
			return nil, NewRuntimeError(Token{}, "Property name must be a string.", i.callStack)
		}

		if value, ok := dict[name.(string)]; ok {
			return value, nil
		}

		return nil, NewRuntimeError(Token{}, fmt.Sprintf("Undefined property '%s'.", name.(string)), i.callStack)
	} else if list, ok := object.(listType); ok {
		index, err := i.Evaluate(expr.name)
		if err != nil {
			return nil, err
		}

		v, ok := index.(float64)
		if !ok {
			return nil, NewRuntimeError(Token{}, "Index must be a number.", i.callStack)
		}

		if int(v) < 0 || int(v) >= len(list) {
			return nil, NewRuntimeError(Token{}, fmt.Sprintf("Index out of range: %d", int(v)), i.callStack)
		}

		return list[int(v)], nil
	}

	return nil, NewRuntimeError(Token{}, "Only dictionaries or list can have properties.", i.callStack)
}

func (i *Interpreter) VisitListExpr(expr *ListExpr) (interface{}, error) {
	var values listType
	for _, v := range expr.values {
		value, err := i.Evaluate(v)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

func (i *Interpreter) VisitExpressionStmt(expr *ExpressionStmt) (interface{}, error) {
	_, err := i.Evaluate(expr.expression)
	return nil, err
}

func (i *Interpreter) VisitPrintStmt(expr *PrintStmt) (interface{}, error) {
	value, err := i.Evaluate(expr.expression)
	if err != nil {
		return nil, err
	}

	fmt.Println(Stringify(value))
	return nil, nil
}

func (i *Interpreter) VisitIfStmt(expr *IfStmt) (interface{}, error) {
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

func (i *Interpreter) VisitWhileStmt(expr *WhileStmt) (interface{}, error) {
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

func (i *Interpreter) VisitBreakStmt(expr *BreakStmt) (interface{}, error) {
	i.breakCurrentLoop = true
	return nil, nil
}

func (i *Interpreter) VisitReturnStmt(expr *ReturnStmt) (v interface{}, err error) {
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

func (i *Interpreter) VisitBlockStmt(expr *BlockStmt) (interface{}, error) {
	return i.executeBlock(expr.statements, NewEnvironment(i.env))
}

func (i *Interpreter) Evaluate(expr Expr) (interface{}, error) {
	if i.breakCurrentLoop {
		return nil, nil
	}

	return expr.Accept(i)
}

func (i *Interpreter) VisitAssignExpr(expr *AssignExpr) (interface{}, error) {
	value, err := i.Evaluate(expr.value)
	if err != nil {
		return nil, err
	}

	if distance, ok := i.localsTable[expr]; ok {
		err = i.env.AssignAt(distance, expr.name, value)
	} else {
		err = i.globals.Assign(expr.name, value)
	}

	if err != nil {
		return nil, NewRuntimeError(expr.name, err.Error(), i.callStack)
	}

	return value, nil
}

func (i *Interpreter) VisitLogicalExpr(expr *LogicalExpr) (interface{}, error) {
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

func (i *Interpreter) VisitTernaryExpr(expr *TernaryExpr) (interface{}, error) {
	condition, err := i.Evaluate(expr.condition)
	if err != nil {
		return nil, err
	}

	if i.isTruthy(condition) {
		return i.Evaluate(expr.left)
	}

	return i.Evaluate(expr.right)
}

func (i *Interpreter) VisitUnaryExpr(expr *UnaryExpr) (interface{}, error) {
	right, err := i.Evaluate(expr.right)
	if err != nil {
		return nil, err
	}

	switch expr.operator.Type {
	case MINUS:
		isNumber := i.isAllNumber(right)
		if !isNumber {
			return nil, NewRuntimeError(expr.operator, "Operand must be a number.", i.callStack) // TODO: return error
		}

		return -right.(float64), nil
	case BANG:
		return !i.isTruthy(right), nil
	}

	return nil, nil // TODO: return error
}

func (i *Interpreter) VisitCallExpr(expr *CallExpr) (interface{}, error) {
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

	callable, isCallable := callee.(Callable)
	if !isCallable {
		return nil, NewRuntimeError(expr.paren, "Can only call functions and classes.", i.callStack)
	}

	if len(arguments) != callable.Arity() {
		return nil, NewRuntimeError(expr.paren, fmt.Sprintf("Expected %d arguments but got %d.", callable.Arity(), len(arguments)), i.callStack)
	}

	i.callStack = append(i.callStack, callable)
	defer func() {
		i.callStack = i.callStack[:len(i.callStack)-1]
	}()
	return callable.Call(i, arguments)
}

func (i *Interpreter) VisitGetExpr(expr *GetExpr) (v interface{}, err error) {
	object, err := i.Evaluate(expr.object)
	if err != nil {
		return
	}

	if instance, ok := object.(*LoxInstance); ok {
		return instance.Get(expr.name)
	}

	return nil, NewRuntimeError(expr.name, "Only instances have properties.", i.callStack)
}

func (i *Interpreter) VisitSetExpr(expr *SetExpr) (v interface{}, err error) {
	object, err := i.Evaluate(expr.object)
	if err != nil {
		return
	}

	instance, ok := object.(*LoxInstance)
	if !ok {
		return nil, NewRuntimeError(expr.name, "Only instances have fields.", i.callStack)
	}

	return nil, instance.Set(expr.name, expr.value)
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

func (i *Interpreter) VisitBinaryExpr(expr *BinaryExpr) (interface{}, error) {
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
			return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
		}

		return left.(float64) - right.(float64), nil
	case SLASH:
		if !i.isAllNumber(left, right) {
			return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
		}

		if right.(float64) == 0 {
			return nil, NewRuntimeError(expr.operator, "Division by zero.", i.callStack)
		}

		return left.(float64) / right.(float64), nil
	case STAR:
		if !i.isAllNumber(left, right) {
			return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
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

		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
	case GREATER:
		if i.isAllNumber(left, right) {
			return left.(float64) > right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) > right.(string), nil
		}

		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
	case GREATER_EQUAL:
		if i.isAllNumber(left, right) {
			return left.(float64) >= right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) >= right.(string), nil
		}

		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
	case LESS:
		if i.isAllNumber(left, right) {
			return left.(float64) < right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) < right.(string), nil
		}
		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
	case LESS_EQUAL:
		if i.isAllNumber(left, right) {
			return left.(float64) <= right.(float64), nil
		} else if i.isAllString(left, right) {
			return left.(string) <= right.(string), nil
		}
		return nil, NewRuntimeError(expr.operator, "Operands must be two numbers or two strings.", i.callStack)
	case EQUAL_EQUAL:
		return left == right, nil
	case BANG_EQUAL:
		return left != right, nil
	}

	return nil, nil // TODO: return error
}

func (i *Interpreter) VisitLiteralExpr(expr *LiteralExpr) (interface{}, error) {
	return expr.value, nil
}

func (i *Interpreter) VisitGroupingExpr(expr *GroupingExpr) (interface{}, error) {
	return i.Evaluate(expr.expression)
}

// VisitVariableExpr is function for variable expression. such as `a` when `a` is a identifier of variable.
func (i *Interpreter) VisitVariableExpr(expr *VariableExpr) (interface{}, error) {
	v, err := i.lookupTable(expr.name, expr)
	if err != nil {
		return nil, err
	}

	if literal, ok := v.(*LiteralExpr); ok {
		return literal.value, nil
	}

	return v, nil
}

func (i *Interpreter) ResolveExpression(expr Expr, depth int) (_ interface{}, err error) {
	i.localsTable[expr] = depth

	return
}

func (i *Interpreter) lookupTable(name Token, expr Expr) (v interface{}, err error) {
	if depth, ok := i.localsTable[expr]; ok {
		v, err := i.env.GetAt(depth, name)
		if err != nil {
			return nil, NewRuntimeError(name, err.Error(), i.callStack)
		}

		return v, nil
	}

	v, err = i.globals.Get(name)
	if err != nil {
		return nil, NewRuntimeError(name, err.Error(), i.callStack)
	}

	return v, nil
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
	case *LiteralExpr:
		return Stringify(d.(*LiteralExpr).value)
	case float64:
		if d.(float64) == float64(int(d.(float64))) {
			return fmt.Sprintf("%.0f", d)
		}
		return fmt.Sprintf("%g", d.(float64))
	case Callable:
		return d.(Callable).ToString()
	case *LoxInstance:
		return d.(*LoxInstance).ToString()
	default:
		return toString(d)
	}
}
