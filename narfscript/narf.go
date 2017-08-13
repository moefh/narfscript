package narfscript

import (
	"fmt"
)

func makeMap(els []string) map[string]bool {
	ret := make(map[string]bool)
	for _, el := range els {
		ret[el] = true
	}
	return ret
}

type Narf struct {
	symtab *symTab
	env    *Env
	parser *bleepParser
	funcs  map[string]*astNamedFuncDef
}

func NewNarf() *Narf {

	keywords := makeMap([]string{
		"include",
		"function",
		"var",
		"return",
		"if",
		"else",
		"while",
		"break",
	})

	operators := []bleepOperator{
		{"=", 10, operatorAssocLeft},

		{"||", 20, operatorAssocLeft},
		{"&&", 30, operatorAssocLeft},

		{"==", 40, operatorAssocLeft},
		{"!=", 40, operatorAssocLeft},
		{">", 50, operatorAssocLeft},
		{">=", 50, operatorAssocLeft},
		{"<", 50, operatorAssocLeft},
		{"<=", 50, operatorAssocLeft},

		{"+", 60, operatorAssocLeft},
		{"-", 60, operatorAssocLeft},
		{"*", 70, operatorAssocLeft},
		{"/", 70, operatorAssocLeft},
		{"%", 70, operatorAssocLeft},

		{"-", 80, operatorAssocPrefix},
		{"!", 80, operatorAssocPrefix},

		{"^", 90, operatorAssocRight},

		{".", 1001, operatorAssocLeft},
	}
	elIndexPrec := int32(1000)
	funCallPrec := int32(1000)

	bleep := &Narf{
		symtab: newSymTab(nil, nil),
		env:    newEnv(nil, 0),
		parser: newParser(keywords, operators, elIndexPrec, funCallPrec),
		funcs:  make(map[string]*astNamedFuncDef, 0),
	}
	bleep.setup()
	return bleep
}

func native(args []Value, env *Env) (Value, error) {
	return NewValueNumber(0), nil
}

func (bleep *Narf) setup() {
	// predefined functions and operators
	bleep.AddVar("null", NewValueNull())
	bleep.AddVar("false", NewValueBool(false))
	bleep.AddVar("true", NewValueBool(true))
	bleep.AddVar("==", NewValueNativeFunction(nativeEquals))
	bleep.AddVar("!=", NewValueNativeFunction(nativeNotEquals))
	bleep.AddVar("+", NewValueNativeFunction(nativeAdd))
	bleep.AddVar("-", NewValueNativeFunction(nativeSub))
	bleep.AddVar("*", NewValueNativeFunction(nativeMul))
	bleep.AddVar("/", NewValueNativeFunction(nativeDiv))
	bleep.AddVar("%", NewValueNativeFunction(nativeMod))
	bleep.AddVar("^", NewValueNativeFunction(nativePow))
	bleep.AddVar("<", NewValueNativeFunction(nativeLess))
	bleep.AddVar(">", NewValueNativeFunction(nativeGreater))
	bleep.AddVar("<=", NewValueNativeFunction(nativeLessEqual))
	bleep.AddVar(">=", NewValueNativeFunction(nativeGreaterEqual))
	bleep.AddVar("error", NewValueNativeFunction(nativeError))
	bleep.AddVar("printf", NewValueNativeFunction(nativePrintf))
}

func (bleep *Narf) AddVar(name string, val Value) {
	sym_index := bleep.symtab.addVar(name)
	if sym_index >= bleep.env.size() {
		env_index := bleep.env.grow(val)
		if env_index != sym_index {
			panic(fmt.Sprintf("error adding variable '%s': %d != %d", name, sym_index, env_index))
		}
	} else {
		bleep.env.set(0, sym_index, val)
	}
}

func (bleep *Narf) Parse(filename string) error {
	funcs, err := bleep.parser.Parse(filename)
	if err != nil {
		return err
	}

	for _, ast_f := range funcs {
		bleep.funcs[ast_f.name] = ast_f
		bleep.AddVar(ast_f.name, nil)
	}

	for _, ast_f := range funcs {
		exec_f, err := ast_f.analyze(bleep.symtab)
		if err != nil {
			return err
		}
		closure := &ValueClosure{
			fun: exec_f,
			env: bleep.env,
		}
		bleep.AddVar(ast_f.name, closure)
	}
	return nil
}

func (bleep *Narf) CallFunction(name string, args []Value) (Value, error) {
	loc := &SrcLoc{"<native>", 0, 0}
	env_index, var_index := bleep.symtab.getVar(name)
	fun := bleep.env.get(env_index, var_index)
	if fun == nil {
		return nil, newExecError(loc, fmt.Sprintf("function '%s' not found", name))
	}
	if f, ok := fun.(ValueCallable); ok {
		return f.Call(args, bleep.env, loc)
	}
	return nil, newExecError(loc, fmt.Sprintf("trying to call non-function value of type '%s'", fun.Type()))
}

func (bleep *Narf) DumpFunctions() {
	fmt.Printf("========================================\n")
	for _, f := range bleep.funcs {
		fmt.Printf("-> ")
		f.dump(0)
	}
	fmt.Printf("========================================\n")
}

func (bleep *Narf) DumpEnv() {
	fmt.Printf("========================================\n")
	bleep.symtab.Dump(bleep.env)
	fmt.Printf("========================================\n")
}
