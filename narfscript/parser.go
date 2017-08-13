package narfscript

import (
	"fmt"
	"os"
)

type bleepParser struct {
	in          []*bleepTokenizer
	keywords    map[string]bool
	operators   []bleepOperator
	elIndexPrec int32
	funCallPrec int32
	last_tok    *token
	tok_saved   bool
}

func newParser(keywords map[string]bool, operators []bleepOperator, elIndexPrec, funCallPrec int32) *bleepParser {
	return &bleepParser{
		in:          make([]*bleepTokenizer, 0),
		keywords:    keywords,
		operators:   operators,
		elIndexPrec: elIndexPrec,
		funCallPrec: funCallPrec,
	}
}

func (parser *bleepParser) openFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	parser.in = append(parser.in, newTokenizer(file, parser.keywords, parser.operators))
	return nil
}

func (parser *bleepParser) getToken() *token {
	// return saved token if any
	if parser.tok_saved {
		parser.tok_saved = false
		return parser.last_tok
	}

	// read next token from current input, go to next input on EOF
	var eof_loc SrcLoc
	for len(parser.in) > 0 {
		cur_in := len(parser.in) - 1
		parser.last_tok = parser.in[cur_in].Next()
		if !parser.last_tok.isEOF() {
			return parser.last_tok
		}
		if eof_loc.Line == 0 {
			eof_loc = parser.in[cur_in].getSrcLoc()
		}
		parser.in = parser.in[:cur_in]
	}

	return newTokenEOF(eof_loc)
}

func (parser *bleepParser) ungetToken() bool {
	if parser.tok_saved {
		return false
	}
	parser.tok_saved = true
	return true
}

func (parser *bleepParser) peekToken() *token {
	if parser.tok_saved {
		return parser.last_tok
	}
	return parser.getToken()
}

func (parser *bleepParser) errMessage(loc *SrcLoc, msg string) error {
	return newParserError(loc, msg)
}

func (parser *bleepParser) errUnexpected(tok *token, expected string) error {
	return parser.errMessage(&tok.loc, fmt.Sprintf("expected %s, found %v", expected, tok))
}

func (parser *bleepParser) errPanic(tok *token, msg string) error {
	return newParserErrorFromTok(tok, msg)
}

func (parser *bleepParser) expectPunct(ch rune) error {
	tok := parser.getToken()
	if tok.isPunct(ch) {
		return nil
	}
	return newParserErrorFromTok(tok, fmt.Sprintf("expected '%c', found %v", ch, tok))
}

// -------------------------------------------------------------------

func (parser *bleepParser) getBinOperator(name string) *bleepOperator {
	for _, op := range parser.operators {
		if op.ident == name && (op.assoc == operatorAssocLeft || op.assoc == operatorAssocRight) {
			return &op
		}
	}
	return nil
}

func (parser *bleepParser) getPrefOperator(name string) *bleepOperator {
	for _, op := range parser.operators {
		if op.ident == name && op.assoc == operatorAssocPrefix {
			return &op
		}
	}
	return nil
}

// expression
func (parser *bleepParser) parseExpression(stop []rune, consume_stop bool) (astExpression, error) {
	stacks := newExprStacks()
	expect_opn := true

	for {
		tok := parser.getToken()
		if tok.tokType == tokenPunct {
			found_stop := false
			for _, st := range stop {
				if st == tok.ch {
					found_stop = true
					break
				}
			}

			if found_stop {
				if !consume_stop {
					parser.ungetToken()
				}

				if err := stacks.resolve(minOperatorPrec, &tok.loc); err != nil {
					return nil, err
				}

				switch stacks.numOperands() {
				case 0:
					return nil, parser.errUnexpected(tok, "expression")

				case 1:
					ret := stacks.popOperand()
					return ret, nil

				default:
					return nil, parser.errMessage(&tok.loc,
						fmt.Sprintf("invalid stack: %d elements left", stacks.numOperands()))
				}
			}
		}

		if tok.isEOF() {
			return nil, parser.errUnexpected(tok, "expression")
		}

		if tok.isPunct('(') {
			if expect_opn {
				// parenthesis expression
				expr, err := parser.parseExpression([]rune{')'}, true)
				if err != nil {
					return nil, err
				}
				stacks.pushOperand(expr)
				expect_opn = false
			} else {
				// function call
				loc := tok.loc
				parser.ungetToken()

				if err := stacks.resolve(parser.funCallPrec, &loc); err != nil {
					return nil, err
				}

				if stacks.numOperands() == 0 {
					return nil, parser.errPanic(tok, "operand stack is empty")
				}

				fun := stacks.popOperand()
				if ident, ok := fun.(*astExprIdent); ok {
					loc = ident.loc
				}
				args, err := parser.parseArgumentList()
				if err != nil {
					return nil, err
				}

				// make call expression
				call := &astExprFuncCall{
					fun:  fun,
					args: args,
					loc:  loc,
				}
				stacks.pushOperand(call)
			}
			continue
		}

		if tok.isPunct('[') {
			if expect_opn {
				// vector literal
				parser.ungetToken()
				vector, err := parser.parseVectorLiteral()
				if err != nil {
					return nil, err
				}
				stacks.pushOperand(vector)
				expect_opn = false
			} else {
				// indexing
				err := stacks.resolve(parser.elIndexPrec, &tok.loc)
				if err != nil {
					return nil, err
				}
				if stacks.numOperands() == 0 {
					return nil, parser.errPanic(tok, "operand stack is empty")
				}

				container := stacks.popOperand()
				index, err := parser.parseExpression([]rune{']'}, true)
				if err != nil {
					return nil, err
				}
				stacks.pushOperand(&astExprElementIndex{
					container: container,
					index:     index,
					loc:       tok.loc,
				})
			}
			continue
		}

		if tok.isPunct('{') {
			if !expect_opn {
				return nil, parser.errUnexpected(tok, "operator or '('")
			}
			// map literal
			parser.ungetToken()
			m, err := parser.parseMapLiteral()
			if err != nil {
				return nil, err
			}
			stacks.pushOperand(m)
			expect_opn = false
			continue
		}

		if tok.isKeyword("function") {
			if !expect_opn {
				return nil, parser.errUnexpected(tok, "operator or '('")
			}
			func_def, err := parser.parseFuncDef()
			if err != nil {
				return nil, err
			}
			stacks.pushOperand(func_def)
			expect_opn = false
			continue
		}

		if tok.isString() {
			if !expect_opn {
				return nil, parser.errUnexpected(tok, "operator or '('")
			}
			stacks.pushOperand(&astExprString{tok.str})
			expect_opn = false
			continue
		}

		if tok.isNumber() {
			if !expect_opn {
				return nil, parser.errUnexpected(tok, "operator or '('")
			}
			stacks.pushOperand(&astExprNumber{tok.num})
			expect_opn = false
			continue
		}

		if tok.isIdent() {
			if !expect_opn {
				return nil, parser.errUnexpected(tok, "operator or '('")
			}
			stacks.pushOperand(&astExprIdent{tok.str, tok.loc})
			expect_opn = false
			continue
		}

		if tok.isOp() {
			if expect_opn {
				op := parser.getPrefOperator(tok.str)
				if op == nil {
					return nil, parser.errMessage(&tok.loc, fmt.Sprintf("unknown prefix operator '%s'", tok.str))
				}
				stacks.pushOperator(&operatorToken{op, &tok.loc})
			} else {
				op := parser.getBinOperator(tok.str)
				if op == nil {
					return nil, parser.errMessage(&tok.loc, fmt.Sprintf("unknown binary operator '%s'", tok.str))
				}
				if err := stacks.resolve(op.prec, &tok.loc); err != nil {
					return nil, err
				}
				stacks.pushOperator(&operatorToken{op, &tok.loc})
				expect_opn = true
			}
			continue
		}

		return nil, parser.errUnexpected(tok, "expression")
	}
}

// map: { expr : expr, ...}
func (parser *bleepParser) parseMapLiteral() (*astExprMapLiteral, error) {
	if err := parser.expectPunct('{'); err != nil {
		return nil, err
	}

	elements := make([][2]astExpression, 0)
	for {
		next := parser.getToken()
		if next.isPunct('}') {
			break
		}
		if !next.isIdent() && !next.isString() {
			return nil, parser.errUnexpected(next, "identifier or string")
		}
		key := &astExprString{next.str}

		if err := parser.expectPunct(':'); err != nil {
			return nil, err
		}

		val, err := parser.parseExpression([]rune{',', '}'}, false)
		if err != nil {
			return nil, err
		}

		elements = append(elements, [2]astExpression{key, val})

		sep := parser.getToken()
		if sep.isPunct('}') {
			break
		}
		if !sep.isPunct(',') {
			return nil, parser.errUnexpected(sep, "',' or '}'")
		}
	}

	ret := &astExprMapLiteral{
		elements: elements,
	}
	return ret, nil
}

// vector: [expr, ...]
func (parser *bleepParser) parseVectorLiteral() (*astExprVectorLiteral, error) {
	if err := parser.expectPunct('['); err != nil {
		return nil, err
	}

	elements := make([]astExpression, 0)

	next := parser.getToken()
	if next.isPunct(']') {
		ret := &astExprVectorLiteral{
			elements: elements,
		}
		return ret, nil
	}
	parser.ungetToken()

	for {
		el, err := parser.parseExpression([]rune{',', ']'}, false)
		if err != nil {
			return nil, err
		}
		elements = append(elements, el)

		sep := parser.getToken()
		if !sep.isPunct(',') && !sep.isPunct(']') {
			return nil, parser.errUnexpected(sep, "',' or ']'")
		}
		if sep.isPunct(']') {
			break
		}
	}

	ret := &astExprVectorLiteral{
		elements: elements,
	}
	return ret, nil
}

// argument list: (expr, ...)
func (parser *bleepParser) parseArgumentList() ([]astExpression, error) {
	if err := parser.expectPunct('('); err != nil {
		return nil, err
	}

	args := make([]astExpression, 0)

	next := parser.getToken()
	if next.isPunct(')') {
		return args, nil
	}
	parser.ungetToken()

	for {
		arg, err := parser.parseExpression([]rune{',', ')'}, false)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		sep := parser.getToken()
		if !sep.isPunct(',') && !sep.isPunct(')') {
			return nil, parser.errUnexpected(sep, "',' or ')'")
		}
		if sep.isPunct(')') {
			break
		}
	}
	return args, nil
}

// param list: (name, ...)
func (parser *bleepParser) parseParamList() ([]string, error) {
	if err := parser.expectPunct('('); err != nil {
		return nil, err
	}

	params := make([]string, 0)

	next := parser.getToken()
	if next.isPunct(')') {
		return params, nil
	}
	parser.ungetToken()

	for {
		param := parser.getToken()
		if !param.isIdent() {
			return nil, parser.errUnexpected(param, "parameter name")
		}
		params = append(params, param.str)

		sep := parser.getToken()
		if !sep.isPunct(',') && !sep.isPunct(')') {
			return nil, parser.errUnexpected(sep, "',' or ')'")
		}
		if sep.isPunct(')') {
			break
		}
	}
	return params, nil
}

// var
func (parser *bleepParser) parseVar() (*astStmtVar, error) {
	// name
	name := parser.getToken()
	if !name.isIdent() {
		return nil, parser.errUnexpected(name, "identifier")
	}

	// value
	val := astExpression(nil)
	next := parser.getToken()
	if next.isOp() && next.str == "=" {
		v, err := parser.parseExpression([]rune{';'}, true)
		if err != nil {
			return nil, err
		}
		val = v
	} else {
		parser.ungetToken()
	}

	ret := &astStmtVar{
		ident: name.str,
		val:   val,
		loc:   name.loc,
	}
	return ret, nil
}

// if
func (parser *bleepParser) parseIf() (*astStmtIf, error) {
	if err := parser.expectPunct('('); err != nil {
		return nil, err
	}
	test_expr, err := parser.parseExpression([]rune{')'}, true)
	if err != nil {
		return nil, err
	}
	true_stmt, err := parser.parseStatement()
	if err != nil {
		return nil, err
	}

	tok := parser.getToken()
	false_stmt := astStatement(nil)
	if !tok.isKeyword("else") {
		parser.ungetToken()
	} else {
		false_stmt, err = parser.parseStatement()
		if err != nil {
			return nil, err
		}
	}

	ret := &astStmtIf{
		test_expr:  test_expr,
		true_stmt:  true_stmt,
		false_stmt: false_stmt,
	}
	return ret, nil
}

// while
func (parser *bleepParser) parseWhile() (*astStmtWhile, error) {
	if err := parser.expectPunct('('); err != nil {
		return nil, err
	}
	test_expr, err := parser.parseExpression([]rune{')'}, true)
	if err != nil {
		return nil, err
	}
	stmt, err := parser.parseStatement()
	if err != nil {
		return nil, err
	}

	ret := &astStmtWhile{
		test_expr: test_expr,
		stmt:      stmt,
	}
	return ret, nil
}

// return
func (parser *bleepParser) parseReturn() (*astStmtReturn, error) {
	next := parser.getToken()
	retval := astExpression(nil)
	if !next.isPunct(';') {
		parser.ungetToken()
		expr, err := parser.parseExpression([]rune{';'}, true)
		if err != nil {
			return nil, err
		}
		retval = expr
	}
	ret := &astStmtReturn{retval}
	return ret, nil
}

// statement
func (parser *bleepParser) parseStatement() (astStatement, error) {
	tok := parser.getToken()

	if tok.isEOF() {
		return nil, parser.errUnexpected(tok, "statement")
	}

	// empty statement
	if tok.isPunct(';') {
		block := &astStmtBlock{
			make([]astStatement, 0),
		}
		return block, nil
	}

	// { ... }
	if tok.isPunct('{') {
		parser.ungetToken()
		return parser.parseBlock()
	}

	// var
	if tok.isKeyword("var") {
		return parser.parseVar()
	}

	// if
	if tok.isKeyword("if") {
		return parser.parseIf()
	}

	// while
	if tok.isKeyword("while") {
		return parser.parseWhile()
	}

	// return
	if tok.isKeyword("return") {
		return parser.parseReturn()
	}

	// break
	if tok.isKeyword("break") {
		if err := parser.expectPunct(';'); err != nil {
			return nil, err
		}
		return &astStmtBreak{tok.loc}, nil
	}

	// expression ;
	parser.ungetToken()
	expr, err := parser.parseExpression([]rune{';'}, true)
	if err != nil {
		return nil, err
	}
	ret := &astStmtExpression{expr}
	return ret, nil
}

// block
func (parser *bleepParser) parseBlock() (*astStmtBlock, error) {
	if err := parser.expectPunct('{'); err != nil {
		return nil, err
	}

	stmts := make([]astStatement, 0)
	for {
		tok := parser.getToken()
		if tok.isPunct('}') {
			break
		}
		parser.ungetToken()

		stmt, err := parser.parseStatement()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}

	block := &astStmtBlock{
		stmts: stmts,
	}
	return block, nil
}

// function definition
func (parser *bleepParser) parseFuncDef() (*astExprFuncDef, error) {
	params, err := parser.parseParamList()
	if err != nil {
		return nil, err
	}

	body, err := parser.parseBlock()
	if err != nil {
		return nil, err
	}

	func_def := &astExprFuncDef{
		params: params,
		body:   body,
	}
	return func_def, nil
}

// named function
func (parser *bleepParser) parseNamedFuncDef() (*astNamedFuncDef, error) {
	name := parser.getToken()
	if !name.isIdent() {
		return nil, parser.errUnexpected(name, "function name")
	}

	func_def, err := parser.parseFuncDef()
	if err != nil {
		return nil, err
	}

	named_func_def := &astNamedFuncDef{
		name: name.str,
		def:  func_def,
	}
	return named_func_def, nil
}

func (parser *bleepParser) Parse(filename string) ([]*astNamedFuncDef, error) {
	if err := parser.openFile(filename); err != nil {
		return nil, err
	}

	funcs := make([]*astNamedFuncDef, 0)

	for {
		tok := parser.getToken()

		if tok.isEOF() {
			return funcs, nil
		}

		if tok.isKeyword("include") {
			filename := parser.getToken()
			if filename.isString() {
				if err := parser.openFile(filename.str); err != nil {
					return nil, newParserErrorFromTok(filename, err.Error())
				}
			}
			continue
		}

		if tok.isKeyword("function") {
			func_def, err := parser.parseNamedFuncDef()
			if err != nil {
				return nil, err
			}
			funcs = append(funcs, func_def)
			continue
		}

		return nil, parser.errUnexpected(tok, "'function' or 'include'")
	}
}
