package narfscript

type Env struct {
	parent *Env
	vals   []Value
}

func newEnv(parent *Env, size int) *Env {
	return &Env{
		parent: parent,
		vals:   make([]Value, size),
	}
}

func (env *Env) size() int {
	return len(env.vals)
}

func (env *Env) grow(val Value) int {
	index := len(env.vals)
	env.vals = append(env.vals, val)
	return index
}

func (env *Env) set(e, i int, val Value) bool {
	if e == 0 {
		if i < 0 || i >= len(env.vals) {
			return false
		}
		env.vals[i] = val
		return true
	}
	if env.parent != nil {
		return env.parent.set(e-1, i, val)
	}
	return false
}

func (env *Env) get(e, i int) Value {
	if e == 0 {
		if i < 0 || i >= len(env.vals) {
			return nil
		}
		return env.vals[i]
	}
	if env.parent != nil {
		return env.parent.get(e-1, i)
	}
	return nil
}
