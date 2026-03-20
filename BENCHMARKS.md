# Benchmarks Baseline

Captured on: 2026-03-20
Go version: see `go version`
Machine: linux/amd64, 12 cores
Command: `go test -bench=. -benchmem -count=1`

## Baseline (pre-Fase 5 optimizations)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| ParseSimple | 3177 | 3458 | 40 |
| ParseWithFilters | 4178 | 3926 | 52 |
| RenderSimple | 727 | 1008 | 16 |
| RenderWithFilters | 3210 | 1584 | 41 |
| RenderForLoop100 | 32804 | 40248 | 327 |
| RenderCondition | 819 | 1024 | 17 |
| RenderNestedLookup | 741 | 1024 | 17 |
| RenderForLoopWithFilters | 135927 | 52192 | 1378 |

## Analysis

**Hotspots identified (by alloc pressure):**

1. 🔥 `RenderForLoopWithFilters` — 1378 allocs for 50 items × ~2 filters each.
   - ~13 allocs per filter invocation (reflection + slice allocations)
   - Root cause: `Strainer.Invoke` does `MethodByName` via reflection on every call
   - Fix: pre-build `name → reflect.Value` map in `StrainerTemplate` (5.2)

2. 🔥 `RenderForLoop100` — 327 allocs for 100 items × 1 variable
   - ~3 allocs per loop iteration (forloop map creation)
   - Fix: reuse forloop map, only update values (5.x)

3. 🟡 `RenderWithFilters` — 41 allocs for a single 2-filter expression
   - Filter dispatch overhead
   - Fix: same as #1 (5.2)

4. 🟢 `RenderSimple` / `RenderCondition` / `RenderNestedLookup` — 16-17 allocs each
   - Baseline render overhead (context creation, scope allocation)
   - Acceptable for now

## Post-optimization Results

_(Updated after each optimization)_

### After 5.2 — Pre-compiled filter dispatch

| Benchmark | ns/op | B/op | allocs/op | Δ allocs |
|-----------|------:|-----:|----------:|----------|
| RenderWithFilters | TBD | TBD | TBD | TBD |
| RenderForLoopWithFilters | TBD | TBD | TBD | TBD |

### After 5.4 — Primitive renderObjToOutput

| Benchmark | ns/op | B/op | allocs/op | Δ allocs |
|-----------|------:|-----:|----------:|----------|
| RenderSimple | TBD | TBD | TBD | TBD |
| RenderForLoop100 | TBD | TBD | TBD | TBD |

## Regression Policy

No commit may worsen any benchmark by >10% without explicit justification in the commit message.
