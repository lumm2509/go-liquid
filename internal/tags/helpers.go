package tags

import "github.com/lumm2509/go-liquid/internal/engine"

// asContext type-asserts a RenderContext to *engine.Context.
// Returns nil if the assertion fails (should never happen in normal usage).
func asContext(ctx engine.RenderContext) *engine.Context {
	c, _ := ctx.(*engine.Context)
	return c
}
