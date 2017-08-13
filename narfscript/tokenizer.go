package narfscript

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"unicode"
	"unicode/utf8"
)

func is_ident(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch == '_')
}

func is_digit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func is_ident_cont(ch rune) bool {
	return is_ident(ch) || is_digit(ch)
}

func is_space(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

// --------------------------------------------------------------------------------
// Tokenizer

type bleepTokenizer struct {
	in        *bufio.Reader
	keywords  map[string]bool
	operators []bleepOperator
	filename  string
	line      int32
	col       int32
	last_line int32
	last_col  int32
}

func newTokenizer(file *os.File, keywords map[string]bool, operators []bleepOperator) *bleepTokenizer {
	return &bleepTokenizer{
		in:        bufio.NewReader(file),
		filename:  file.Name(),
		line:      1,
		col:       1,
		keywords:  keywords,
		operators: operators,
	}
}

func (t *bleepTokenizer) getRune() (rune, error) {
	ch, len, err := t.in.ReadRune()
	if err != nil {
		return 0, err
	}
	if ch == unicode.ReplacementChar && len == 1 {
		return 0, &tokenizerError{"invalid utf-8"}
	}

	t.last_line = t.line
	t.last_col = t.col
	if ch == '\n' {
		t.line++
		t.col = 1
	} else {
		t.col++
	}
	return ch, nil
}

func (t *bleepTokenizer) ungetRune() {
	t.line = t.last_line
	t.col = t.last_col
	t.in.UnreadRune()
}

func (t *bleepTokenizer) getSrcLoc() SrcLoc {
	return SrcLoc{t.filename, t.line, t.col}
}

func (t *bleepTokenizer) toTokenError(err error) *token {
	if err == io.EOF {
		return newTokenEOF(t.getSrcLoc())
	}
	return newTokenError(err.Error(), t.getSrcLoc())
}

func (t *bleepTokenizer) isKeyword(str string) bool {
	_, found := t.keywords[str]
	return found
}

func (t *bleepTokenizer) isOperator(str []rune) bool {
	for _, op := range t.operators {
		if len(str) != utf8.RuneCountInString(op.ident) {
			continue
		}
		match := true
		i := 0
		for _, ch := range op.ident {
			if str[i] != ch {
				match = false
				break
			}
			i++
		}
		if match {
			return true
		}
	}
	return false
}

func (t *bleepTokenizer) Next() *token {
	var (
		first rune
		loc   SrcLoc
	)
	for {
		loc = t.getSrcLoc()
		ch, err := t.getRune()
		if err != nil {
			return t.toTokenError(err)
		}
		if !is_space(ch) {
			first = ch
			break
		}
	}

	buf := make([]rune, 0)
	switch {

	// comment
	case first == '#':
		for {
			ch, err := t.getRune()
			if err != nil {
				return t.toTokenError(err)
			}
			if ch == '\n' {
				break
			}
		}
		return t.Next()

	// identifier or keyword
	case is_ident(first):
		buf = append(buf, first)
		for {
			ch, err := t.getRune()
			if err != nil {
				if err == io.EOF {
					break
				}
				return t.toTokenError(err)
			}
			if !is_ident_cont(ch) {
				t.ungetRune()
				break
			}
			buf = append(buf, ch)
		}
		ident := string(buf)
		if t.isKeyword(ident) {
			return newTokenKeyword(ident, loc)
		} else {
			return newTokenIdent(ident, loc)
		}

	// number
	case is_digit(first):
		buf = append(buf, first)
		last := first
		for {
			ch, err := t.getRune()
			if err != nil {
				if err == io.EOF {
					break
				}
				return t.toTokenError(err)
			}
			if is_digit(ch) || ch == '.' ||
				(ch == 'e' && last != '-' && last != '+') ||
				(ch == '-' && last == 'e') ||
				(ch == '+' && last == 'e') {
				last = ch
				buf = append(buf, ch)
			} else {
				t.ungetRune()
				break
			}
		}
		num, err := strconv.ParseFloat(string(buf), 64)
		if err != nil {
			return t.toTokenError(err)
		}
		return newTokenNumber(num, loc)

	// string
	case first == '"':
		for {
			ch, err := t.getRune()
			if err != nil {
				if err == io.EOF {
					return newTokenError("unterminated string", t.getSrcLoc())
				}
				return t.toTokenError(err)
			}
			if ch == '"' {
				break
			}
			if ch == '\\' {
				next, err := t.getRune()
				if err != nil {
					if err == io.EOF {
						return newTokenError("unterminated string", t.getSrcLoc())
					}
					return t.toTokenError(err)
				}
				switch next {
				case '"':
					buf = append(buf, '"')
				case '\'':
					buf = append(buf, '\'')
				case '\\':
					buf = append(buf, '\\')
				case 'r':
					buf = append(buf, '\r')
				case 'n':
					buf = append(buf, '\n')
				case 't':
					buf = append(buf, '\t')
				default:
					return newTokenError(
						fmt.Sprintf("invalid character escape: '\\%c'", next),
						t.getSrcLoc())
				}
			} else {
				buf = append(buf, ch)
			}
		}
		return newTokenString(string(buf), loc)

	case first == ',' || first == ';' || first == ':' ||
		first == '(' || first == ')' ||
		first == '{' || first == '}' ||
		first == '[' || first == ']':
		return newTokenPunct(first, loc)

	// any other character starts an operator
	default:
		buf = append(buf, first)
		for {
			ch, err := t.getRune()
			if err != nil {
				if err == io.EOF {
					break
				}
				return t.toTokenError(err)
			}
			buf = append(buf, ch)
			if !t.isOperator(buf) {
				t.ungetRune()
				buf = buf[:len(buf)-1]
				break
			}
		}
		return newTokenOp(string(buf), loc)
	}
}
