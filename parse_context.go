package liquid

import (
	"fmt"
)

// modos de error internos del ParseContext (implementación, no API pública)
const (
	errorModeLazy    = "lazy"
	errorModeWarn    = "warn"
	errorModeStrict  = "strict"
	errorModeStrict2 = "strict2"
)

type ParseContext struct {
	Locale         *I18n // Asumiendo estructura de I18n
	LineNumber     int
	TrimWhitespace bool
	Depth          int
	Partial        bool
	Warnings       []error
	ErrorMode      string
	Environment    *Environment // Asumiendo estructura de Environment

	// Privados para manejo interno
	templateOptions map[string]interface{}
	options         map[string]interface{}
	stringScanner   *StringScanner         // Estructura compartida para evitar re-alojar memoria
	expressionCache map[string]interface{} // Puede ser nil si está desactivado
	partialOptions  map[string]interface{} // Caché de opciones para parciales
}

// NewParseContext equivale a initialize
func NewParseContext(options map[string]interface{}) *ParseContext {
	// Copiar el mapa para no mutar el del caller (contrato de pureza).
	optsCopy := make(map[string]interface{}, len(options))
	for k, v := range options {
		optsCopy[k] = v
	}
	options = optsCopy

	// 1. Obtener Environment
	env, ok := options["environment"].(*Environment)
	if !ok {
		env = DefaultEnvironment()
	}

	pc := &ParseContext{
		Environment:     env,
		templateOptions: options,
		Warnings:        []error{},
		stringScanner:   NewStringScanner(""), // Scanner compartido inicializado vacío
		Depth:           0,
		Partial:         false,
	}

	// 2. Configurar Locale
	if loc, ok := options["locale"].(*I18n); ok {
		pc.Locale = loc
	} else {
		pc.Locale = NewI18n("")
		pc.templateOptions["locale"] = pc.Locale
	}

	// 3. Configurar Caché de expresiones
	pc.setupExpressionCache(options)

	// 4. Inicializar opciones activas
	pc.options = pc.templateOptions

	return pc
}

// Get permite acceso a las opciones (equivalente a [])
func (pc *ParseContext) Get(key string) interface{} {
	return pc.options[key]
}

// NewParser prepara el scanner compartido y devuelve un nuevo parser
func (pc *ParseContext) NewParser(input string) *Parser {
	pc.stringScanner.SetString(input)
	return NewParser(pc.stringScanner)
}

// SafeParseExpression lógica de parseo seguro
func (pc *ParseContext) SafeParseExpression(parser *Parser) (interface{}, error) {
	// Delegar al objeto Expression
	return ParseExpressionSafe(parser, pc.stringScanner, pc.expressionCache)
}

// ParseExpression equivale al método principal de parseo de markup
func (pc *ParseContext) ParseExpression(markup string, safe bool) (interface{}, error) {
	if !safe && pc.ErrorMode == errorModeStrict2 {
		return nil, fmt.Errorf("InternalError: unsafe parse_expression cannot be used in strict2 mode")
	}

	return ParseExpression(markup, pc.stringScanner, pc.expressionCache)
}

// SetPartial es el setter de partial (partial=) que maneja el cambio de opciones
func (pc *ParseContext) SetPartial(isPartial bool) {
	pc.Partial = isPartial
	if isPartial {
		pc.options = pc.getPartialOptions()
	} else {
		pc.options = pc.templateOptions
	}

	// Actualizar el modo de error basado en las opciones actuales o el environment
	if mode, ok := pc.options["error_mode"].(string); ok {
		pc.ErrorMode = mode
	} else {
		pc.ErrorMode = pc.Environment.ErrorMode
	}
}

// getPartialOptions (equivalente a partial_options) maneja el blacklist
func (pc *ParseContext) getPartialOptions() map[string]interface{} {
	if pc.partialOptions != nil {
		return pc.partialOptions
	}

	dontPass := pc.templateOptions["include_options_blacklist"]

	switch v := dontPass.(type) {
	case bool:
		if v {
			// Si es true, solo pasamos el locale
			pc.partialOptions = map[string]interface{}{
				"locale": pc.Locale,
			}
			return pc.partialOptions
		}
	case []string:
		// Si es un array, filtramos las llaves prohibidas
		pc.partialOptions = make(map[string]interface{})
		blacklist := make(map[string]bool)
		for _, key := range v {
			blacklist[key] = true
		}

		for k, val := range pc.templateOptions {
			if !blacklist[k] {
				pc.partialOptions[k] = val
			}
		}
		return pc.partialOptions
	}

	// Por defecto devolvemos todas las opciones
	pc.partialOptions = pc.templateOptions
	return pc.partialOptions
}

// setupExpressionCache lógica interna para inicializar el mapa de caché
func (pc *ParseContext) setupExpressionCache(options map[string]interface{}) {
	cacheOption := options["expression_cache"]

	if cacheOption == nil {
		pc.expressionCache = make(map[string]interface{})
		return
	}

	// Si es un map ya existente (respond_to? :[]), lo usamos
	if cache, ok := cacheOption.(map[string]interface{}); ok {
		pc.expressionCache = cache
		return
	}

	// Si es simplemente un valor true o algo no nulo, creamos uno nuevo
	if cacheOption != nil {
		pc.expressionCache = make(map[string]interface{})
	}
}
