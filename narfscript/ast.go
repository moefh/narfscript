package narfscript

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func notImplemented(loc *SrcLoc, msg string) error {
	return newParserError(loc, fmt.Sprintf("%s not implemented", msg))
}

type analyzeFlags uint32

const (
	analyzeFlagsAllowBreak analyzeFlags = 1 << iota
)

type astStatement interface {
	dump(int)
	analyzeStmt(*symTab, analyzeFlags) (execStatement, error)
}

type astExpression interface {
	dump(int)
	analyzeExpr(*symTab) (execExpression, error)
}

// named func def
type astNamedFuncDef struct {
	name string
	def  *astExprFuncDef
}

func (e *astNamedFuncDef) dump(indent int) {
	fmt.Printf("%s = ", e.name)
	e.def.dump(indent)
	fmt.Printf("\n")
}

func (e *astNamedFuncDef) analyze(symtab *symTab) (*execExprFuncDef, error) {
	def, err := e.def.analyze(symtab)
	if err != nil {
		return nil, err
	}
	return def, nil
}

// block
type astStmtBlock struct {
	stmts []astStatement
}

func (e *astStmtBlock) dump(indent int) {
	fmt.Printf("{\n")
	for _, s := range e.stmts {
		fmt.Printf("%[1]*[2]s", indent+4, "")
		s.dump(indent + 4)
		fmt.Printf("\n")
	}
	fmt.Printf("%[1]*[2]s", indent, "")
	fmt.Printf("}")
}

func (e *astStmtBlock) analyzePart(var_value execExpression, stmts []astStatement, symtab *symTab, flags analyzeFlags) (*execStmtBlock, error) {
	ret_stmts := make([]execStatement, 0)
	for i := 0; i < len(stmts); i++ {
		ast_s := stmts[i]
		if ast_var, ok := ast_s.(*astStmtVar); ok {
			var_value, err := ast_var.val.analyzeExpr(symtab)
			if err != nil {
				return nil, err
			}
			new_symtab := newSymTab(symtab, []string{ast_var.ident})
			block, err := e.analyzePart(var_value, stmts[i+1:], new_symtab, flags)
			if err != nil {
				return nil, err
			}
			ret_stmts = append(ret_stmts, block)
			break
		} else {
			exec_s, err := ast_s.analyzeStmt(symtab, flags)
			if err != nil {
				return nil, err
			}
			ret_stmts = append(ret_stmts, exec_s)
		}
	}

	ret := &execStmtBlock{
		var_value: var_value,
		stmts:     ret_stmts,
	}
	return ret, nil
}

func (e *astStmtBlock) analyze(symtab *symTab, flags analyzeFlags) (*execStmtBlock, error) {
	return e.analyzePart(nil, e.stmts, symtab, flags)
}

func (e *astStmtBlock) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return e.analyze(symtab, flags)
}

// var
type astStmtVar struct {
	ident string
	val   astExpression
	loc   SrcLoc
}

func (e *astStmtVar) dump(indent int) {
	fmt.Printf("var %s", e.ident)
	if e.val != nil {
		fmt.Printf(" = ")
		e.val.dump(indent)
	}
	fmt.Printf(";")
}

func (e *astStmtVar) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return nil, newParserError(&e.loc, "trying to analyze 'var' statement")
}

// if
type astStmtIf struct {
	test_expr  astExpression
	true_stmt  astStatement
	false_stmt astStatement
}

func (e *astStmtIf) dump(indent int) {
	fmt.Printf("if (")
	e.test_expr.dump(indent + 2)
	fmt.Printf(") ")

	e.true_stmt.dump(indent)
	if e.false_stmt == nil {
		return
	}
	fmt.Printf(" else ")
	e.false_stmt.dump(indent)
}

func (e *astStmtIf) analyze(symtab *symTab, flags analyzeFlags) (*execStmtIf, error) {
	test_expr, err := e.test_expr.analyzeExpr(symtab)
	if err != nil {
		return nil, err
	}
	true_stmt, err := e.true_stmt.analyzeStmt(symtab, flags)
	if err != nil {
		return nil, err
	}
	false_stmt := execStatement(nil)
	if e.false_stmt != nil {
		false_stmt, err = e.false_stmt.analyzeStmt(symtab, flags)
		if err != nil {
			return nil, err
		}
	}

	ret := &execStmtIf{
		test_expr:  test_expr,
		true_stmt:  true_stmt,
		false_stmt: false_stmt,
	}
	return ret, nil
}

func (e *astStmtIf) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return e.analyze(symtab, flags)
}

// while
type astStmtWhile struct {
	test_expr astExpression
	stmt      astStatement
}

func (e *astStmtWhile) dump(indent int) {
	fmt.Printf("while (")
	e.test_expr.dump(indent + 2)
	fmt.Printf(")")

	if block, ok := e.stmt.(*astStmtBlock); ok {
		fmt.Printf(" ")
		block.dump(indent)
	} else {
		fmt.Printf("\n%[1]*[2]s", indent+2, "")
		e.stmt.dump(indent + 2)
	}
}

func (e *astStmtWhile) analyze(symtab *symTab, flags analyzeFlags) (*execStmtWhile, error) {
	test_expr, err := e.test_expr.analyzeExpr(symtab)
	if err != nil {
		return nil, err
	}
	stmt, err := e.stmt.analyzeStmt(symtab, flags|analyzeFlagsAllowBreak)
	if err != nil {
		return nil, err
	}

	ret := &execStmtWhile{
		test_expr: test_expr,
		stmt:      stmt,
	}
	return ret, nil
}

func (e *astStmtWhile) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return e.analyze(symtab, flags)
}

// return
type astStmtReturn struct {
	retval astExpression
}

func (e *astStmtReturn) dump(indent int) {
	fmt.Printf("return")
	if e.retval != nil {
		fmt.Printf(" ")
		e.retval.dump(indent)
	}
	fmt.Printf(";")
}

func (e *astStmtReturn) analyze(symtab *symTab, flags analyzeFlags) (*execStmtReturn, error) {
	retval := execExpression(nil)
	if e.retval != nil {
		expr, err := e.retval.analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		retval = expr
	}
	ret := &execStmtReturn{
		retval: retval,
	}
	return ret, nil
}

func (e *astStmtReturn) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return e.analyze(symtab, flags)
}

// break
type astStmtBreak struct {
	loc SrcLoc
}

func (e *astStmtBreak) dump(indent int) {
	fmt.Printf("break;")
}

func (e *astStmtBreak) analyze(symtab *symTab, flags analyzeFlags) (*execStmtBreak, error) {
	if (flags & analyzeFlagsAllowBreak) == 0 {
		return nil, newParserError(&e.loc, "break not allowed here")
	}
	ret := &execStmtBreak{}
	return ret, nil
}

func (e *astStmtBreak) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return e.analyze(symtab, flags)
}

// expression statement
type astStmtExpression struct {
	e astExpression
}

func (e *astStmtExpression) dump(indent int) {
	e.e.dump(indent)
	fmt.Printf(";")
}

func (e *astStmtExpression) analyze(symtab *symTab, flags analyzeFlags) (*execStmtExpression, error) {
	expr, err := e.e.analyzeExpr(symtab)
	if err != nil {
		return nil, err
	}
	ret := &execStmtExpression{
		e: expr,
	}
	return ret, nil
}

func (e *astStmtExpression) analyzeStmt(symtab *symTab, flags analyzeFlags) (execStatement, error) {
	return e.analyze(symtab, flags)
}

// func def
type astExprFuncDef struct {
	params []string
	body   *astStmtBlock
}

func (e *astExprFuncDef) dump(indent int) {
	fmt.Printf("function(")
	for i, p := range e.params {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s", p)
	}
	fmt.Printf(") ")
	e.body.dump(indent)
}

func (e *astExprFuncDef) analyze(symtab *symTab) (*execExprFuncDef, error) {
	new_symtab := newSymTab(symtab, e.params)

	body, err := e.body.analyze(new_symtab, 0)
	if err != nil {
		return nil, err
	}

	ret := &execExprFuncDef{
		num_params: len(e.params),
		body:       body,
	}
	return ret, nil
}

func (e *astExprFuncDef) analyzeExpr(symtab *symTab) (execExpression, error) {
	return e.analyze(symtab)
}

// map literal
type astExprMapLiteral struct {
	elements [][2]astExpression
}

func (e *astExprMapLiteral) dump(indent int) {
	fmt.Printf("{ ")
	for _, el := range e.elements {
		el[0].dump(indent + 4)
		fmt.Printf(" : ")
		el[1].dump(indent + 4)
		fmt.Printf(", ")
	}
	fmt.Printf("}")
}

func (e *astExprMapLiteral) analyzeExpr(symtab *symTab) (execExpression, error) {
	elements := make([][2]execExpression, 0, len(e.elements))
	for _, ast_el := range e.elements {
		exec_key, err := ast_el[0].analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		exec_val, err := ast_el[1].analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		elements = append(elements, [2]execExpression{exec_key, exec_val})
	}
	ret := &execExprMapLiteral{
		elements: elements,
	}
	return ret, nil
}

// vector literal
type astExprVectorLiteral struct {
	elements []astExpression
}

func (e *astExprVectorLiteral) dump(indent int) {
	fmt.Printf("[ ")
	for i, el := range e.elements {
		if i > 0 {
			fmt.Printf(", ")
		}
		el.dump(indent)
	}
	fmt.Printf(" ]")
}

func (e *astExprVectorLiteral) analyzeExpr(symtab *symTab) (execExpression, error) {
	elements := make([]execExpression, 0, len(e.elements))
	for _, ast_el := range e.elements {
		exec_el, err := ast_el.analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		elements = append(elements, exec_el)
	}
	ret := &execExprVectorLiteral{
		elements: elements,
	}
	return ret, nil
}

// element index
type astExprElementIndex struct {
	container astExpression
	index     astExpression
	loc       SrcLoc
}

func (e *astExprElementIndex) dump(indent int) {
	fmt.Printf("(")
	e.container.dump(indent)
	fmt.Printf(")")
	fmt.Printf("[")
	e.index.dump(indent)
	fmt.Printf("]")
}

func (e *astExprElementIndex) analyze(symtab *symTab) (*execExprElementIndex, error) {
	container, err := e.container.analyzeExpr(symtab)
	if err != nil {
		return nil, err
	}
	index, err := e.index.analyzeExpr(symtab)
	if err != nil {
		return nil, err
	}

	ret := &execExprElementIndex{
		container: container,
		index:     index,
		loc:       e.loc,
	}
	return ret, nil
}

func (e *astExprElementIndex) analyzeExpr(symtab *symTab) (execExpression, error) {
	return e.analyze(symtab)
}

// ident
type astExprIdent struct {
	name string
	loc  SrcLoc
}

func (e *astExprIdent) dump(indent int) {
	fmt.Printf("%s", e.name)
}

func (e *astExprIdent) analyze(symtab *symTab) (*execExprIdent, error) {
	env_index, var_index := symtab.getVar(e.name)
	if var_index < 0 {
		return nil, newParserError(&e.loc, fmt.Sprintf("undeclared variable '%s'", e.name))
	}
	ret := &execExprIdent{
		env_index: env_index,
		var_index: var_index,
		loc:       e.loc,
	}
	return ret, nil
}

func (e *astExprIdent) analyzeExpr(symtab *symTab) (execExpression, error) {
	return e.analyze(symtab)
}

// string
type astExprString struct {
	str string
}

func (e *astExprString) dump(indent int) {
	fmt.Printf("%q", e.str)
}

func (e *astExprString) analyze(symtab *symTab) (*execExprString, error) {
	ret := &execExprString{
		str: e.str,
	}
	return ret, nil
}

func (e *astExprString) analyzeExpr(symtab *symTab) (execExpression, error) {
	return e.analyze(symtab)
}

// number
type astExprNumber struct {
	num float64
}

func (e *astExprNumber) dump(indent int) {
	fmt.Printf("%g", e.num)
}

func (e *astExprNumber) analyze(symtab *symTab) (*execExprNumber, error) {
	ret := &execExprNumber{
		num: e.num,
	}
	return ret, nil
}

func (e *astExprNumber) analyzeExpr(symtab *symTab) (execExpression, error) {
	return e.analyze(symtab)
}

// func call
type astExprFuncCall struct {
	fun  astExpression
	args []astExpression
	loc  SrcLoc
}

func (e *astExprFuncCall) dump(indent int) {
	// binary operator
	if len(e.args) == 2 {
		if fun_op, ok := e.fun.(*astExprIdent); ok {
			first, _ := utf8.DecodeRuneInString(fun_op.name)
			if first != utf8.RuneError && !unicode.IsLetter(first) && first != '_' {
				fmt.Printf("(")
				e.args[0].dump(indent)
				fmt.Printf(" %s ", fun_op.name)
				e.args[1].dump(indent)
				fmt.Printf(")")
				return
			}
		}
	}

	e.fun.dump(indent)
	fmt.Printf("(")
	for i, a := range e.args {
		if i > 0 {
			fmt.Printf(", ")
		}
		a.dump(indent)
	}
	fmt.Printf(")")
}

func (e *astExprFuncCall) analyzeAssignment(symtab *symTab) (execExpression, error) {
	switch lval := e.args[0].(type) {
	case *astExprIdent:
		env_e, env_i := symtab.getVar(lval.name)
		if env_e < 0 {
			return nil, newParserError(&e.loc, fmt.Sprintf("unknown variable: '%s'", lval.name))
		}

		val, err := e.args[1].analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		ret := &execExprVarAssignment{
			env_e: env_e,
			env_i: env_i,
			val:   val,
			loc:   e.loc,
		}
		return ret, nil

	case *astExprElementIndex:
		container, err := lval.container.analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		index, err := lval.index.analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		val, err := e.args[1].analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		ret := &execExprContainerSet{
			container: container,
			index:     index,
			val:       val,
			loc:       e.loc,
		}
		return ret, nil

	default:
		return nil, newParserError(&e.loc, "assignment to invalid expression")
	}
}

func (e *astExprFuncCall) analyzeDot(symtab *symTab) (execExpression, error) {
	if ident, ok := e.args[1].(*astExprIdent); ok {
		container, err := e.args[0].analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		index := &execExprString{ident.name}

		ret := &execExprElementIndex{
			container: container,
			index:     index,
			loc:       e.loc,
		}
		return ret, nil
	}

	return nil, newParserError(&e.loc, "expected identifier after '.'")
}

func (e *astExprFuncCall) analyzeExpr(symtab *symTab) (execExpression, error) {
	// assignment to variable
	if len(e.args) == 2 {
		if fun_op, ok := e.fun.(*astExprIdent); ok {
			switch fun_op.name {
			case "=":
				return e.analyzeAssignment(symtab)

			case ".":
				return e.analyzeDot(symtab)
			}
		}
	}

	fun, err := e.fun.analyzeExpr(symtab)
	if err != nil {
		return nil, err
	}
	args := make([]execExpression, 0)

	for _, ast_arg := range e.args {
		exec_arg, err := ast_arg.analyzeExpr(symtab)
		if err != nil {
			return nil, err
		}
		args = append(args, exec_arg)
	}

	ret := &execExprFuncCall{
		fun:  fun,
		args: args,
		loc:  e.loc,
	}
	return ret, nil
}
