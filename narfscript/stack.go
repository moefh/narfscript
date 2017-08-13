package narfscript

import (
	"fmt"
)

type operatorToken struct {
	op  *bleepOperator
	loc *SrcLoc
}

type exprStacks struct {
	opn_stack []astExpression
	opr_stack []*operatorToken
}

func newExprStacks() *exprStacks {
	return &exprStacks{
		opn_stack: make([]astExpression, 0),
		opr_stack: make([]*operatorToken, 0),
	}
}

func (s *exprStacks) dump(msg string) {
	fmt.Printf("=== %s ==================\n", msg)
	fmt.Printf("OPERATORS\n")
	for i, op := range s.opr_stack {
		fmt.Printf("[%3d]  %s\n", i, op.op.ident)
	}
	fmt.Printf("OPERANDS\n")
	for i, e := range s.opn_stack {
		fmt.Printf("[%3d]  ", i)
		e.dump(0)
		fmt.Printf("\n")
	}
	fmt.Printf("===============================\n")
}

func (s *exprStacks) numOperators() int {
	return len(s.opr_stack)
}

func (s *exprStacks) numOperands() int {
	return len(s.opn_stack)
}

func (s *exprStacks) peekOperator() *operatorToken {
	return s.opr_stack[len(s.opr_stack)-1]
}

func (s *exprStacks) peekOperand() astExpression {
	return s.opn_stack[len(s.opn_stack)-1]
}

func (s *exprStacks) pushOperator(opr *operatorToken) {
	s.opr_stack = append(s.opr_stack, opr)
}

func (s *exprStacks) pushOperand(opn astExpression) {
	s.opn_stack = append(s.opn_stack, opn)
}

func (s *exprStacks) popOperand() astExpression {
	l := len(s.opn_stack)
	ret := s.opn_stack[l-1]
	s.opn_stack = s.opn_stack[:l-1]
	return ret
}

func (s *exprStacks) popOperator() *operatorToken {
	l := len(s.opr_stack)
	ret := s.opr_stack[l-1]
	s.opr_stack = s.opr_stack[:l-1]
	return ret
}

func (s *exprStacks) resolve(stop_prec int32, loc *SrcLoc) error {
	//s.dump("BEFORE")

	for len(s.opr_stack) > 0 {
		// check if it's time to stop
		op := s.peekOperator()
		op_prec := op.op.prec
		if op.op.assoc == operatorAssocRight {
			op_prec--
		}
		if op_prec < stop_prec {
			break
		}
		op = s.popOperator()

		if op.op.assoc == operatorAssocPrefix {
			// resolve prefix op
			if len(s.opn_stack) < 1 {
				return newParserError(loc, "stack underflow")
			}
			opn := s.popOperand()
			call := &astExprFuncCall{
				fun:  &astExprIdent{op.op.ident, *op.loc},
				args: []astExpression{opn},
				loc:  *op.loc,
			}
			s.pushOperand(call)
		} else {
			// resolve binary op
			if len(s.opn_stack) < 2 {
				return newParserError(loc, "stack underflow")
			}
			right := s.popOperand()
			left := s.popOperand()
			call := &astExprFuncCall{
				fun:  &astExprIdent{op.op.ident, *op.loc},
				args: []astExpression{left, right},
				loc:  *op.loc,
			}
			s.pushOperand(call)
		}
	}

	//s.dump("AFTER")
	//fmt.Printf("\n\n")

	return nil
}
