# liquid

A Shopify Liquid template engine for Go. Parses and renders Liquid templates — concurrently, and mostly correctly.

![](.github/gopher-surfer.jpg)

## what this is

- A Go implementation of [Shopify Liquid](https://shopify.github.io/liquid/) — variables, filters, control flow, loops, partials, the whole thing
- Immutable parsed templates: parse once, render from many goroutines without locking
- Rendering is a pure function — same inputs, same output, no mutation of your data
- Extensible: register custom filters and tags against an `Environment`

## why it exists

Needed a Liquid engine with a clear concurrency model. Built it around one constraint: `*Template` holds no render state, `Context` is created per-render and dies with it. That's basically the whole thesis. Whether it was worth it is left as an exercise.

## how to use

```bash
go get github.com/go-liquid
```

```go
tmpl, err := liquid.Parse(`Hello, {{ name | upcase }}!`, nil)
if err != nil {
    // parse error
}

out, err := tmpl.Render(map[string]interface{}{
    "name": "world",
}, nil)
// out → "Hello, WORLD!"
```

**With a custom environment:**

```go
type MyFilters struct{}

func (f MyFilters) Shout(input interface{}) string {
    return strings.ToUpper(fmt.Sprintf("%v", input)) + "!!!"
}

env := liquid.BuildEnvironment(func(e *liquid.Environment) {
    e.RegisterFilter(MyFilters{})
})

tmpl, _ := liquid.ParseWithEnv(`{{ msg | shout }}`, env, nil)
```

**With caching** (if you're parsing the same templates repeatedly):

```go
cache := liquid.NewTemplateCache(env)
tmpl, err := cache.Get("key", source)
```

**With context propagation and timeouts:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

out, err := tmpl.RenderWithContext(ctx, data, nil)
```

**Strict mode** (unknown variables and filters return errors instead of silently passing through):

```go
out, err := tmpl.Render(data, &liquid.RenderOptions{
    StrictVariables: true,
    StrictFilters:   true,
})
```

**Debug logging** (observe internal engine events without `fmt.Printf` scattered everywhere):

```go
env := liquid.NewEnvironment()
env.Logger = liquid.StdoutLogger{} // or your own implementation
```

## design notes

**Immutability.** `*Template` is read-only after parse. All execution state — variable scopes, loop counters, capture buffers — lives in a `Context` that's created and destroyed per `Render` call. This is not a recommendation; it's a hard contract enforced by the architecture.

**The cache is sharded.** `TemplateCache` uses 16 buckets with `maphash`-based routing. Reads are effectively lock-free under the common case. You shouldn't need to think about this, and frankly neither should we, but here we are.

**Lax by default.** Unknown variables resolve to `""`. Unknown filters pass input through unchanged. This matches Shopify's reference behavior. Flip to strict mode if you want errors instead.

**No auto-escape.** HTML output is not escaped automatically (matching original Liquid behavior). Use `{{ var | escape }}` explicitly, or enable it globally via `RenderOptions` if you want it on by default.

## what it is NOT

- Not a general-purpose expression evaluator. You can't call arbitrary Go functions or reach into runtime state from templates.
- Not a security sandbox. If you're rendering untrusted *templates* (not just untrusted data), that's a different problem entirely and not one we've thought hard about.
- Not v1. The API will probably change in ways that will mildly inconvenience you at some point.

## status

Work in progress. Core rendering is functional and passes the Shopify Liquid spec suite. Concurrent rendering is verified under `-race`. Error messages are honest but not always helpful. Documentation is, as you can tell, a work in progress too.
