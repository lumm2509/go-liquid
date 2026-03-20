// Package liquid implements the Shopify Liquid template language for Go.
//
// # Basic usage
//
//	tmpl, err := liquid.Parse(`Hello {{ name }}!`, nil)
//	if err != nil {
//	    // parse errors: syntax errors, unknown tags, etc.
//	}
//	out, err := tmpl.Render(map[string]interface{}{"name": "World"}, nil)
//	// out == "Hello World!"
//
// # Custom environment
//
// Use ParseWithEnv to control tags, filters, and file system access:
//
//	env := liquid.BuildEnvironment(func(e *liquid.Environment) {
//	    e.RegisterFilter(MyFilters{})
//	    e.RegisterTag("mytag", myTagFactory)
//	    e.FileSystem = myFileSystem
//	})
//	tmpl, err := liquid.ParseWithEnv(source, env, nil)
//
// # Render options
//
// Use *RenderOptions to configure render-time behavior:
//
//	out, err := tmpl.Render(data, &liquid.RenderOptions{
//	    StrictVariables: true,  // error on undefined variables
//	    StrictFilters:   true,  // error on unknown filters
//	})
//
// # Security
//
// This engine does NOT auto-escape HTML output. If rendering user-provided
// content in an HTML context, always use the escape filter explicitly:
//
//	{{ user_input | escape }}
//
// # Concurrency
//
// A *Template is safe to use from multiple goroutines concurrently after Parse
// returns. Each call to Render creates its own isolated Context; no state is
// shared between concurrent renders.
//
// # Contracts
//
// Template is immutable after Parse. A parsed *Template holds no render state.
// All execution state lives in the Context, which is created and destroyed per
// Render call. As a result, Render is safe to call concurrently on the same
// *Template without any locking.
//
// Render is a pure function. It does not mutate the assigns map passed by the
// caller. The same input always produces the same output.
//
// Filters MUST be pure functions. They must not mutate their arguments.
// If a filter needs to transform a collection, it returns a copy.
// The engine does not perform defensive copies automatically.
//
// # Public API surface
//
// Entry points:   Parse, ParseWithEnv
// Core types:     Template, Environment, RenderOptions, ParseOptions
// Configuration:  BuildEnvironment, NewEnvironment, ErrorMode
// Extension:      TagFactory, FileSystem, Drop, DebugLogger, DebugEvent
// Error types:    see errors.go
//
// Everything else is implementation detail subject to change without notice.
//
// # Versioning
//
// This library is currently v0.x. The API may change between minor versions.
// Breaking changes are documented in the CHANGELOG. v1.0 will be released
// once Phases 0–5 are complete and spec compliance is documented.
//
// # Debug logging
//
// Environment accepts an optional DebugLogger. When nil (default), the engine
// produces no log output and allocates nothing extra in hot paths.
//
// Available events:
//
//   - "filter.not_found"              — filter not registered (StrictFilters=false)
//   - "condition.unknown_operator"    — unknown comparison operator
//   - "environment.frozen_tag_skipped"— RegisterTag called on a frozen Environment
//   - "context.overflow"              — scope nesting depth exceeded 100
//   - "render.node_error"             — a node produced an error during render
package liquid
