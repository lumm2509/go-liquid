# liquid

A Shopify Liquid template engine for Go.

> **v0.x** — API may change between minor versions. See [CHANGELOG](CHANGELOG.md) for breaking changes. v1.0 will be released once Phases 0–5 are complete and spec compliance is documented.

## Installation

```bash
go get github.com/go-liquid
```

## Examples

### 1. Basic rendering

```go
tmpl, err := liquid.Parse(`Hello {{ name | upcase }}!`, nil)
if err != nil {
    log.Fatal(err)
}

out, err := tmpl.Render(map[string]interface{}{
    "name": "world",
}, nil)
// out == "Hello WORLD!"
```

### 2. Loops and conditionals

```go
const src = `
{% for product in products %}
  {{ product.title }}{% if product.available %} — in stock{% endif %}
{% endfor %}`

tmpl, _ := liquid.Parse(src, nil)
out, _ := tmpl.Render(map[string]interface{}{
    "products": []map[string]interface{}{
        {"title": "Widget", "available": true},
        {"title": "Gadget", "available": false},
    },
}, nil)
```

### 3. Custom environment with filters and tags

```go
type MyFilters struct{}

func (f MyFilters) Shout(input interface{}) string {
    return strings.ToUpper(fmt.Sprintf("%v", input)) + "!!!"
}

env := liquid.BuildEnvironment(func(e *liquid.Environment) {
    e.RegisterFilter(MyFilters{})
})

tmpl, _ := liquid.ParseWithEnv(`{{ msg | shout }}`, env, nil)
out, _ := tmpl.Render(map[string]interface{}{"msg": "hello"}, nil)
// out == "HELLO!!!"
```

## Render options

```go
out, err := tmpl.Render(data, &liquid.RenderOptions{
    StrictVariables: true,  // error on undefined variables
    StrictFilters:   true,  // error on unknown filters
})
```

## Concurrency

A `*Template` is safe to use from multiple goroutines concurrently. Each call
to `Render` creates its own isolated `Context`; no state is shared between
concurrent renders.

## Security

This engine does **not** auto-escape HTML output. When rendering user-provided
content in an HTML context, use the `escape` filter explicitly:

```liquid
{{ user_input | escape }}
```

## Debug logging

Attach a `DebugLogger` to observe internal engine events without `fmt.Printf`:

```go
env := liquid.NewEnvironment()
env.Logger = liquid.StdoutLogger{}  // or your own implementation

tmpl, _ := liquid.ParseWithEnv(src, env, nil)
```

Available events: `filter.not_found`, `condition.unknown_operator`,
`environment.frozen_tag_skipped`, `context.overflow`, `render.node_error`.
