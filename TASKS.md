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

### A4 · Fix `contains` operator — guard `DeepEqual` with type check

**File:** `internal/engine/condition.go:61-76`

`reflect.DeepEqual` is called unconditionally after the interface comparison fails,
even when types are different — guaranteed miss.

- [ ] Add `reflect.TypeOf(elem) == reflect.TypeOf(right)` guard before `DeepEqual` call
- [ ] Add test: `contains` on slice of mixed types returns correct result
- [ ] Run `go test ./...`

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

### A7 · Fix `ForloopDrop` — stack-allocate, avoid per-loop heap alloc  ⚠️ SYNC with B

**File:** `internal/tags/tag_for.go:106`

`drop := &ForloopDrop{...}` forces heap allocation per for-loop render.

- [ ] Change `drop` to a value type: `drop := ForloopDrop{Length: length}`
- [ ] Pass pointer to `ctx.Set("forloop", &drop)` — same address, no new alloc per iteration
- [ ] Confirm `ForloopDrop` fields are not retained after loop exits (check include/render tags)
- [ ] Run `go test ./...`

> **SYNC:** Coordinate with Dev B on A7+B4 if `ForloopDrop` is used in `tag_render.go` isolated subcontext.

**Expected gain:** -1 heap alloc per `{% for %}` tag render.

---

### A8 · Fix `Context` init — use nil slices for Errors/Warnings/interrupts

**File:** `internal/engine/context.go`

`NewContext` no longer pre-allocates `[]error{}` / `[]interface{}{}` — verified by code.
`interrupts` field remains as `[]interface{}` (nil zero value). Confirm callers are nil-safe.

- [ ] Verify `append(c.Errors, ...)` and `len(c.Errors)` are nil-safe (they are in Go)
- [ ] Confirm `interrupts` is never pre-allocated with `{}` in current code
- [ ] Run `go test ./...`

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

### B4 · Fix `SliceCollection` — lazy iterator for typed slices  ⚠️ SYNC with A7

**File:** `internal/engine/utils.go`, `internal/tags/tag_for.go`

For any `[]SomeType` (not `[]interface{}`), `SliceCollection` materializes a full
copy via reflection before the loop starts.

- [ ] Define interface: `type Iterable interface { Len() int; At(i int) interface{} }`
- [ ] Implement `reflectSliceIterable` wrapping a `reflect.Value`
- [ ] Implement `interfaceSliceIterable` as a fast path for `[]interface{}`
- [ ] Update `tag_for.go:RenderToOutputBuffer` to iterate via `Iterable` instead of `[]interface{}`
- [ ] Update `SliceCollection` or add `ToIterable` alongside it — preserve existing callers
- [ ] Run `go test ./...`

> **SYNC with A7** — both touch `tag_for.go`. Do after A7 is merged.

**Expected gain:** Eliminates O(n) reflect + slice alloc before loop body executes.

---

### B5 · Fix `Sort` filter — eliminate second intermediate slice allocation ✅

**File:** `internal/filters/standard_filters.go`

- [x] Replace `[]kv` approach with a pre-computed `keys []interface{}` + `indices []int`
- [x] Sort `indices` by `keys[indices[i]]` comparison
- [x] Apply permutation into a new `sorted` slice
- [x] Run `go test ./...` — sort output is stable

**Expected gain:** -1 full slice alloc per `| sort: "property"` call.

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

### S1 · Add benchmarks before starting any task  ⚠️ BOTH devs

Before any optimization, establish baselines. Without numbers, gains are guesses.

- [ ] **Dev A:** `BenchmarkConditionEval`, `BenchmarkVariableRender`, `BenchmarkFindVariable`
- [ ] **Dev B:** `BenchmarkTemplateCacheGet` (50 goroutines), `BenchmarkSortFilter`, `BenchmarkForLoop`
- [ ] Commit benchmark file: `bench_test.go` in relevant packages
- [ ] Record baseline in this file under **Baselines** section below

---

### S2 · Migrate Value fully in the filter dispatch path  ⚠️ BOTH devs — large task

**Files:** `internal/engine/strainer.go`, `variable.go`, `context.go`

The `Value` type was introduced but the pipeline still uses `interface{}` everywhere,
causing a double-conversion on every builtin filter call. Full migration eliminates this.

This is the highest-leverage but riskiest task. **Do last, after S1 baselines exist.**

- [ ] Agree on migration order: bottom-up (`FilterDispatcher.Invoke` → `Variable.Render` → `Context.Evaluate`)
- [ ] `Variable.Render` returns `Value` instead of `interface{}`
- [ ] `RenderContext.InvokeFilter` signature changes to `(method string, obj Value, args []Value) Value`
- [ ] `Context.Evaluate` returns `Value`
- [ ] Update all callers in `internal/tags/`
- [ ] Run full test suite + benchmarks
- [ ] Compare against S1 baselines

---

## Baselines

_Fill in after S1 is complete._

| Benchmark | Before | After | Delta |
|---|---|---|---|
| `BenchmarkConditionEval` | — | — | — |
| `BenchmarkVariableRender` | — | — | — |
| `BenchmarkFindVariable` | — | — | — |
| `BenchmarkTemplateCacheGet/50goroutines` | — | — | — |
| `BenchmarkSortFilter` | — | — | — |
| `BenchmarkForLoop/1000items` | — | — | — |

---

## Task Order (suggested)

```
Dev A:  S1 → A1 ✅ → A2 ✅ → A5 ✅ → A3 ✅ → A4 → A6 ✅ → A8 → A7* → S2
Dev B:  S1 → B3 ✅ → B6 ✅ → B5 ✅ → B7 ✅ → B1 ✅ → B2 ✅ → B4* → S2

* A7 and B4 touch tag_for.go — coordinate before starting.
  S2 requires both tracks complete.
```
