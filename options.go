package liquid

// ErrorMode controla cómo el engine maneja errores en tiempo de render.
type ErrorMode int

const (
	// ErrorModeLax (default): errores no-fatales no detienen el render.
	// Variables no encontradas devuelven "", filtros no encontrados pasan el input.
	ErrorModeLax ErrorMode = iota
	// ErrorModeStrict: cualquier error de runtime detiene el render y retorna error.
	ErrorModeStrict
)

// RenderOptions configura el comportamiento de Template.Render.
// Todos los campos tienen un valor cero que corresponde al comportamiento por defecto.
type RenderOptions struct {
	// StrictVariables retorna error si se accede a una variable no definida.
	// Default: false (variable no encontrada devuelve "").
	StrictVariables bool

	// StrictFilters retorna error si se usa un filtro no registrado.
	// Default: false (filtro no encontrado pasa el input sin modificar).
	StrictFilters bool

	// RethrowErrors convierte panics internos en errors retornados.
	// Default: false.
	RethrowErrors bool

	// Registers permite pasar estado adicional accesible desde tags custom.
	Registers map[string]interface{}
}

// ParseOptions configura el comportamiento de Parse / ParseWithEnv.
type ParseOptions struct {
	// ErrorMode controla cómo se reportan errores de sintaxis.
	ErrorMode ErrorMode

	// Locale configura las traducciones para tags que soportan i18n.
	Locale *I18n
}

// renderOptionsFromMap convierte el formato legacy map[string]interface{}
// a *RenderOptions. Para uso interno de RenderWithMap.
func renderOptionsFromMap(m map[string]interface{}) *RenderOptions {
	if m == nil {
		return nil
	}
	opts := &RenderOptions{}
	if v, ok := m["strict_variables"].(bool); ok {
		opts.StrictVariables = v
	}
	if v, ok := m["strict_filters"].(bool); ok {
		opts.StrictFilters = v
	}
	if v, ok := m["rethrow_errors"].(bool); ok {
		opts.RethrowErrors = v
	}
	if v, ok := m["registers"].(map[string]interface{}); ok {
		opts.Registers = v
	}
	return opts
}
