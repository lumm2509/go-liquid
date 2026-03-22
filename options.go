package liquid

type ErrorMode int

const (
	// ErrorModeLax (default): missing variables return "", unknown filters pass input unchanged
	ErrorModeLax ErrorMode = iota
	// ErrorModeStrict: any runtime error stops the render and returns an error
	ErrorModeStrict
)

// RenderOptions configures Template.Render behavior. All fields default to zero = safe defaults.
type RenderOptions struct {
	// StrictVariables returns an error on undefined variable access
	StrictVariables bool

	// StrictFilters returns an error when an unregistered filter is used
	StrictFilters bool

	RethrowErrors bool

	// DisableAutoEscape disables HTML auto-escaping (on by default); use {{ v | raw }} to emit safe HTML
	DisableAutoEscape bool

	// Registers passes extra state accessible from custom tags
	Registers map[string]interface{}
}

type ParseOptions struct {
	ErrorMode ErrorMode
	Locale    *I18n
}

func (o *ParseOptions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	switch o.ErrorMode {
	case ErrorModeStrict:
		m["error_mode"] = "strict"
	default:
		m["error_mode"] = "lax"
	}
	if o.Locale != nil {
		m["locale"] = o.Locale
	}
	return m
}

// renderOptionsFromMap converts the legacy map format; used by RenderWithMap
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
