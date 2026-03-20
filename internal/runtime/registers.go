package runtime

// Registers maneja el almacenamiento de objetos del sistema (como el file_system o el template_factory).
type Registers struct {
	static  map[string]interface{}
	changes map[string]interface{}
}

// NewRegisters equivale a initialize.
// Si se le pasa otro objeto Registers, extrae su mapa estático.
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

// Set equivale a []=
// Guarda los cambios en el mapa local 'changes' para no afectar al 'static'.
func (r *Registers) Set(key string, value interface{}) {
	r.changes[key] = value
}

// Get equivale a []
// Busca primero en los cambios locales y, si no existe, en el mapa estático.
func (r *Registers) Get(key string) interface{} {
	if val, ok := r.changes[key]; ok {
		return val
	}
	return r.static[key]
}

// Delete elimina una llave de los cambios locales.
func (r *Registers) Delete(key string) {
	delete(r.changes, key)
}

// Fetch imita el comportamiento complejo de Ruby para recuperar valores con defaults.
func (r *Registers) Fetch(key string, defaultValue interface{}, block func() interface{}) interface{} {
	// 1. Buscar en cambios
	if val, ok := r.changes[key]; ok {
		return val
	}

	// 2. Si hay un bloque (callback), tiene prioridad sobre el default fijo
	if block != nil {
		if val, ok := r.static[key]; ok {
			return val
		}
		return block()
	}

	// 3. Buscar en estáticos con valor default
	if val, ok := r.static[key]; ok {
		return val
	}

	return defaultValue
}

// Key equivale a key?
func (r *Registers) Key(key string) bool {
	if _, ok := r.changes[key]; ok {
		return true
	}
	_, ok := r.static[key]
	return ok
}

// Static devuelve el mapa estático (attr_reader :static)
func (r *Registers) Static() map[string]interface{} {
	return r.static
}

// SetStatic permite modificar el mapa base (usado en la inicialización del Context)
func (r *Registers) SetStatic(key string, value interface{}) {
	if r.static == nil {
		r.static = make(map[string]interface{})
	}
	r.static[key] = value
}
