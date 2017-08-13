package narfscript

import (
	"fmt"
	"math"
	"strings"
)

type NativeFunction func([]Value, *Env, *SrcLoc) (Value, error)

func valueIsTrue(val Value) bool {
	switch v := val.(type) {
	case *ValueNull:
		return false

	case *ValueBool:
		return v.val

	case *ValueNumber:
		return v.Number() != 0

	default:
		return true
	}
}

func valuesAreEqual(v1, v2 Value) bool {
	// both are numeric
	if n1, ok := v1.(ValueNumeric); ok {
		if n2, ok := v2.(ValueNumeric); ok {
			return n1.Number() == n2.Number()
		}
	}

	// both are strings
	if s1, ok := v1.(*ValueString); ok {
		if s2, ok := v2.(*ValueString); ok {
			return s1.String() == s2.String()
		}
	}

	return v1 == v2
}

func valueToNumber(val Value, loc *SrcLoc) (float64, error) {
	if v, ok := val.(ValueNumeric); ok {
		return v.Number(), nil
	}
	return 0, newExecError(loc, fmt.Sprintf("'%s' is not a number", val.Type()))
}

func valueToInt(val Value, loc *SrcLoc) (int, error) {
	n, err := valueToNumber(val, loc)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, newExecError(loc, fmt.Sprintf("can't convert %f to int", n))
	}
	return int(n), nil
}

// === error ===================================================

func nativeError(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	if len(args) == 0 {
		return nil, newExecError(loc, "error")
	}
	return nil, newExecException(loc, args[0])
}

// === printf ==================================================

func doSprintf(args []Value, env *Env, loc *SrcLoc) ([]string, error) {
	active := false
	next_arg := 1

	if len(args) == 0 {
		return make([]string, 0), nil
	}
	format_val, ok := args[0].(*ValueString)
	if !ok {
		return nil, newExecError(loc, "argument 1 must be string")
	}
	format := format_val.str

	buf := make([]string, 0, 16)
	for _, ch := range format {
		if active {
			switch ch {
			case '%':
				fmt.Printf("%%")

			case 's':
				if next_arg >= len(args) {
					return nil, newExecError(loc, "not enough arguments")
				} else {
					if str, ok := args[next_arg].(*ValueString); ok {
						buf = append(buf, fmt.Sprintf("%s", str.str))
					} else {
						buf = append(buf, fmt.Sprintf("%s", args[next_arg]))
					}
				}
				next_arg++

			case 'd':
				if next_arg >= len(args) {
					return nil, newExecError(loc, "not enough arguments")
				} else {
					num, err := valueToInt(args[next_arg], loc)
					if err != nil {
						return nil, err
					}
					buf = append(buf, fmt.Sprintf("%d", int64(num)))
				}
				next_arg++

			case 'f', 'g':
				if next_arg >= len(args) {
					return nil, newExecError(loc, "not enough arguments")
				} else {
					num, err := valueToNumber(args[next_arg], loc)
					if err != nil {
						return nil, err
					}
					buf = append(buf, fmt.Sprintf("%g", num))
				}
				next_arg++

			default:
				return nil, newExecError(loc, fmt.Sprintf("invalid format specifier: '%%%c'", ch))
			}
			active = false
		} else if ch == '%' {
			active = true
		} else {
			// this is horribly innefficient
			buf = append(buf, fmt.Sprintf("%c", ch))
		}
	}

	return buf, nil
}

func nativePrintf(args []Value, env *Env, loc *SrcLoc) (Value, error) {

	buf, err := doSprintf(args, env, loc)
	if err != nil {
		return nil, err
	}

	ret_n := 0
	for _, s := range buf {
		n, _ := fmt.Printf("%s", s)
		ret_n += n
	}

	return NewValueNumber(float64(ret_n)), nil
}

func nativeSprintf(args []Value, env *Env, loc *SrcLoc) (Value, error) {

	buf, err := doSprintf(args, env, loc)
	if err != nil {
		return nil, err
	}

	s := strings.Join(buf, "")
	return NewValueString(s), nil
}

// === Equality ================================================

func nativeEquals(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	if len(args) != 2 {
		return nil, newExecError(loc, "2 arguments required")
	}
	ret := NewValueBool(valuesAreEqual(args[0], args[1]))
	return ret, nil
}

func nativeNotEquals(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	if len(args) != 2 {
		return nil, newExecError(loc, "2 arguments required")
	}
	ret := NewValueBool(!valuesAreEqual(args[0], args[1]))
	return ret, nil
}

// === Binaty numeric ops ======================================

func getOpNumbers(args []Value, loc *SrcLoc) (float64, float64, error) {
	if len(args) != 2 {
		return 0, 0, newExecError(loc, "2 arguments required")
	}
	x, err := valueToNumber(args[0], loc)
	if err != nil {
		return 0, 0, err
	}
	y, err := valueToNumber(args[1], loc)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func nativeAdd(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueNumber(x + y), nil
}

func nativeSub(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	if len(args) == 1 {
		x, err := valueToNumber(args[0], loc)
		if err != nil {
			return nil, err
		}
		return NewValueNumber(-x), nil
	}

	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueNumber(x - y), nil
}

func nativeMul(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueNumber(x * y), nil
}

func nativeDiv(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueNumber(x / y), nil
}

func nativeMod(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueNumber(math.Mod(x, y)), nil
}

func nativePow(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueNumber(math.Pow(x, y)), nil
}

func nativeGreater(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueBool(x > y), nil
}

func nativeGreaterEqual(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueBool(x >= y), nil
}

func nativeLess(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueBool(x < y), nil
}

func nativeLessEqual(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	x, y, err := getOpNumbers(args, loc)
	if err != nil {
		return nil, err
	}
	return NewValueBool(x <= y), nil
}
