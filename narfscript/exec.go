package narfscript

import (
	"fmt"
)

type execStatement interface {
	dump(int)
	exec(env *Env) error
}

type execExpression interface {
	dump(int)
	eval(env *Env) (Value, error)
}

// func def
type execExprFuncDef struct {
	num_params int
	body       *execStmtBlock
}

func (e *execExprFuncDef) dump(indent int) {
	fmt.Printf("function(")
	for i := 0; i < e.num_params; i++ {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("<0:%d>", i)
	}
	fmt.Printf(") ")
	e.body.dump(indent)
	fmt.Printf("\n")
}

func (e *execExprFuncDef) eval(env *Env) (Value, error) {
	ret := &ValueClosure{
		fun: e,
		env: env,
	}
	return ret, nil
}

// block
type execStmtBlock struct {
	var_value execExpression
	stmts     []execStatement
}

func (e *execStmtBlock) dump(indent int) {
	fmt.Printf("{\n")
	for _, s := range e.stmts {
		fmt.Printf("%[1]*[2]s", indent+4, "")
		s.dump(indent + 4)
	}
	fmt.Printf("%[1]*[2]s", indent, "")
	fmt.Printf("}\n")
}

func (e *execStmtBlock) exec(env *Env) error {
	if e.var_value != nil {
		val, err := e.var_value.eval(env)
		if err != nil {
			return err
		}
		new_env := newEnv(env, 1)
		new_env.set(0, 0, val)
		env = new_env
	}
	for _, s := range e.stmts {
		err := s.exec(env)
		if err != nil {
			return err
		}
	}
	return nil
}

// if
type execStmtIf struct {
	test_expr  execExpression
	true_stmt  execStatement
	false_stmt execStatement
}

func (e *execStmtIf) dump(indent int) {
	fmt.Printf("if (")
	e.test_expr.dump(indent + 2)
	fmt.Printf(")")

	if block, ok := e.true_stmt.(*execStmtBlock); ok {
		fmt.Printf(" ")
		block.dump(indent)
	} else {
		fmt.Printf("\n%[1]*[2]s", indent+2, "")
		e.true_stmt.dump(indent + 2)
	}

	if e.false_stmt == nil {
		return
	}
	fmt.Printf("%[1]*[2]selse", indent, "")

	if block, ok := e.false_stmt.(*execStmtBlock); ok {
		fmt.Printf(" ")
		block.dump(indent)
	} else {
		fmt.Printf("\n%[1]*[2]s", indent+2, "")
		e.false_stmt.dump(indent + 2)
	}
}

func (e *execStmtIf) exec(env *Env) error {
	test_val, err := e.test_expr.eval(env)
	if err != nil {
		return err
	}
	if valueIsTrue(test_val) {
		return e.true_stmt.exec(env)
	}
	if e.false_stmt != nil {
		return e.false_stmt.exec(env)
	}
	return nil
}

// while
type execStmtWhile struct {
	test_expr execExpression
	stmt      execStatement
}

func (e *execStmtWhile) dump(indent int) {
	fmt.Printf("while (")
	e.test_expr.dump(indent + 2)
	fmt.Printf(")")

	if block, ok := e.stmt.(*execStmtBlock); ok {
		fmt.Printf(" ")
		block.dump(indent)
	} else {
		fmt.Printf("\n%[1]*[2]s", indent+2, "")
		e.stmt.dump(indent + 2)
	}
}

func (e *execStmtWhile) exec(env *Env) error {
	for {
		test_val, err := e.test_expr.eval(env)
		if err != nil {
			return err
		}
		if !valueIsTrue(test_val) {
			break
		}
		err = e.stmt.exec(env)
		if err != nil {
			if _, ok := err.(*breakError); ok {
				break
			}
			return err
		}
	}
	return nil
}

// return
type execStmtReturn struct {
	retval execExpression
}

func (e *execStmtReturn) dump(indent int) {
	fmt.Printf("return")
	if e.retval != nil {
		fmt.Printf(" ")
		e.retval.dump(indent)
	}
	fmt.Printf(";\n")
}

func (e *execStmtReturn) exec(env *Env) error {
	retval, err := e.retval.eval(env)
	if err != nil {
		return err
	}
	return newReturnError(retval)
}

// break
type execStmtBreak struct {
}

func (e *execStmtBreak) dump(indent int) {
	fmt.Printf("break;\n")
}

func (e *execStmtBreak) exec(env *Env) error {
	return newBreakError()
}

// statement expression
type execStmtExpression struct {
	e execExpression
}

func (e *execStmtExpression) dump(indent int) {
	e.e.dump(indent)
	fmt.Printf(";\n")
}

func (e *execStmtExpression) exec(env *Env) error {
	_, err := e.e.eval(env)
	return err
}

// map literal
type execExprMapLiteral struct {
	elements [][2]execExpression
}

func (e *execExprMapLiteral) dump(indent int) {
	fmt.Printf("{ ")
	for _, el := range e.elements {
		fmt.Printf("%[1]*[2]s", indent+4, "")
		el[0].dump(indent)
		fmt.Printf(" : ")
		el[1].dump(indent)
		fmt.Printf(",\n")
	}
	fmt.Printf("%[1]*[2]s", indent, "")
	fmt.Printf("}")
}

func (e *execExprMapLiteral) eval(env *Env) (Value, error) {
	elements := make([][2]Value, 0, len(e.elements))
	for _, exec_el := range e.elements {
		val_key, err := exec_el[0].eval(env)
		if err != nil {
			return nil, err
		}
		val_val, err := exec_el[1].eval(env)
		if err != nil {
			return nil, err
		}
		elements = append(elements, [2]Value{val_key, val_val})
	}
	ret := &ValueMap{
		elements: elements,
	}
	return ret, nil
}

// vector literal
type execExprVectorLiteral struct {
	elements []execExpression
}

func (e *execExprVectorLiteral) dump(indent int) {
	fmt.Printf("[ ")
	for i, el := range e.elements {
		if i > 0 {
			fmt.Printf(", ")
		}
		el.dump(indent)
	}
	fmt.Printf(" ]")
}

func (e *execExprVectorLiteral) eval(env *Env) (Value, error) {
	elements := make([]Value, 0, len(e.elements))
	for _, exec_el := range e.elements {
		val_el, err := exec_el.eval(env)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val_el)
	}
	ret := &ValueVector{
		elements: elements,
	}
	return ret, nil
}

// element index
type execExprElementIndex struct {
	container execExpression
	index     execExpression
	loc       SrcLoc
}

func (e *execExprElementIndex) dump(indent int) {
	e.container.dump(indent)
	fmt.Printf("[")
	e.index.dump(indent)
	fmt.Printf("]")
}

func (e *execExprElementIndex) eval(env *Env) (Value, error) {
	container, err := e.container.eval(env)
	if err != nil {
		return nil, err
	}
	index, err := e.index.eval(env)
	if err != nil {
		return nil, err
	}

	if c, ok := container.(ValueContainer); ok {
		return c.Get(index, &e.loc)
	}
	return nil, newExecError(&e.loc, fmt.Sprintf("trying to index non-containver value of type '%s'", container.Type()))
}

// ident
type execExprIdent struct {
	env_index int
	var_index int
	loc       SrcLoc
}

func (e *execExprIdent) dump(indent int) {
	fmt.Printf("<%d:%d>", e.env_index, e.var_index)
}

func (e *execExprIdent) eval(env *Env) (Value, error) {
	val := env.get(e.env_index, e.var_index)
	if val == nil {
		return nil, newExecError(&e.loc, fmt.Sprintf("undefined variable <%d:%d>", e.env_index, e.var_index))
	}
	return val, nil
}

// string
type execExprString struct {
	str string
}

func (e *execExprString) dump(indent int) {
	fmt.Printf("%q", e.str)
}

func (e *execExprString) eval(env *Env) (Value, error) {
	ret := &ValueString{e.str}
	return ret, nil
}

// number
type execExprNumber struct {
	num float64
}

func (e *execExprNumber) dump(indent int) {
	fmt.Printf("%g", e.num)
}

func (e *execExprNumber) eval(env *Env) (Value, error) {
	ret := &ValueNumber{e.num}
	return ret, nil
}

// var assignment
type execExprVarAssignment struct {
	env_e int
	env_i int
	val   execExpression
	loc   SrcLoc
}

func (e *execExprVarAssignment) dump(indent int) {
	fmt.Printf("<%d:%d> = ", e.env_e, e.env_i)
	e.val.dump(indent)
	fmt.Printf(";")
}

func (e *execExprVarAssignment) eval(env *Env) (Value, error) {
	val, err := e.val.eval(env)
	if err != nil {
		return nil, err
	}

	if !env.set(e.env_e, e.env_i, val) {
		return nil, newParserError(&e.loc, "unknown variable in assignment")
	}

	return val, nil
}

// container assignment
type execExprContainerSet struct {
	container execExpression
	index     execExpression
	val       execExpression
	loc       SrcLoc
}

func (e *execExprContainerSet) dump(indent int) {
	e.container.dump(indent)
	fmt.Printf("[")
	e.index.dump(indent)
	fmt.Printf("] = ")
	e.val.dump(indent)
	fmt.Printf(";")
}

func (e *execExprContainerSet) eval(env *Env) (Value, error) {
	container, err := e.container.eval(env)
	if err != nil {
		return nil, err
	}
	if c, ok := container.(ValueContainer); ok {
		index, err := e.index.eval(env)
		if err != nil {
			return nil, err
		}
		val, err := e.val.eval(env)
		if err != nil {
			return nil, err
		}
		err = c.Set(index, val, &e.loc)
		if err != nil {
			return nil, err
		}
		return val, nil
	}

	return nil, newParserError(&e.loc, fmt.Sprintf("trying to set value of non-container object of type '%s'", container.Type()))
}

// func call
type execExprFuncCall struct {
	fun  execExpression
	args []execExpression
	loc  SrcLoc
}

func (e *execExprFuncCall) dump(indent int) {
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

func (e *execExprFuncCall) eval(env *Env) (Value, error) {
	// evaluate function value
	fun_val, err := e.fun.eval(env)
	if err != nil {
		return nil, err
	}
	fun, ok := fun_val.(ValueCallable)
	if !ok {
		return nil, newExecError(&e.loc, fmt.Sprintf("trying to call non-function value of type '%s'", fun_val.Type()))
	}

	// evaluate argument values
	args := make([]Value, 0, len(e.args))
	for _, arg := range e.args {
		arg_val, err := arg.eval(env)
		if err != nil {
			return nil, err
		}
		args = append(args, arg_val)
	}

	// call function
	return fun.Call(args, env, &e.loc)
}
