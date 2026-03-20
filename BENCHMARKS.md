# Benchmarks

Captured on: 2026-03-20
Go version: see `go version`
Machine: linux/amd64, 12 cores
Command: `go test -bench=. -benchmem -count=1`

## Baseline (pre-Fase 5)

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

## Post-Fase 5 Results

| Benchmark | ns/op | B/op | allocs/op | Δ ns | Δ allocs |
|-----------|------:|-----:|----------:|-----:|----------|
| ParseSimple | 3426 | 3506 | 41 | ≈0 | ≈0 |
| ParseWithFilters | 4227 | 3976 | 53 | ≈0 | ≈0 |
| RenderSimple | 662 | 1008 | 16 | -9% | 0 |
| **RenderWithFilters** | **1767** | **1248** | **29** | **-45%** | **-29%** |
| **RenderForLoop100** | **18710** | **6984** | **129** | **-43%** | **-61%** |
| RenderCondition | 815 | 1024 | 17 | ≈0 | 0 |
| RenderNestedLookup | 775 | 1024 | 17 | ≈0 | 0 |
| **RenderForLoopWithFilters** | **73705** | **19696** | **729** | **-46%** | **-47%** |

## Optimizations Applied

| Task | Change | Impact |
|------|--------|--------|
| 5.2 | Pre-compiled filter dispatch (`StrainerTemplate.combined` map + global type cache) | RenderWithFilters -45% |
| 5.3 | Skip `ToLiquidValue` for primitives in condition eval | Micro, within noise |
| 5.4 | `strconv` instead of `fmt.Sprintf` for int/float64/bool rendering | Micro for these benchmarks |
| 5.5 | Type-based filter cache key (no value serialization) | Setup path only |
| 5.6 | Reuse forloop/tablerow maps across iterations | ForLoop100 -61% allocs |

## Regression Policy

No commit may worsen any benchmark by >10% without explicit justification in the commit message.
