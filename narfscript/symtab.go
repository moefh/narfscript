package narfscript

import (
	"fmt"
)

type symTab struct {
	parent *symTab
	names  map[string]int
}

func newSymTab(parent *symTab, names []string) *symTab {
	symtab := &symTab{
		parent: parent,
		names:  make(map[string]int, 0),
	}
	if names != nil {
		for _, name := range names {
			symtab.addVar(name)
		}
	}
	return symtab
}

func (symtab *symTab) addVar(name string) int {
	index, ok := symtab.names[name]
	if ok {
		return index
	}
	new_index := len(symtab.names)
	symtab.names[name] = new_index
	return new_index
}

func (symtab *symTab) getVar(name string) (int, int) {
	index, ok := symtab.names[name]
	if ok {
		return 0, index
	}
	if symtab.parent != nil {
		env, index := symtab.parent.getVar(name)
		if env >= 0 && index >= 0 {
			return env + 1, index
		}
	}
	return -1, -1
}

func (symtab *symTab) Dump(env *Env) {
	for name, index := range symtab.names {
		val := env.get(0, index)
		fmt.Printf("-> %s = %s\n", name, val)
	}
}
