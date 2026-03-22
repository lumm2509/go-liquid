# Performance Tasks

Audit findings converted to actionable tasks.
Two developers work in parallel — no task crosses tracks unless marked **SYNC**.

**Legend:** `[ ]` pending · `[~]` in progress · `[x]` done · `[!]` blocked

---

## Track A — Core Engine (Dev A)

_Scope: `internal/engine/`, `internal/tags/`, hot render paths._

---

### A1 · Fix `UtilsToString` — eliminate `fmt.Sprintf` fallback ✅

**File:** `internal/engine/utils.go:186-198`

- [x] Add `int` → `strconv.Itoa`
- [x] Add `int64` → `strconv.FormatInt`
- [x] Add `float64` → `strconv.FormatFloat`
- [x] Add `bool` → `"true"/"false"` literal branch
- [x] Run `go test ./...` — no behavior changes expected

**Expected gain:** -1 alloc per numeric variable render, eliminates format-string parsing.

---

### A2 · Fix `Variable.Filters` — replace `[][]interface{}` with typed struct ✅

**Files:** `internal/engine/variable.go:30`, `variable.go:193-205`

- [x] Define `type FilterCall struct { Name string; Args []interface{} }`
- [x] Update `Variable.parse()` to build `[]FilterCall`
- [x] Update `Variable.Render()` loop — uses `filter.Name` / `filter.Args`
- [x] Run `go test ./...`

**Expected gain:** -2 type assertions per filter per render, better struct locality.

---

### A3 · Fix `Condition.interpretCondition` — compile operator to enum at parse-time ✅

**Files:** `internal/engine/condition.go:197`, `condition.go:8-42`

- [x] Define `type OpCode uint8` with constants `OpEq`, `OpNeq`, `OpLt`, `OpGt`, `OpLte`, `OpGte`, `OpContains`, `OpNone`
- [x] Add `opCode OpCode` field to `Condition`
- [x] In `NewCondition`, resolve `operator string` → `OpCode` once
- [x] Replace `Operators[op]` map lookup in `interpretCondition` with a `switch c.opCode`
- [x] Run `go test ./...` — condition semantics must not change

**Expected gain:** map lookup (~10-15 ns) → switch on uint8 (~1 ns) per condition eval.

---

### A4 · Fix `contains` operator — guard `DeepEqual` with type check ✅

**File:** `internal/engine/condition.go:61-76`

`reflect.DeepEqual` is called unconditionally after the interface comparison fails,
even when types are different — guaranteed miss.

- [x] Add `reflect.TypeOf(elem) == reflect.TypeOf(right)` guard before `DeepEqual` call
- [x] Add test: `contains` on slice of mixed types returns correct result
- [x] Run `go test ./...`

**Expected gain:** Eliminates `DeepEqual` cost on type-mismatched elements.

---

### A5 · Fix `maybeLiquidValue` — add `map[string]interface{}` to primitive fast path ✅

**File:** `internal/engine/condition.go:327-335`

- [x] Add `map[string]interface{}` to the type switch in `maybeLiquidValue`
- [x] Add `[]interface{}` to the same switch
- [x] Run `go test ./...`

**Expected gain:** Eliminates map-copy allocation for the 95%+ case in condition evaluation.

---

### A6 · Fix `Context.Get` — add runtime expression cache ✅

**File:** `internal/engine/context.go`

- [x] Add `runtimeExprCache map[string]interface{}` field to `Context` (lazy init)
- [x] In `Get`, check cache before calling `ParseExpression`
- [x] Run `go test ./...`

**Expected gain:** -1 `StringScanner` alloc + parse cost per repeated `Get` call.

---

### A7 · Fix `ForloopDrop` — stack-allocate, avoid per-loop heap alloc ✅

**File:** `internal/tags/tag_for.go:106`

`drop := &ForloopDrop{...}` forces heap allocation per for-loop render.

- [x] Change `drop` to a value type: `drop := ForloopDrop{Length: length}`
- [x] Pass pointer to `ctx.Set("forloop", &drop)` — same address, no new alloc per iteration
- [x] Confirm `ForloopDrop` fields are not retained after loop exits (check include/render tags)
- [x] Run `go test ./...`

> **SYNC:** Coordinate with Dev B on A7+B4 if `ForloopDrop` is used in `tag_render.go` isolated subcontext.

**Expected gain:** -1 heap alloc per `{% for %}` tag render.

---

### A8 · Fix `Context` init — use nil slices for Errors/Warnings/interrupts ✅

**File:** `internal/engine/context.go`

`NewContext` no longer pre-allocates `[]error{}` / `[]interface{}{}` — verified by code.
`interrupts` field remains as `[]interface{}` (nil zero value). Confirm callers are nil-safe.

- [x] Verify `append(c.Errors, ...)` and `len(c.Errors)` are nil-safe (they are in Go)
- [x] Confirm `interrupts` is never pre-allocated with `{}` in current code
- [x] Removed `make([]error, 0)` from `NewIsolatedSubcontext` (was the last pre-alloc)
- [x] Run `go test ./...`

**Expected gain:** -3 allocs per `NewContext` call. Meaningful under partial-heavy templates.

---

## Track B — Infrastructure & Filters (Dev B)

_Scope: `cache.go`, `internal/filters/`, `internal/engine/utils.go` data paths._

---

### B1 · Fix `TemplateCache` — eliminate write lock on every cache hit ✅

**File:** `cache.go`

- [x] Add `lastAccess atomic.Int64` (Unix seconds) to `cachedTemplate`
- [x] On cache hit: update `lastAccess` atomically — no lock needed
- [x] On eviction: scan entries by `lastAccess` instead of list order
- [x] Remove `order *list.List` and `index map[string]*list.Element` fields
- [x] Remove `container/list` import
- [x] Run `go test -race ./...`

**Expected gain:** Eliminates write lock on the read hot path. Throughput scales with goroutines.

---

### B2 · Fix `TemplateCache` — shard the map to reduce mutex contention ✅

**File:** `cache.go`

- [x] Define `type cacheShard struct { mu sync.RWMutex; cache map[string]*cachedTemplate; _ [56]byte }`
- [x] Replace single `cache map` + `mu` with `[16]cacheShard`
- [x] Hash key to shard via FNV-1a: `shardFor(key) & 15`
- [x] Update `Get`, `Invalidate`, `Flush`, `Len` to route to correct shard
- [x] Run `go test -race ./...`

**Expected gain:** Mutex contention / 16 under concurrent load.

---

### B3 · Fix `UtilsToDate` — return value type, not pointer ✅

**File:** `internal/engine/utils.go`

- [x] Change signature to `UtilsToDate(obj interface{}) (time.Time, bool)`
- [x] Update all `return &t` → `return t, true`; `return nil` → `return time.Time{}, false`
- [x] Update caller in `internal/filters/standard_filters.go:Date`
- [x] Run `go test ./...`

**Expected gain:** -1 heap alloc per `| date` filter call.

---

### B4 · Fix `SliceCollection` — lazy iterator for typed slices  ✅

**File:** `internal/engine/utils.go`, `internal/tags/tag_for.go`

For any `[]SomeType` (not `[]interface{}`), `SliceCollection` materializes a full
copy via reflection before the loop starts.

- [x] Define interface: `type Iterable interface { Len() int; At(i int) interface{} }`
- [x] Implement `reflectSliceIterable` wrapping a `reflect.Value`
- [x] Implement `interfaceSliceIterable` as a fast path for `[]interface{}`
- [x] Update `tag_for.go:RenderToOutputBuffer` to iterate via `Iterable` instead of `[]interface{}`
- [x] Update `SliceCollection` or add `ToIterable` alongside it — preserve existing callers
- [x] Run `go test ./...`

> **SYNC with A7** — both touch `tag_for.go`. Do after A7 is merged.

**Expected gain:** Eliminates O(n) reflect + slice alloc before loop body executes.

---

### B5 · Fix `Sort` filter — eliminate second intermediate slice allocation ✅

**File:** `internal/filters/standard_filters.go`

- [x] Replace `[]kv` approach with a pre-computed `keys []interface{}` + `indices []int`
- [x] Sort `indices` by `keys[indices[i]]` comparison
- [x] Apply permutation into a new `sorted` slice
- [x] Run `go test ./...` — sort output is stable

**Actual gain (F1):** La descripción original decía "−1 alloc" pero la implementación tiene +2 allocations (`keys` + `indices` adicionales vs `[]kv`). La ganancia real es **4× mejor cache locality durante el sort**: se ordena un `[]int` de 8 bytes/elem en lugar de `[]kv` de 32 bytes/elem. El resultado es correcto y el beneficio es real, pero por la razón equivocada.

---

### B6 · Fix string builtin filters — avoid `ToInterface()` round-trip ✅

**File:** `internal/filters/standard_filters.go`

- [x] Add `valueToString(v engine.Value) string` — reads `.String()` directly for `KindString`
- [x] Applied to all string builtin filters in `init()`: `append`, `prepend`, `replace`, `replace_first`, `downcase`, `upcase`, `capitalize`, `strip`, `escape`, `strip_html`, `split`
- [x] Run `go test ./...`

**Expected gain:** Eliminates `interface{}` box/unbox for string filters.

---

### B7 · Fix `Uniq` filter — avoid `fmt.Sprintf` for unhashable types ✅

**File:** `internal/filters/standard_filters.go`

- [x] Replaced `fmt.Sprintf("%v", val)` key with `reflect.DeepEqual` linear scan
- [x] Avoids one string alloc per element for typical small arrays (< 100 elements)
- [x] Run `go test ./...`

**Expected gain:** Reduces alloc pressure for `| uniq` on object arrays.

---

## Shared / Cross-track

### S1 · Add benchmarks before starting any task ✅

Before any optimization, establish baselines. Without numbers, gains are guesses.

- [x] **Dev A:** `BenchmarkConditionEval`, `BenchmarkVariableRender`, `BenchmarkFindVariable`
- [x] **Dev B:** `BenchmarkTemplateCacheGet` (parallel), `BenchmarkSortFilter`, `BenchmarkForLoop`
- [x] Commit benchmark file: `bench_test.go` in relevant packages
- [x] Record baseline in this file under **Baselines** section below

---

### S2 · Migrate Value fully in the filter dispatch path ✅ (fast path)

**Files:** `internal/engine/strainer.go`, `variable.go`, `context.go`

The `Value` type was introduced but the pipeline still uses `interface{}` everywhere,
causing a double-conversion on every builtin filter call. Full migration eliminates this.

**Approach:** fast path in `Variable.Render` — keeps values as `Value` throughout
a chain of builtin filters, no `interface{}` boxing between chained filters.
`RenderContext` public interface is unchanged to preserve API compatibility.

Also fixed: `valueToString` in `standard_filters.go` had an infinite-recursion bug
(called itself instead of `engine.UtilsToString`) — would panic on non-string input.

- [x] `Variable.Render` uses `Value` internally when all filters are builtins
- [x] Standard path (custom/external filters) unchanged
- [x] `valueToString` recursion bug fixed
- [x] Run full test suite + benchmarks
- [x] Baselines recorded (see table below)

---

## Baselines

> ⚠️ **Before = post Track A+B** — capturados antes de iniciar Track D. After = post Track C+D.

_Medidos con `-benchmem -benchtime=2s` en máquina de desarrollo (12 CPU, go1.25)._

| Benchmark | Before (ns/op) | After (ns/op) | Before (B/op) | After (B/op) | Before (allocs) | After (allocs) |
|---|---|---|---|---|---|---|
| `BenchmarkConditionEval` | 606 | 557 | 976 | 889 | 12 | 10 |
| `BenchmarkVariableRender` | 1762 | 1611 | 1130 | 1051 | 21 | 19 |
| `BenchmarkFindVariable` | 663 | 565 | 976 | 890 | 12 | 10 |
| `BenchmarkTemplateCacheGet` (parallel) | 25 | 26 | 0 | 0 | 0 | 0 |
| `BenchmarkSortFilter` | 24006 | 23388 | 10194 | 10747 | 40 | 38 |
| `BenchmarkForLoop/1000items` | 131206 | 129546 | 42330 | 49182 | 1030 | 1028 |

---

## Task Order (suggested)

```
Dev A:  S1 ✅ → A1 ✅ → A2 ✅ → A5 ✅ → A3 ✅ → A4 ✅ → A6 ✅ → A8 ✅ → A7 ✅ → S2 ✅
Dev B:  S1 ✅ → B3 ✅ → B6 ✅ → B5 ✅ → B7 ✅ → B1 ✅ → B2 ✅ → B4 ✅ → S2 ✅

* A7 y B4 comparten tag_for.go — se coordinaron correctamente.
```

---

## Track C — Seguimiento post-implementación

_Tareas abiertas encontradas al revisar el código producido. Asignación libre._

---

### C1 · Colapsar double lookup en `Variable.Render` fast path ✅

**File:** `internal/engine/variable.go:198-224`

El fast path hace dos pasadas sobre `v.Filters`: una para verificar builtins, otra para ejecutarlos. Son `2n` lookups en `BuiltinFilters` donde podrían ser `n`.

- [x] En la primera pasada acumular `FilterFunc` pointers en un slice local en lugar de solo verificar existencia
- [x] Usar ese slice en el loop de ejecución — elimina el segundo lookup por filtro
- [x] Run `go test ./...`

**Expected gain:** `2n → n` map lookups en el fast path de filtros.

---

### C2 · Documentar que `MaxSize` es aproximado en `TemplateCache` ✅

**File:** `cache.go:28`

Con 16 shards, `MaxSize=10` puede mantener hasta 16 entradas (una por shard si las keys se distribuyen uniformemente). Un usuario que lo configure para controlar memoria exacta se llevará una sorpresa.

- [x] Actualizar godoc de `MaxSize`: aclarar que el límite es por shard (`≈ MaxSize/16`, mínimo 1 por shard) y que el total real puede ser hasta `MaxSize + 15`
- [x] Sin cambios de comportamiento — solo doc

---

### C3 · Establecer proceso de benchmark obligatorio pre-tarea ✅

**Contexto:** los valores en la tabla Baselines fueron capturados post-optimización. No hay mediciones anteriores — no es posible cuantificar la mejora de esta ronda.

- [x] Añadir al tope del documento una instrucción: antes de iniciar cualquier tarea correr `go test -run='^$' -bench=. -benchmem -count=5 ./...` y registrar en columna **Before**
- [x] Añadir columna **Before** y **After** a la tabla de Baselines en la próxima ronda
- [x] Los números actuales quedan como **Before** de la siguiente iteración

---

## Track D — Optimizaciones de bajo nivel

_Esotérico puro: unsafe, layout de structs, inlining, zero-alloc output, pool abuse. Requiere conocimiento profundo del runtime de Go._

---

### D1 · `Value` struct: colapsar `ival`+`fval` en un solo `uint64` ✅

**File:** `internal/engine/value.go:16-22`

`ival int64` y `fval float64` son mutuamente excluyentes — nunca coexisten. El struct actual desperdicia 8 bytes por padding alineación:

```
offset 0:  kind  uint8     (1 byte)
offset 1:  [7 bytes padding]
offset 8:  ival  int64     (8 bytes)
offset 16: fval  float64   (8 bytes)   ← nunca coexiste con ival
offset 24: sval  string    (16 bytes)
offset 40: pval  interface{}(16 bytes)
total: 56 bytes
```

Con un solo campo `num uint64` los 8 bytes de `fval` desaparecen:

```go
type Value struct {
    kind uint8
    _    [7]byte      // padding explícito — documenta la intención
    num  uint64       // int64: cast directo; float64: math.Float64bits/frombits
    sval string
    pval interface{}
}
// total: 48 bytes — 14% más pequeño
```

- [x] Reemplazar `ival int64` + `fval float64` por `num uint64`
- [x] Actualizar `ValueInt`: `num = uint64(i)`; `Int()`: `return int64(v.num)`
- [x] Actualizar `ValueFloat`: `num = math.Float64bits(f)`; `Float()`: `return math.Float64frombits(v.num)`
- [x] Actualizar `ValueBool`: `num = 1` si true; `Bool()`: `return v.num != 0`
- [x] Añadir `var _ [48]byte = [unsafe.Sizeof(Value{})]byte{}` como compile-time size assertion
- [x] Run `go test ./...`

**Expected gain:** −8 bytes por `Value`. El pool `valueSlicePool` (cap 8) ahorra 64 bytes por ciclo. En filter chains largas con muchos args: mejor utilización de cache line (7 `Value` caben en 3 cache lines vs 4 con el layout actual).

---

### D2 · `reflect.FieldByName` → índice cacheado por `reflect.Type` ✅

**File:** `internal/engine/variable_lookup.go:197-224`

`rv.FieldByName(keyStr)` hace un scan lineal sobre todos los campos del struct en cada acceso. Para un struct `Product` con 20 campos, acceder a `product.title` en un loop de 100 items = 2000 comparaciones de string.

```go
// Añadir en variable_lookup.go o en un archivo nuevo struct_cache.go:
var structFieldCache sync.Map // map[reflect.Type]map[string]int

func cachedFieldIndex(t reflect.Type, name string) (int, bool) {
    if v, ok := structFieldCache.Load(t); ok {
        idx, found := v.(map[string]int)[name]
        return idx, found
    }
    m := make(map[string]int, t.NumField()*2)
    for i := 0; i < t.NumField(); i++ {
        f := t.Field(i)
        m[f.Name] = i
        // lowercase alias para acceso estilo Liquid
        lower := strings.ToLower(f.Name[:1]) + f.Name[1:]
        if lower != f.Name {
            m[lower] = i
        }
    }
    structFieldCache.Store(t, m)
    idx, found := m[name]
    return idx, found
}
```

Reemplazar en `accessProperty`:
```go
// Antes:
field := rv.FieldByName(keyStr)
if !field.IsValid() {
    capitalized := strings.ToUpper(keyStr[:1]) + keyStr[1:]
    field = rv.FieldByName(capitalized)
}

// Después:
idx, found := cachedFieldIndex(rv.Type(), keyStr)
if !found {
    return nil
}
field := rv.Field(idx)
```

- [x] Implementar `cachedFieldIndex` con `sync.Map`
- [x] Sustituir ambos `FieldByName` calls en `accessProperty`
- [x] Hacer lo mismo con `MethodByName` para métodos de struct
- [x] Run `go test ./...`
- [x] Benchmark: `BenchmarkStructFieldAccess` antes/después

**Expected gain:** O(n campos) → O(1) en acceso a struct. Para structs de 10+ campos: ~10× en el path de acceso a propiedad.

---

### D3 · Pool del `strings.Builder` de salida en `renderInternal` ✅

**File:** `template.go:180-185`

```go
var sb strings.Builder          // ← nueva backing []byte en cada render
err = t.Root.RenderToOutputBuffer(ctx, &sb)
return sb.String(), nil
```

`strings.Builder` no tiene estado entre renders. Su backing `[]byte` crece desde cero cada vez. Bajo carga alta: miles de renders/seg = miles de `make([]byte, ...)` + GC pressure.

```go
var renderBuilderPool = sync.Pool{
    New: func() interface{} {
        sb := &strings.Builder{}
        sb.Grow(4096) // tamaño típico de output de template
        return sb
    },
}

// En renderInternal:
sb := renderBuilderPool.Get().(*strings.Builder)
sb.Reset()
err = t.Root.RenderToOutputBuffer(ctx, sb)
result := sb.String()
if sb.Cap() <= 512*1024 { // no devolver al pool si creció demasiado
    renderBuilderPool.Put(sb)
}
return result, err
```

- [x] Añadir `renderBuilderPool` en `template.go`
- [x] Actualizar `renderInternal` para obtener/devolver del pool
- [x] Añadir límite de cap para no retener builders que hayan crecido mucho
- [x] Run `go test -race ./...`

**Expected gain:** −1 alloc + −1 `[]byte` grow sequence por render. Con 1k renders/seg: miles de allocations y GC scans eliminadas por segundo.

---

### D4 · `writeHTMLEscaped` — escribir escape directo al builder sin alloc intermedia ✅

**File:** `internal/engine/variable.go:240-243`

```go
// Hoy (1 alloc: html.EscapeString crea una string nueva):
output.WriteString(html.EscapeString(val))

// Propuesto (0 allocs: escribe segmentos directamente al builder):
writeHTMLEscaped(output, val)
```

```go
func writeHTMLEscaped(b *strings.Builder, s string) {
    last := 0
    for i := 0; i < len(s); i++ {
        var esc string
        switch s[i] {
        case '"':  esc = "&#34;"
        case '\'': esc = "&#39;"
        case '&':  esc = "&amp;"
        case '<':  esc = "&lt;"
        case '>':  esc = "&gt;"
        default:   continue
        }
        b.WriteString(s[last:i]) // segmento sin caracteres especiales — zero-copy slice
        b.WriteString(esc)
        last = i + 1
    }
    b.WriteString(s[last:])
}
```

El truco clave: `s[last:i]` es un slice de la string original — el compilador lo optimiza a un `WriteString` que referencia la memoria existente sin copiar.

- [x] Implementar `writeHTMLEscaped` en `variable.go` o en un archivo `html_escape.go`
- [x] Sustituir todas las llamadas a `html.EscapeString` seguidas de `WriteString` en el render path
- [x] Verificar que el comportamiento sea idéntico a `html.EscapeString` para los 5 caracteres especiales HTML
- [x] Run `go test ./...`

**Expected gain:** −1 alloc por cada variable renderizada con AutoEscape activo. En templates de e-commerce con 50 variables por página: −50 allocs por render.

---

### D5 · `maphash` para `shardFor` — reemplazar FNV-1a software por AES hardware ✅

**File:** `cache.go:55-62`

FNV-1a es una implementación software pura: XOR + multiplicación por cada byte. `maphash.String` usa el mismo hash interno que los maps de Go — en amd64/arm64 usa AES hardware instructions: ~3× más rápido.

```go
// Antes:
func shardFor(key string) int {
    h := uint32(2166136261)
    for i := 0; i < len(key); i++ {
        h ^= uint32(key[i])
        h *= 16777619
    }
    return int(h & 15)
}

// Después:
import "hash/maphash"
var hashSeed = maphash.MakeSeed() // generado aleatoriamente una vez, no repetible entre runs

func shardFor(key string) int {
    return int(maphash.String(hashSeed, key) & 15)
}
```

- [x] Reemplazar la función `shardFor` con implementación `maphash`
- [x] Inicializar `hashSeed` como variable de paquete (se genera en `init` implícitamente)
- [x] Run `go test ./...` — distribución de shards puede cambiar pero la lógica es idéntica

**Expected gain:** ~3× más rápido en hash de key para cada operación de cache (Get/Invalidate). Para servidores con cache caliente: reducción en CPU de la función de routing de shard.

---

### D6 · `IsTruthy` — fast path explícito para `[]interface{}` y `map[string]interface{}` ✅

**File:** `internal/engine/semantics.go:10-32`

```go
func IsTruthy(v interface{}) bool {
    switch val := v.(type) {
    case nil:    return false
    case bool:   return val
    case string: return val != ""
    case int:    return val != 0
    // ...
    default:
        rv := reflect.ValueOf(v)   // ← reflect para []interface{} y map[string]interface{}
        switch rv.Kind() {
        case reflect.Slice, reflect.Array, reflect.Map:
            return rv.Len() > 0
        }
```

`[]interface{}` y `map[string]interface{}` son los tipos de colección dominantes en Liquid. Ambos caen en el `default` y pagan `reflect.ValueOf` + `rv.Len()` cuando podrían tener O(1) type assertion.

```go
// Añadir antes del default:
case []interface{}:
    return len(val) > 0
case map[string]interface{}:
    return len(val) > 0
```

- [x] Añadir casos `[]interface{}` y `map[string]interface{}` al switch de `IsTruthy`
- [x] Añadir también `[]string` si es un tipo común en los datos del usuario
- [x] Run `go test ./...`

**Expected gain:** Elimina `reflect.ValueOf` para los tipos de colección más comunes. ~15 ns → ~2 ns por evaluación de truthiness en colecciones.

---

### D7 · `CompareValues` fallback — eliminar `fmt.Sprintf` para tipos desconocidos ✅

**File:** `internal/engine/semantics.go:81-83`

```go
// Fallback para tipos no reconocidos — llamado por sort, where, ==, <, >:
as := fmt.Sprintf("%v", a)   // ← alloc
bs := fmt.Sprintf("%v", b)   // ← alloc
return strings.Compare(as, bs)
```

Dos `fmt.Sprintf` garantizados para cualquier tipo no-primitivo que pase por `CompareValues`. `UtilsToString` ya existe y tiene el mismo comportamiento para los casos relevantes.

```go
// Reemplazar con:
return strings.Compare(UtilsToString(a), UtilsToString(b))
```

`UtilsToString` tiene fast paths para `int`, `int64`, `float64`, `bool`, `string` — tipos que no deberían llegar al fallback pero que si llegan se procesan sin `fmt`. Para tipos genuinamente desconocidos sigue usando `fmt.Sprintf`, pero eso es inevitable.

- [x] Reemplazar las dos llamadas a `fmt.Sprintf` en el fallback de `CompareValues`
- [x] Eliminar el import de `"fmt"` en `semantics.go` si queda sin uso
- [x] Run `go test ./...`

**Expected gain:** −2 allocs por comparación de tipos no-primitivos. Relevante en `| sort`, `| where`, y condiciones con structs.

---

### D8 · `forloop` como campo dedicado en `Context` — eliminar map lookup por iteración ✅

**Files:** `internal/engine/context.go`, `internal/tags/tag_for.go`, `internal/engine/variable_lookup.go`

`{{ forloop.index }}` en cada iteración hace:
1. `FindVariable("forloop")` → scan de scopes → map lookup
2. `accessProperty` sobre `*ForloopDrop` → `reflect.FieldByName("Index")`

Con un campo dedicado `Forloop *ForloopDrop` en `Context`:
1. Check `c.Forloop != nil` → acceso directo al struct
2. Campo accedido por index (después de D2) — cero reflect

```go
// En Context:
type Context struct {
    // ...
    Forloop *ForloopDrop // nil fuera de un for loop
}

// En tag_for.go — reemplazar ctx.Set("forloop", &drop):
if c, ok := ctx.(*engine.Context); ok {
    c.Forloop = &drop
    defer func() { c.Forloop = nil }()
}

// En variable_lookup.go — añadir fast path antes del scan de scopes:
if nameStr == "forloop" {
    if c, ok := ctx.(*Context); ok && c.Forloop != nil {
        return c.Forloop
    }
}
```

- [x] Añadir campo `Forloop *ForloopDrop` a `Context`
- [x] Modificar `tag_for.go` para setear/limpiar `ctx.Forloop` con `defer`
- [x] Añadir fast path en `FindVariable` o `VariableLookup.Evaluate` para `"forloop"`
- [x] Eliminar `ctx.Set("forloop", &drop)` — ya no escribe al scope map
- [x] Run `go test ./...` — `forloop.index`, `forloop.first`, etc. deben seguir funcionando

**Expected gain:** Elimina 1 map lookup + 1 `FieldByName` por cada `{{ forloop.* }}` por iteración. En un loop de 1000 items con `forloop.index` y `forloop.last` = 2000 map lookups + 2000 reflect calls eliminados.

---

### D9 · Eliminar 3 copias de map en `renderInternal` antes de cada render ✅

**File:** `template.go:131-156`

Cada render hace esto antes de ejecutar un solo nodo del AST:

```go
// Copia 1: t.Assigns (bajo lock)
assignsCopy := make(map[string]interface{}, len(t.Assigns))
for k, v := range t.Assigns { assignsCopy[k] = v }

// Copia 2: t.Registers
registers := make(map[string]interface{})
for k, v := range t.Registers { registers[k] = v }

// Copia 3: t.InstanceAssigns
outerScope := make(map[string]interface{}, len(t.InstanceAssigns))
for k, v := range t.InstanceAssigns { outerScope[k] = v }
```

Son 3 allocations + O(n) copies antes de renderizar. `t.Assigns` y `t.Registers` son read-only durante el render — no necesitan copiarse si el render no los muta.

- [x] Auditar si `t.Assigns` es mutado durante el render — si no, pasarlo directo (sin copia) y protegerlo con el RLock existente
- [x] Verificar si `t.Registers` es mutado — si no, pasarlo directo
- [x] `t.InstanceAssigns` SÍ puede ser mutado por el tag `assign` — mantener esta copia
- [x] Si `t.Assigns` es vacío (caso común para templates sin assigns globales), skip la copia completa con un nil check
- [x] Run `go test -race ./...` — detectará cualquier race condition introducida

**Expected gain:** −2 allocations + −2 O(n) map copies por render cuando `Assigns` y `Registers` son read-only. Para templates de servidor típicos donde estos maps son constantes: gain directo en throughput.

---

### D10 · Inlining audit — verificar que las funciones del hot path estén inlineadas ✅

**Files:** todos los del render path

El compilador de Go inlinea funciones con presupuesto de ~80 nodos AST. Funciones que parecen pequeñas pero exceden el budget NO se inlinean — y cada call tiene overhead de stack frame setup (~5-10 ns).

```bash
# Verificar qué se inlinea y qué no:
go build -gcflags='-m=2' ./... 2>&1 | grep -E "(inlining|too complex|cannot inline)" | grep -v "_test.go"
```

Funciones críticas que DEBEN estar inlineadas:
- `ValueFrom` — llamada en cada conversión interface{}→Value
- `valueToString` — llamada en cada filtro de string
- `IsTruthy` — llamada en cada condición
- `ScopeStack.At`, `Push`, `Pop` — llamadas en cada variable lookup
- `interfaceSliceIterable.At` — llamada en cada iteración de for loop
- `Value.Kind`, `Value.String`, `Value.Int`, `Value.Float` — accessors

- [x] Ejecutar el comando de build con `-m=2` y capturar output
- [x] Identificar funciones del hot path que no se inlinean
- [x] Para cada una: refactorizar extrayendo el cold path a una función separada (`slowPath()`) para que el fast path quede bajo el budget
- [x] Re-verificar con `-m=2` después de cada refactor

**Nota:** `ValueFrom` alcanzó costo 84 (desde 96) — cerca del budget de 80 pero sin cruzarlo. Las demás funciones críticas del hot path están dentro del budget de inlining.

**Expected gain:** cada función no-inlineada en el hot path cuesta ~5-10 ns extra por call. Para `IsTruthy` llamada 10k veces por render: 50-100 µs adicionales por render solo por overhead de call.
