package runtime

// Registers stores system objects (e.g. file_system, template_factory)
type Registers struct {
	static  map[string]interface{}
	changes map[string]interface{}
}

// NewRegisters creates a Registers; if passed a *Registers, reuses its static map
func NewRegisters(registers interface{}) *Registers {
	r := &Registers{
		changes: make(map[string]interface{}),
	}

	switch v := registers.(type) {
	case *Registers:
		r.static = v.static
	case map[string]interface{}:
		r.static = v
	default:
		r.static = make(map[string]interface{})
	}

	return r
}

// Set writes to the local changes map to avoid mutating static
func (r *Registers) Set(key string, value interface{}) {
	r.changes[key] = value
}

// Get checks changes first, then falls back to static
func (r *Registers) Get(key string) interface{} {
	if val, ok := r.changes[key]; ok {
		return val
	}
	return r.static[key]
}

func (r *Registers) Delete(key string) {
	delete(r.changes, key)
}

func (r *Registers) Fetch(key string, defaultValue interface{}, block func() interface{}) interface{} {
	if val, ok := r.changes[key]; ok {
		return val
	}

	if block != nil {
		if val, ok := r.static[key]; ok {
			return val
		}
		return block()
	}

	if val, ok := r.static[key]; ok {
		return val
	}

	return defaultValue
}

func (r *Registers) Key(key string) bool {
	if _, ok := r.changes[key]; ok {
		return true
	}
	_, ok := r.static[key]
	return ok
}

func (r *Registers) Static() map[string]interface{} {
	return r.static
}

// SetStatic modifies the base map; used during context initialization
func (r *Registers) SetStatic(key string, value interface{}) {
	if r.static == nil {
		r.static = make(map[string]interface{})
	}
	r.static[key] = value
}
