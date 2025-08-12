package lox_interpreter

import (
	"fmt"
)

var _ StmtVisitor = (*AstPrinter)(nil)
var _ ExprVisitor = (*AstPrinter)(nil)

type AstPrinter struct{}

func (ap *AstPrinter) VisitListExpr(expr *ListExpr) (interface{}, error) {
	return ap.parenthesize("list", expr.values...)
}

func (ap *AstPrinter) VisitDictionaryExpr(expr *DictionaryExpr) (interface{}, error) {
	builder := "(dict"
	for k, v := range expr.mapExpr {
		p, err := ap.parenthesize(k.Lexeme, v)
		if err != nil {
			return "", err
		}
		builder += " " + p
	}
	builder += ")"
	return builder, nil
}

func (ap *AstPrinter) VisitSelectExpr(expr *SelectExpr) (interface{}, error) {
	return ap.parenthesize("select", expr.object, expr.name)
}

func (ap *AstPrinter) VisitSuperExpr(expr *SuperExpr) (interface{}, error) {
	return fmt.Sprintf("(%s %s)", expr.keyword.Lexeme, expr.method.Lexeme), nil
}

func (ap *AstPrinter) VisitThisExpr(expr *ThisExpr) (interface{}, error) {
	return expr.keyword.Lexeme, nil
}

func (ap *AstPrinter) VisitSetExpr(expr *SetExpr) (interface{}, error) {
	return ap.parenthesize("set "+expr.name.Lexeme, expr.object, expr.value)
}

func (ap *AstPrinter) VisitGetExpr(expr *GetExpr) (interface{}, error) {
	return ap.parenthesize("get "+expr.name.Lexeme, expr.object)
}

func (ap *AstPrinter) VisitClassStmt(expr *ClassStmt) (interface{}, error) {
	builder := "(class " + expr.name.Lexeme
	if expr.superClass != nil {
		sc, err := ap.parenthesize("super", expr.superClass)
		if err != nil {
			return "", err
		}
		builder += " " + sc
	}
	for _, method := range expr.methods {
		m, err := method.Accept(ap)
		if err != nil {
			return "", err
		}
		builder += " " + toString(m)
	}
	builder += ")"
	return builder, nil
}

func (ap *AstPrinter) VisitReturnStmt(expr *ReturnStmt) (interface{}, error) {
	if expr.value != nil {
		v, err := expr.value.Accept(ap)
		if err != nil {
			return "", err
		}
		return "(return " + toString(v) + ")", nil
	}
	return "(return)", nil
}

func (ap *AstPrinter) Print(stmts []Stmt) (string, error) {
	for _, stmt := range stmts {
		v, err := stmt.Accept(ap)
		if err != nil {
			return "", err
		}

		fmt.Println(v)
	}

	return "", nil
}

func (ap *AstPrinter) evaluate(expr Expr) (interface{}, error) {
	return expr.Accept(ap)
}

func (ap *AstPrinter) VisitVarStmt(stmt *VarStmt) (interface{}, error) {
	statementString := fmt.Sprintf("var (%s", stmt.name)

	if stmt.initializer != nil {
		d, err := ap.parenthesize("=", stmt.initializer)
		if err != nil {
			return "", err
		}
		statementString += toString(d)
	}

	statementString += ")"
	return statementString, nil
}

func (ap *AstPrinter) VisitExpressionStmt(expr *ExpressionStmt) (interface{}, error) {
	return expr.expression.Accept(ap)
}

func (ap *AstPrinter) VisitPrintStmt(expr *PrintStmt) (interface{}, error) {
	return ap.parenthesize("print", expr.expression)
}

func (ap *AstPrinter) VisitWhileStmt(expr *WhileStmt) (interface{}, error) {
	cond, err := expr.condition.Accept(ap)
	if err != nil {
		return "", err
	}
	body, err := expr.body.Accept(ap)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(while %s %s)", toString(cond), toString(body)), nil
}

func (ap *AstPrinter) VisitBreakStmt(expr *BreakStmt) (interface{}, error) {
	return "(" + expr.keyword.Lexeme + ")", nil
}

func (ap *AstPrinter) VisitBlockStmt(expr *BlockStmt) (interface{}, error) {
	build := "{"
	for _, stmt := range expr.statements {
		d, err := stmt.Accept(ap)
		if err != nil {
			return "", err
		}

		build += toString(d)
	}
	build += "}"

	return build, nil
}

func (ap *AstPrinter) VisitIfStmt(expr *IfStmt) (interface{}, error) {
	cond, err := expr.condition.Accept(ap)
	if err != nil {
		return "", err
	}
	thenStr, err := expr.thenBranch.Accept(ap)
	if err != nil {
		return "", err
	}
	if expr.elseBranch != nil {
		elseStr, err := expr.elseBranch.Accept(ap)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(if %s %s %s)", toString(cond), toString(thenStr), toString(elseStr)), nil
	}
	return fmt.Sprintf("(if %s %s)", toString(cond), toString(thenStr)), nil
}

func (ap *AstPrinter) VisitFunStmt(expr *FunStmt) (interface{}, error) {
	builder := "(fun " + expr.name.Lexeme + " ("
	for i, p := range expr.params {
		if i > 0 {
			builder += " "
		}
		builder += p.Lexeme
	}
	builder += ")"
	for _, stmt := range expr.body {
		s, err := stmt.Accept(ap)
		if err != nil {
			return "", err
		}
		builder += " " + toString(s)
	}
	builder += ")"
	return builder, nil
}

func (ap *AstPrinter) VisitAssignExpr(expr *AssignExpr) (interface{}, error) {
	return ap.parenthesize("= "+expr.name.Lexeme, expr.value)
}

func (ap *AstPrinter) VisitLogicalExpr(expr *LogicalExpr) (interface{}, error) {
	return ap.parenthesize(expr.operator.Lexeme, expr.left, expr.right)
}

func (ap *AstPrinter) VisitTernaryExpr(expr *TernaryExpr) (interface{}, error) {
	return ap.parenthesize("?:", expr.condition, expr.left, expr.right)
}

func (ap *AstPrinter) VisitBinaryExpr(expr *BinaryExpr) (interface{}, error) {
	return ap.parenthesize(expr.operator.Lexeme, expr.left, expr.right)
}

func (ap *AstPrinter) VisitGroupingExpr(expr *GroupingExpr) (interface{}, error) {
	return ap.parenthesize("group", expr.expression)
}

func (ap *AstPrinter) VisitLiteralExpr(expr *LiteralExpr) (interface{}, error) {
	if expr.value == nil {
		return "nil", nil
	}
	return toString(expr.value), nil
}

func (ap *AstPrinter) VisitUnaryExpr(expr *UnaryExpr) (interface{}, error) {
	return ap.parenthesize(expr.operator.Lexeme, expr.right)
}

func (ap *AstPrinter) parenthesize(name string, exprs ...Expr) (string, error) {
	var builder string
	builder += "(" + name

	for _, expr := range exprs {
		builder += " "
		d, err := expr.Accept(ap)
		if err != nil {
			return "", err
		}

		builder += toString(d)
	}

	builder += ")"

	return builder, nil
}

type RPNAstPrinter struct {
	AstPrinter
}

func (ap *RPNAstPrinter) Print(expr Expr) (string, error) {
	v, err := expr.Accept(ap)
	return v.(string), err
}

func (ap *RPNAstPrinter) VisitBinaryExpr(expr *BinaryExpr) (interface{}, error) {
	l, err := expr.left.Accept(ap)
	if err != nil {
		return "", err
	}

	r, err := expr.right.Accept(ap)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s %s", toString(l), toString(r), expr.operator.Lexeme), nil
}

func (ap *RPNAstPrinter) VisitGroupingExpr(expr *GroupingExpr) (interface{}, error) {
	return expr.expression.Accept(ap)
}

func (ap *AstPrinter) VisitCallExpr(expr *CallExpr) (interface{}, error) {
	exprs := append([]Expr{expr.callee}, expr.arguments...)
	return ap.parenthesize("call", exprs...)
}

func (ap *AstPrinter) VisitVariableExpr(expr *VariableExpr) (interface{}, error) {
	return expr.name.Lexeme, nil
}

func toString(d interface{}) string {
	switch d.(type) {
	case string:
		return d.(string)
	case float64:
		if d.(float64) == float64(int(d.(float64))) {
			return fmt.Sprintf("%.1f", d)
		}
		return fmt.Sprintf("%g", d.(float64))
	case int:
		return fmt.Sprintf("%d", d)
	case int64:
		return fmt.Sprintf("%d", d)
	case bool:
		return fmt.Sprintf("%t", d)
	default:
		return "nil"
	}
}
