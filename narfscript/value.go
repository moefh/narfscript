package narfscript

import (
	"fmt"
	"math"
	"strings"
)

var bleepNull ValueNull = ValueNull{}
var bleepTrue ValueBool = ValueBool{true}
var bleepFalse ValueBool = ValueBool{false}
var bleepNumbers []ValueNumber = make([]ValueNumber, 10)

func init() {
	for i := 0; i < len(bleepNumbers); i++ {
		bleepNumbers[i] = ValueNumber{float64(i)}
	}
}

type Value interface {
	Type() string
	String() string
}

type ValueNumeric interface {
	Type() string
	String() string
	Number() float64
}

type ValueCallable interface {
	Type() string
	String() string
	Call([]Value, *Env, *SrcLoc) (Value, error)
}

type ValueContainer interface {
	Type() string
	String() string
	Get(Value, *SrcLoc) (Value, error)
	Set(Value, Value, *SrcLoc) error
}

// null
type ValueNull struct {
}

func NewValueNull() *ValueNull {
	return &bleepNull
}

func (v *ValueNull) String() string {
	return "null"
}

func (v *ValueNull) Type() string {
	return "null"
}

// boolean
type ValueBool struct {
	val bool
}

func NewValueBool(val bool) *ValueBool {
	if val {
		return &bleepTrue
	} else {
		return &bleepFalse
	}
}

func (v *ValueBool) String() string {
	return fmt.Sprintf("%t", v.val)
}

func (v *ValueBool) Type() string {
	return "bool"
}

// number
type ValueNumber struct {
	num float64
}

func NewValueNumber(num float64) *ValueNumber {
	if num == math.Floor(num) && int(num) >= 0 && int(num) < len(bleepNumbers) {
		return &bleepNumbers[int(num)]
	}
	return &ValueNumber{num}
}

func (v *ValueNumber) Type() string {
	return "number"
}

func (v *ValueNumber) String() string {
	return fmt.Sprintf("%g", v.num)
}

func (v *ValueNumber) Number() float64 {
	return v.num
}

// string
type ValueString struct {
	str string
}

func NewValueString(str string) *ValueString {
	return &ValueString{str}
}

func (v *ValueString) Type() string {
	return "string"
}

func (v *ValueString) String() string {
	return fmt.Sprintf("%q", v.str)
}

// closure
type ValueClosure struct {
	fun *execExprFuncDef
	env *Env
}

func (v *ValueClosure) String() string {
	return "<closure>"
}

func (v *ValueClosure) Type() string {
	return "closure"
}

func (v *ValueClosure) Call(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	// chech number of parameters
	if len(args) != v.fun.num_params {
		return nil, newExecError(loc,
			fmt.Sprintf("invalid number of arguments: expected %d, got %d", v.fun.num_params, len(args)))
	}

	// create new env with arguments
	new_env := newEnv(v.env, v.fun.num_params)
	for i, arg := range args {
		new_env.set(0, i, arg)
	}

	// run function body
	err := v.fun.body.exec(new_env)
	if err != nil {
		if ret, ok := err.(*returnError); ok {
			return ret.retval, nil
		}
		return nil, err
	}

	// return
	ret := NewValueNull()
	return ret, nil
}

// native function
type ValueNativeFunction struct {
	fun NativeFunction
}

func NewValueNativeFunction(fun NativeFunction) *ValueNativeFunction {
	return &ValueNativeFunction{fun}
}

func (v *ValueNativeFunction) Type() string {
	return "native_function"
}

func (v *ValueNativeFunction) String() string {
	return fmt.Sprintf("<native function %p>", v.fun)
}

func (v *ValueNativeFunction) Call(args []Value, env *Env, loc *SrcLoc) (Value, error) {
	return v.fun(args, env, loc)
}

// vector
type ValueVector struct {
	elements []Value
}

func NewValueVector(elements []Value) *ValueVector {
	return &ValueVector{elements}
}

func (v *ValueVector) Type() string {
	return "vector"
}

func (v *ValueVector) String() string {
	ret := make([]string, 0, 1+2*len(v.elements))
	ret = append(ret, "[ ")
	for i, el := range v.elements {
		if i > 0 {
			ret = append(ret, ", ")
		}
		ret = append(ret, el.String())
	}
	ret = append(ret, " ]")
	return strings.Join(ret, "")
}

func (v *ValueVector) Get(index Value, loc *SrcLoc) (Value, error) {
	if n, ok := index.(ValueNumeric); ok {
		f := n.Number()
		i := int(f)
		if float64(i) != f {
			return nil, newExecError(loc, fmt.Sprintf("trying to index vector with non-integer number '%g'", f))
		}
		if i >= 0 && i < len(v.elements) {
			return v.elements[i], nil
		}
		return nil, newExecError(loc, fmt.Sprintf("array index out of bounds: %d", i))
	}
	return nil, newExecError(loc, fmt.Sprintf("trying to index vector with a non-numeric value of type '%s'", index.Type()))
}

func (v *ValueVector) Set(index Value, val Value, loc *SrcLoc) error {
	if n, ok := index.(ValueNumeric); ok {
		f := n.Number()
		i := int(f)
		if float64(i) != f {
			return newExecError(loc, fmt.Sprintf("trying to index vector with non-integer number '%g'", f))
		}
		if i >= 0 && i < len(v.elements) {
			v.elements[i] = val
			return nil
		}
		if i == len(v.elements) {
			v.elements = append(v.elements, val)
			return nil
		}
		return newExecError(loc, fmt.Sprintf("array index out of bounds: %d", i))
	}
	return newExecError(loc, fmt.Sprintf("trying to index vector with a non-numeric value of type '%s'", index.Type()))
}

// map
type ValueMap struct {
	elements [][2]Value
}

func NewValueMap(elements [][2]Value) *ValueMap {
	return &ValueMap{elements}
}

func (v *ValueMap) Type() string {
	return "map"
}

func (v *ValueMap) String() string {
	ret := make([]string, 0, 2+4*len(v.elements))
	ret = append(ret, "{ ")
	for _, el := range v.elements {
		ret = append(ret, el[0].String())
		ret = append(ret, " : ")
		ret = append(ret, el[1].String())
		ret = append(ret, ", ")
	}
	ret = append(ret, "}")
	return strings.Join(ret, "")
}

func (v *ValueMap) Get(key Value, loc *SrcLoc) (Value, error) {
	for _, el := range v.elements {
		if valuesAreEqual(key, el[0]) {
			return el[1], nil
		}
	}
	return NewValueNull(), nil
}

func (v *ValueMap) Set(key Value, val Value, loc *SrcLoc) error {
	for i, el := range v.elements {
		if valuesAreEqual(key, el[0]) {
			v.elements[i][1] = val
			return nil
		}
	}
	v.elements = append(v.elements, [2]Value{key, val})
	return nil
}
