package narfscript

import "fmt"

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenError
	tokenKeyword
	tokenPunct
	tokenIdent
	tokenString
	tokenNumber
	tokenOp
)

type token struct {
	tokType tokenType
	str     string
	num     float64
	ch      rune
	loc     SrcLoc
}

func newTokenEOF(loc SrcLoc) *token {
	return &token{
		tokType: tokenEOF,
		loc:     loc,
	}
}

func newTokenError(msg string, loc SrcLoc) *token {
	return &token{
		tokType: tokenError,
		str:     msg,
		loc:     loc,
	}
}

func newTokenKeyword(ident string, loc SrcLoc) *token {
	return &token{
		tokType: tokenKeyword,
		str:     ident,
		loc:     loc,
	}
}

func newTokenIdent(ident string, loc SrcLoc) *token {
	return &token{
		tokType: tokenIdent,
		str:     ident,
		loc:     loc,
	}
}

func newTokenString(str string, loc SrcLoc) *token {
	return &token{
		tokType: tokenString,
		str:     str,
		loc:     loc,
	}
}

func newTokenNumber(num float64, loc SrcLoc) *token {
	return &token{
		tokType: tokenNumber,
		num:     num,
		loc:     loc,
	}
}

func newTokenOp(op string, loc SrcLoc) *token {
	return &token{
		tokType: tokenOp,
		str:     op,
		loc:     loc,
	}
}

func newTokenPunct(ch rune, loc SrcLoc) *token {
	return &token{
		tokType: tokenPunct,
		ch:      ch,
		loc:     loc,
	}
}

func (t *token) isEOF() bool {
	return t.tokType == tokenEOF
}

func (t *token) isError() bool {
	return t.tokType == tokenError
}

func (t *token) isKeyword(keyword string) bool {
	return t.tokType == tokenKeyword && t.str == keyword
}

func (t *token) isString() bool {
	return t.tokType == tokenString
}

func (t *token) isNumber() bool {
	return t.tokType == tokenNumber
}

func (t *token) isIdent() bool {
	return t.tokType == tokenIdent
}

func (t *token) isOp() bool {
	return t.tokType == tokenOp
}

func (t *token) isPunct(ch rune) bool {
	return t.tokType == tokenPunct && t.ch == ch
}

func (t *token) String() string {
	switch t.tokType {
	case tokenKeyword:
		return fmt.Sprintf("'%s'", t.str)
	case tokenIdent:
		return fmt.Sprintf("'%s'", t.str)
	case tokenString:
		return fmt.Sprintf("string")
	case tokenNumber:
		return fmt.Sprintf("'%g'", t.num)
	case tokenOp:
		return fmt.Sprintf("'%s'", t.str)
	case tokenPunct:
		return fmt.Sprintf("'%c'", t.ch)
	case tokenEOF:
		return "end of file"
	default:
		return fmt.Sprintf("<ERROR: %s>", t.str)
	}
}
