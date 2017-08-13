package narfscript

import (
	"fmt"
)

var theBreakError breakError = breakError{}

// --------------------------------------------------------
// tokenizerError
type tokenizerError struct {
	msg string
}

func (e *tokenizerError) Error() string {
	return e.msg
}

// --------------------------------------------------------
// parserError
type parserError struct {
	msg      string
	filename string
	loc      SrcLoc
}

func (e *parserError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.loc.Filename, e.loc.Line, e.loc.Col, e.msg)
}

func newParserError(loc *SrcLoc, msg string) *parserError {
	return &parserError{
		msg: msg,
		loc: *loc,
	}
}

func newParserErrorFromTok(tok *token, msg string) *parserError {
	return &parserError{
		msg: msg,
		loc: tok.loc,
	}
}

// --------------------------------------------------------
// ExecError
type ExecError struct {
	msg string
	val Value
	loc SrcLoc
}

func newExecError(loc *SrcLoc, msg string) *ExecError {
	return &ExecError{
		msg: msg,
		val: nil,
		loc: *loc,
	}
}

func newExecException(loc *SrcLoc, val Value) *ExecError {
	var msg string
	if str, ok := val.(*ValueString); ok {
		msg = str.str
	} else {
		msg = "exception"
	}
	return &ExecError{
		msg: msg,
		val: val,
		loc: *loc,
	}
}

func (e *ExecError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.loc.Filename, e.loc.Line, e.loc.Col, e.msg)
}

func (e *ExecError) Value() Value {
	if e.val != nil {
		return e.val
	}
	return NewValueString(e.msg)
}

// --------------------------------------------------------
// Return
type returnError struct {
	retval Value
}

func (e *returnError) Error() string {
	return fmt.Sprintf("<return %s>", e.retval)
}

func newReturnError(retval Value) *returnError {
	return &returnError{retval}
}

// --------------------------------------------------------
// Break
type breakError struct {
	retval Value
}

func (e *breakError) Error() string {
	return "<break>"
}

func newBreakError() *breakError {
	return &theBreakError
}
