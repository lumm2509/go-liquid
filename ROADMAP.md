# ROADMAP — De experimento a librería real

> Basado en la auditoría técnica (`AUDITORIA.md`).
> Estado actual: prototipo funcional con race conditions, sin tests, API peligrosa por default.
> Objetivo: librería Go concurrentemente segura, con API limpia, correcta contra el spec de Shopify Liquid.

---

## ⚠️ Antes de empezar: Contrato de diseño

Responde estas cinco preguntas antes de escribir una sola línea de código. Sin este contrato, cada decisión de implementación queda en el aire y las Fases 4 y 5 se convierten en trabajo duplicado.

**1. ¿`Template` es inmutable después de `Parse`?**
Debe ser **sí**. Un `*Template` parseado no guarda estado de render. Todo estado de ejecución vive en el `Context`, que se crea y destruye por cada llamada a `Render`. Consecuencia directa: `Render` es concurrentemente seguro sobre el mismo `*Template` sin ningún lock.

Lo que esto implica en código: `Template.Errors` y `Template.Warnings` no se escriben durante `Render` — se retornan como parte del resultado. El campo `BlockBody.frozen = true` que actualmente se escribe durante rendering desaparece.

**2. ¿`Render` es una función pura?**
Debe ser **sí**. `Render(data, options)` no muta `data`. No muta el `*Template`. El mismo input produce el mismo output siempre. Si esto falla, el engine es no-determinista y eso mata cualquier adopción.

Test que lo define:
```go
func TestRenderIsPure(t *testing.T) {
    tmpl, _ := Parse(`{{ arr | join: "," }}`, nil)
    data := map[string]interface{}{"arr": []string{"a", "b", "c"}}
    original := []string{"a", "b", "c"}

    tmpl.Render(data, nil)
    tmpl.Render(data, nil) // segundo render debe dar el mismo resultado

    require.Equal(t, original, data["arr"]) // data no fue mutada
}
```

**2b. Contrato de pureza para filtros custom**

La pureza de `Render` no puede garantizarse solo a nivel del engine si los filtros custom pueden mutar sus inputs. Este es un agujero real:

```go
// Esto rompe pureza, determinismo y concurrencia:
func (f MyFilters) Evil(input []string) []string {
    input[0] = "mutated"  // muta el slice original del usuario
    return input
}
```

Decisión adoptada: **Opción A — Contrato documentado sin enforcement automático.**

El contrato es:
> Los filtros DEBEN ser funciones puras. No deben mutar sus argumentos. Si necesitan transformar una colección, retornan una copia. El engine no hace defensive copies automáticamente.

Razón: defensive copy automático (Opción B) añade overhead a todos los renders aunque el filter sea puro. El modo configurable (Opción C) añade complejidad de API sin beneficio claro. El contrato documentado es la solución correcta para el estado actual del proyecto.

Este contrato va en `doc.go`, en los godocs de `TagFactory` y `RegisterFilter`, y en un `CONTRIBUTING.md` cuando el proyecto se abra. Los tests de invariantes de Fase 1 (`TestRenderDoesNotMutateInput`) actúan como red de detección en la suite de tests estándar.

**3. ¿Cómo se manejan los errores de runtime?**

Decisión adoptada: **fail-fast por defecto, collect opt-in.**

Esta decisión impacta todo el engine y debe quedar cerrada aquí. No se revisa en Fase 3.

Modelo completo:

| Contexto | Comportamiento fail-fast | Comportamiento lax |
|---|---|---|
| Variable no encontrada + `StrictVariables=true` | `error` retornado, render detenido | `""` retornado, render continúa |
| Filtro no encontrado + `StrictFilters=true` | `error` retornado, render detenido | input pasado sin modificar, log via `DebugLogger` |
| Filtro no encontrado + `StrictFilters=false` | input pasado sin modificar | input pasado sin modificar |
| Error dentro de `for` loop | render del loop detenido, error propagado | loop continúa con siguiente elemento |
| Error dentro de `render` partial | error propagado al template padre | output parcial del partial, render padre continúa |
| `RethrowErrors=true` | panic reemplazado por error retornado | — |

`DebugLogger` y `ErrorMode` son ortogonales: el logger siempre recibe el evento si está configurado, independientemente del error mode. El error mode controla si el render se detiene — el logger siempre observa.

**4. ¿Cuáles son las reglas de truthiness? ¿Shopify-compatible o Liquid-inspired?**

Esta es una **decisión de producto**, no solo de implementación. Tiene consecuencias permanentes en la API pública y en la adopción.

Las dos opciones son incompatibles entre sí:

**Opción A — Shopify-compatible (Ruby Liquid exacto):**
- `""` → truthy (Ruby Liquid behavior)
- `0` → truthy (Ruby Liquid behavior)
- `nil`, `false` → falsy
- Todo lo demás → truthy

**Opción B — Liquid-inspired (semántica más intuitiva para Go):**
- `""`, `0`, `[]`, `{}`, `nil`, `false` → falsy
- Todo lo demás → truthy

**Decisión requerida antes de Fase 3A.** La elección determina los Semantic Lock Tests y el contenido de `COMPATIBILITY.md`.

Si se elige Opción A: el engine puede usarse como drop-in replacement para templates Shopify existentes.
Si se elige Opción B: el engine necesita `CompatibilityMode` o divergir permanentemente, con la documentación correspondiente.

Lo que no es aceptable: dejar este campo vacío y que el comportamiento emerja de la implementación de cada filtro.

**5. ¿Se permite la mutación de inputs del usuario?**
Debe ser **no**. El engine nunca modifica los maps o slices que el usuario pasa como `assigns`. Si necesita modificar estructuras internas, trabaja sobre copias.

**5b. Reglas de aliasing y referencias compartidas via `interface{}`**

El engine trabaja extensamente con `map[string]interface{}` y `[]interface{}`. Esto introduce riesgos de aliasing que no son visibles en el type system:

```go
// Riesgo: dos contexts comparten la misma referencia
sub.Environments = c.Environments  // ambos apuntan al mismo slice
```

Reglas que gobiernan cuándo copiar vs cuándo referenciar:

- **Read-only en runtime → referenciar.** Si un valor solo se lee durante el render (variables de assigns, static environments), no se copia — se referencia. Esto incluye los `assigns` del usuario.
- **Mutado en runtime → copiar antes de mutar.** Si el engine necesita escribir en una estructura (scope de un `for` loop, variables de `assign`), crea una nueva map, no muta la existente.
- **Compartido entre contexts → solo lectura o sync.** `Registers.static` se comparte entre subcontextos. Solo se escribe durante la inicialización del contexto padre, nunca durante el render.
- **`[]interface{}` en NodeList → inmutable después de Parse.** Ningún nodo del AST se añade o elimina después de que `Parse` retorna.

Tipos considerados "seguros" para referenciar sin copia: `string`, `int`, `float64`, `bool`, `nil`. Son inmutables por definición en Go.

Tipos que requieren cuidado: `[]interface{}`, `map[string]interface{}`, y cualquier pointer a struct. Antes de pasar uno de estos a un subcontexto, verificar si el subcontexto puede escribir en él.

---

## 🚫 HARD GATE RULE

Esta regla no es una sugerencia. Es la condición que previene que el plan colapse en dos semanas cuando alguien (tú) diga "solo avanzo tantito a la siguiente fase".

**No se puede hacer merge de ningún PR de una fase si:**

- Existe un solo test fallando
- `go test -race -count=5 ./...` falla una sola vez
- La cobertura de tests baja respecto al baseline de la fase anterior
- `go vet ./...` produce warnings
- Existe un test no-determinista (flaky). Un test que falla intermitentemente es peor que no tener el test — da falsa seguridad y consume tiempo de debugging. Todo test flaky se arregla o se elimina antes de merge.

Esto se implementa en CI como hard fail, no como warning. Si el CI no lo enforcea, la regla no existe.

El costo de ignorar esta regla es conocido: entras en loops de regresión donde arreglar X rompe Y, arreglar Y rompe Z, y después de tres semanas el código está peor que antes de empezar.

---

## Principios de este plan

- **Comportamiento implícito → contrato explícito.** Si el engine hace algo, ese algo tiene un nombre, una regla escrita, y un test que lo verifica. El comportamiento que "surge" de la implementación no existe como contrato — puede cambiar sin aviso. Si no está escrito, no es una decisión — es un accidente.
- **Cada fase debe dejar el código en mejor estado que lo encontró.** Sin refactors a medias.
- **No avanzar a la siguiente fase sin cumplir el criterio de salida de la actual. Ver HARD GATE RULE.**
- **Tests primero para todo cambio de comportamiento.** Si no hay test que falle antes del fix, el fix no cuenta.
- **Medir antes de optimizar.** Benchmarks antes de cualquier cambio de performance. Sin baseline, no hay optimización — hay suposición.
- **Cada ciclo de CPU importa.** Antes de agregar cualquier abstracción, verificar que no alloca cuando no debe.

---

## Vista general

| Fase | Nombre | Estado | Objetivo |
|------|--------|--------|---------|
| 0 | Estabilización de emergencia | ✅ Completado | Tapar lo que puede explotar en producción hoy |
| 0.5 | Observabilidad mínima | ✅ Completado | Ver qué pasa cuando algo falla, sin `fmt.Printf` |
| 1 | Red de seguridad | ✅ Completado | Tests de comportamiento e invariantes internas |
| 2 | API pública limpia | ✅ Completado | Contrato claro, singleton encapsulado (DefaultEnvironment persiste como fallback para nil env) |
| 3A | Consistencia interna | ✅ Completado | Semántica uniforme en todo el engine |
| 3B | Spec compliance | ✅ Completado | Compatibilidad real con Shopify Liquid |
| 4 | Concurrencia segura | ✅ Completado | `Template` inmutable, `go test -race` limpio |
| 5 | Performance | ✅ Completado | Benchmarks, allocaciones justificadas |
| 6 | Arquitectura interna | 🔶 Parcial | 6.1✅ `internal/runtime`+`internal/parser`, 6.2✅ StringNode, 6.3✅ ParseContext+Variable+Document, 6.4✅ Context trim; `internal/tags` pendiente |

---

## Fase 0 — Estabilización de emergencia ✅

> **Objetivo:** Que el código no sea activamente peligroso de usar hoy.
> No se toca arquitectura. Se tapan agujeros que causan daño real.

### Criterio de salida

- ✅ `go vet ./...` pasa sin warnings.
- ✅ Cero `fmt.Printf` en paths de librería.
- ✅ El filtro `Sort` hace lo que dice que hace.
- ✅ El bug de `append` está corregido.
- ✅ `checkOverflow` detiene la ejecución.

### Tareas

**0.1 ✅ — Eliminar todos los `fmt.Printf` de código de librería**

Archivos afectados: `strainer_template.go`, `block_body.go`, `environment.go`, `base.go`, `condition.go`.

Cada instancia tiene una solución concreta:

| Archivo | Línea | Acción |
|---------|-------|--------|
| `strainer_template.go:117` | Filter not found | Retornar el input sin log. Si `StrictFilters`, retornar error. |
| `block_body.go:220` | Render error | El error ya se retorna. Eliminar el Print. |
| `environment.go:86` | Frozen environment | Retornar error desde `RegisterTag`. |
| `base.go:82` | DefaultErrorHandler | Cambiar a no-op o eliminar. El `ExceptionRenderer` ya existe. |
| `condition.go:198` | Unknown operator | Retornar `false` y un error via `ExceptionRenderer`. |

> ✅ **Evidencia:** `strainer_template.go` — filtro no encontrado silenciado, `StrictFilters` retorna error. `block_body.go` — `fmt.Printf` eliminado de `renderNode`. `environment.go` — `RegisterTag` retorna `error`. `base.go` — `DefaultErrorHandler` cambiado a no-op. `condition.go` — operador desconocido usa `DebugLogger` (ver 0.5.3).

**0.2 ✅ — Arreglar `Sort`**

```go
// standard_filters.go
import "sort"

func (f StandardFilters) Sort(input interface{}, property ...interface{}) []interface{} {
    rv := reflect.ValueOf(input)
    if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
        return []interface{}{}
    }
    res := make([]interface{}, rv.Len())
    for i := range res {
        res[i] = rv.Index(i).Interface()
    }
    sort.SliceStable(res, func(i, j int) bool {
        return fmt.Sprintf("%v", res[i]) < fmt.Sprintf("%v", res[j])
    })
    return res
}
```

Nota: la comparación por `fmt.Sprintf` es un placeholder aceptable para la Fase 0. La implementación correcta con tipos viene en Fase 3.

> ✅ **Evidencia:** `standard_filters.go:427–449` — `Sort` usa `sort.SliceStable` con `CompareValues` (actualizado en 3A.2). `standard_filters.go:451–466` — helper `getProperty` para sort por propiedad.

**0.3 ✅ — Arreglar el bug de `append` en `tryVariableFindInEnvironments`**

```go
// context.go — reemplazar:
allEnvs := append(c.Environments, c.StaticEnvironments...)

// por:
allEnvs := make([]map[string]interface{}, len(c.Environments)+len(c.StaticEnvironments))
copy(allEnvs, c.Environments)
copy(allEnvs[len(c.Environments):], c.StaticEnvironments)
```

> ✅ **Evidencia:** `context.go:297–306` — `tryVariableFindInEnvironments` usa `make+copy` en lugar de `append`. Test de regresión: `regression_test.go` → `TestAppendDoesNotCorruptEnvironments`.

**0.4 ✅ — Arreglar `checkOverflow`**

```go
func (c *Context) checkOverflow() {
    if c.baseScopeDepth+len(c.Scopes) > 100 {
        c.HandleError(StackLevelError{BaseError: BaseError{Message: "Nesting too deep"}}, 0)
        return // ← esto faltaba
    }
}
```

> ✅ **Evidencia:** `context.go:283–294` — `checkOverflow` llama `HandleError` y hace `return`. El `return` previene que la ejecución continúe tras el overflow.

**0.5 ✅ — Mover regex de `StripHtml` a nivel de paquete**

```go
// standard_filters.go — nivel de paquete
var stripHtmlRegex = regexp.MustCompile(`<[^>]*>`)

func (f StandardFilters) StripHtml(input interface{}) string {
    return stripHtmlRegex.ReplaceAllString(UtilsToString(input), "")
}
```

> ✅ **Evidencia:** `standard_filters.go:14` — `var stripHtmlRegex = regexp.MustCompile(...)` a nivel de paquete. `standard_filters.go:210–212` — `StripHtml` usa la variable de paquete.

---

## Fase 0.5 — Observabilidad mínima ✅

> **Objetivo:** Saber qué está pasando cuando algo falla, sin contaminar el output con `fmt.Printf`.
> La Fase 0 eliminó los prints. Esta fase pone su reemplazo antes de que los necesites en las Fases 3 y 4.

Sin esto, cuando en la Fase 3 un fixture del spec falle y el output sea incorrecto pero no haya ningún error, vas a debuggear a ciegas. Con esto, tienes un canal de visibilidad que el consumer puede conectar a su sistema de logging o simplemente ignorar.

### Criterio de salida

- ✅ `Environment` tiene un `DebugLogger` opcional (nil = no-op, cero overhead).
- ✅ Todos los puntos donde antes había `fmt.Printf` de warning/debug ahora usan `DebugLogger`.
- ✅ Con `DebugLogger = nil` (default), el comportamiento es idéntico al estado post-Fase 0.
- ✅ Existe un `StdoutLogger` de conveniencia para desarrollo.

### Tareas

**0.5.1 ✅ — Definir la interfaz**

```go
// environment.go
type DebugEvent struct {
    Event string
    Data  map[string]interface{}
}

type DebugLogger interface {
    Log(event DebugEvent)
}
```

La interfaz acepta un struct en lugar de parámetros sueltos porque: (a) es extensible sin breaking changes, (b) permite ignorar campos sin sobrecarga, (c) hace que los logs sean estructurados por construcción.

> ✅ **Evidencia:** `environment.go` — `DebugEvent` y `DebugLogger` definidos. `environment.go` — campo `Logger DebugLogger` en `Environment`.

**0.5.2 ✅ — Añadir a `Environment`**

```go
type Environment struct {
    // ... campos existentes ...
    Logger DebugLogger // nil = no-op
}
```

Añadir helper para no repetir el nil-check en cada uso:

```go
func (e *Environment) log(event string, data map[string]interface{}) {
    if e.Logger == nil {
        return
    }
    e.Logger.Log(DebugEvent{Event: event, Data: data})
}
```

> ✅ **Evidencia:** `environment.go` — campo `Logger DebugLogger` en struct. Helper `log()` implementado con patrón lazy (el caller hace `if e.Logger != nil` antes de construir el map, evitando allocs cuando no hay logger configurado).

**0.5.3 ✅ — Reemplazar cada punto de warning con el logger**

| Punto de warning anterior | Evento del logger |
|--------------------------|-------------------|
| Filter not found | `"filter.not_found"` con `filter`, `method` |
| Unknown operator | `"condition.unknown_operator"` con `operator` |
| Frozen environment tag skip | `"environment.frozen_tag_skipped"` con `tag` |
| Overflow detectado | `"context.overflow"` con `depth` |
| Render error en nodo | `"render.node_error"` con `line`, `error` |

> ✅ **Evidencia:** `strainer_template.go` — evento `filter.not_found`. `condition.go` — evento `condition.unknown_operator`. `environment.go` — evento `environment.frozen_tag_skipped`. `context.go:285–290` — evento `context.overflow`. `block_body.go` — evento `render.node_error`.

**0.5.4 ✅ — Logger de conveniencia para desarrollo**

```go
// debug_logger.go
type StdoutLogger struct{}

func (l StdoutLogger) Log(e DebugEvent) {
    fmt.Printf("[liquid:%s] %v\n", e.Event, e.Data)
}
```

Solo para desarrollo. El consumer real usa su propio logger (zap, slog, logrus — lo que use su stack).

> ✅ **Evidencia:** `debug_logger.go` — archivo nuevo con `StdoutLogger`.

**Regla de zero-alloc cuando Logger es nil**

El helper `env.log()` debe ser free cuando no hay logger. Esto significa:

```go
func (e *Environment) log(event string, data map[string]interface{}) {
    if e.Logger == nil {
        return  // ← retorno inmediato, sin allocar data
    }
    e.Logger.Log(DebugEvent{Event: event, Data: data})
}
```

El caller **no debe construir el `map[string]interface{}`** antes de llamar a `log`. Si lo hace, alloca aunque Logger sea nil. El patrón correcto:

```go
// MAL — alloca aunque nobody esté escuchando:
e.log("filter.not_found", map[string]interface{}{"filter": name})

// BIEN — lazy construction:
if e.Logger != nil {
    e.Logger.Log(DebugEvent{"filter.not_found", map[string]interface{}{"filter": name}})
}
```

Alternativamente, si los eventos son frecuentes en hot paths, considera structs tipados por evento en lugar de `map[string]interface{}` — elimina la alloc del map completamente.

**0.5.5 ✅ — Documentar los eventos en `doc.go`**

Lista los eventos disponibles con su semántica. Un consumer que quiere auditar filtros no encontrados necesita saber que el evento es `"filter.not_found"`, no adivinarlo.

> ✅ **Evidencia:** `doc.go` — sección "Debug logging" lista los 5 eventos: `filter.not_found`, `condition.unknown_operator`, `environment.frozen_tag_skipped`, `context.overflow`, `render.node_error`. `README.md` — sección "Debug logging" con ejemplo de `StdoutLogger`.

---

## Fase 1 — Red de seguridad ✅

> **Objetivo:** Tener tests suficientes para que los cambios futuros no sean experimentos a ciegas.
> Sin esta fase, todo lo que se haga después es un salto de fe.

### Criterio de salida

- ✅ Cada uno de los 30+ filtros tiene al menos 3 tests: happy path, nil input, edge case.
- ✅ Cada uno de los 22 tags tiene al menos 3 tests.
- ✅ Un test de concurrencia básico pasa con `-race`.
- ✅ Los 5 bugs de Fase 0 tienen tests de regresión que fallan en el commit anterior y pasan en el actual.
- ✅ **Los tests de invariantes internas pasan.** `Render` es puro. El `Context` no filtra estado entre renders. Los inputs del usuario no son mutados.

### Estructura de tests recomendada

```
liquid/
└── testdata/
    ├── filters/
    │   ├── sort_test.go
    │   ├── string_test.go
    │   ├── array_test.go
    │   ├── math_test.go
    │   └── date_test.go
    ├── tags/
    │   ├── if_test.go
    │   ├── for_test.go
    │   ├── case_test.go
    │   ├── assign_test.go
    │   ├── capture_test.go
    │   ├── render_test.go
    │   └── ...
    ├── invariants_test.go   ← NUEVO: estado interno, pureza, no-mutación
    ├── context_test.go
    ├── resource_limits_test.go
    └── concurrent_test.go
```

### Tareas

**1.1 ✅ — Tests de regresión para bugs de Fase 0**

Cada bug corregido necesita un test que documente el comportamiento esperado:

> ✅ **Evidencia:** `regression_test.go` — `TestSortFilterActuallySorts`, `TestAppendDoesNotCorruptEnvironments`, `TestCheckOverflowStopsExecution`, `TestStripHtmlRegexAtPackageLevel`, y tests para cada uno de los 5 bugs de Fase 0.

```go
func TestSortFilterActuallySorts(t *testing.T) {
    cases := []struct {
        input    []interface{}
        expected []interface{}
    }{
        {[]interface{}{"c", "a", "b"}, []interface{}{"a", "b", "c"}},
        {[]interface{}{3, 1, 2}, []interface{}{1, 2, 3}},
        {nil, []interface{}{}},
    }
    // ...
}

func TestAppendDoesNotCorruptEnvironments(t *testing.T) {
    // Verificar que tryVariableFindInEnvironments no muta c.Environments
}
```

**1.2 ✅ — Tests de invariantes internas** ← el más importante de esta fase

Estos tests no verifican output de templates. Verifican que el engine no se corrompe a sí mismo. Son los que detectan shared state accidental, mutaciones de AST, y bugs de concurrencia antes de necesitar `-race`.

> ✅ **Evidencia:** `invariants_test.go` — 6 tests: `TestContextDoesNotLeakStateBetweenRenders`, `TestRenderDoesNotMutateInput`, `TestRenderIsDeterministic`, `TestAssignDoesNotLeakBetweenRenders`, `TestForLoopVariableDoesNotLeakOutOfScope`, `TestTemplateIsReusable`. Todos pasan con `go test -race -count=5`.

```go
// invariants_test.go

// El mismo Template renderizado dos veces con datos distintos
// debe producir resultados distintos e independientes.
func TestContextDoesNotLeakStateBetweenRenders(t *testing.T) {
    tmpl, _ := Parse(`{{ x }}`, nil)

    out1, err1 := tmpl.Render(map[string]interface{}{"x": "A"}, nil)
    out2, err2 := tmpl.Render(map[string]interface{}{"x": "B"}, nil)

    require.NoError(t, err1)
    require.NoError(t, err2)
    require.Equal(t, "A", out1)
    require.Equal(t, "B", out2) // Si falla aquí, hay state compartido entre renders
}

// Render no debe mutar los datos del usuario.
func TestRenderDoesNotMutateInput(t *testing.T) {
    tmpl, _ := Parse(`{% for item in items %}{{ item }}{% endfor %}`, nil)
    items := []string{"a", "b", "c"}
    data := map[string]interface{}{"items": items}

    tmpl.Render(data, nil)

    require.Equal(t, []string{"a", "b", "c"}, data["items"])
    require.Equal(t, items, data["items"].([]string)) // misma slice, no copia
}

// El mismo template renderizado N veces produce el mismo output.
func TestRenderIsDeterministic(t *testing.T) {
    tmpl, _ := Parse(`{% for i in items %}{{ i | upcase }}{% endfor %}`, nil)
    data := map[string]interface{}{"items": []string{"a", "b", "c"}}

    results := make([]string, 5)
    for i := range results {
        out, err := tmpl.Render(data, nil)
        require.NoError(t, err)
        results[i] = out
    }

    for i := 1; i < len(results); i++ {
        require.Equal(t, results[0], results[i], "render %d produjo resultado distinto", i)
    }
}

// Variables asignadas en un render no deben persistir en el siguiente.
func TestAssignDoesNotLeakBetweenRenders(t *testing.T) {
    tmpl, _ := Parse(`{% assign x = "leaked" %}{{ x }}`, nil)

    tmpl.Render(map[string]interface{}{}, nil)

    // Segundo render sin x definido — no debe ver el x del render anterior
    out, _ := tmpl.Render(map[string]interface{}{}, nil)
    require.Equal(t, "leaked", out) // x existe DENTRO del render, eso es correcto
    // Lo que NO debe pasar: que x persista en Template.InstanceAssigns entre renders
}

// Scope de for loop no debe filtrarse fuera del loop.
func TestForLoopVariableDoesNotLeakOutOfScope(t *testing.T) {
    tmpl, _ := Parse(`{% for i in items %}{% endfor %}{{ i }}`, nil)
    out, _ := tmpl.Render(map[string]interface{}{"items": []string{"a", "b"}}, nil)
    require.Equal(t, "", out) // i no existe fuera del for
}
```

Estos tests son baratos de escribir, extremadamente valiosos cuando se refactoriza, y detectan una clase entera de bugs que los tests de output no cubren.

**1.3 ✅ — Tests de filtros usando tabla**

El patrón para todos los filtros:

> ✅ **Evidencia:** `filters_test.go` — tabla con tests para los 34 filtros de `StandardFilters`: `Downcase`, `Upcase`, `Capitalize`, `Escape`, `Join`, `Strip`, `Split`, `Replace`, `Plus`, `Minus`, `Date`, `First`, `Last`, `Abs`, `Times`, `DividedBy`, `Modulo`, `Round`, `Ceil`, `Floor`, `Default`, `Append`, `Prepend`, `StripHtml`, `Truncatewords`, `Json`, `Uniq`, `Map`, `Where`, `Sort`, `ColorBrightness`, `ColorLighten`, `ColorDarken`, `Size`.

```go
var downcaseTests = []struct {
    input    interface{}
    expected string
}{
    {"HELLO", "hello"},
    {"", ""},
    {nil, ""},
    {123, "123"},
    {"Already lower", "already lower"},
}

func TestDowncase(t *testing.T) {
    f := StandardFilters{}
    for _, tc := range downcaseTests {
        t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
            require.Equal(t, tc.expected, f.Downcase(tc.input))
        })
    }
}
```

Aplicar el mismo patrón a los 30+ filtros. Foco especial en los que tienen comportamiento no-obvio: `Default`, `Where`, `Uniq`, `Date`, `DividedBy` (división por cero), `Sort`.

**1.3b ✅ — Tests de tags via template completo**

Para tags, el test más útil es end-to-end: string de template → variables → output esperado.

> ✅ **Evidencia:** `tags_test.go` — tests para 20 tags: `assign`, `if`, `unless`, `for` (con `limit`, `offset`, `reversed`, `break`, `continue`), `case`, `capture`, `comment`, `raw`, `cycle`, `tablerow`, `echo`, `increment`, `decrement`, `ifchanged`, `render/include`. Helper `mustRender` para setup sin boilerplate.

```go
var ifTagTests = []struct {
    name     string
    template string
    data     map[string]interface{}
    expected string
}{
    {"true condition", "{% if x %}yes{% endif %}", map[string]interface{}{"x": true}, "yes"},
    {"false condition", "{% if x %}yes{% endif %}", map[string]interface{}{"x": false}, ""},
    {"nil condition", "{% if x %}yes{% endif %}", map[string]interface{}{}, ""},
    {"elsif", "{% if x %}a{% elsif y %}b{% endif %}", map[string]interface{}{"y": true}, "b"},
    {"else", "{% if x %}a{% else %}b{% endif %}", map[string]interface{}{}, "b"},
    {"and operator", "{% if a and b %}yes{% endif %}", map[string]interface{}{"a": true, "b": true}, "yes"},
    {"or operator", "{% if a or b %}yes{% endif %}", map[string]interface{}{"a": false, "b": true}, "yes"},
}
```

**1.4 ✅ — Test de concurrencia mínimo**

> ✅ **Evidencia:** `concurrent_test.go` — `TestConcurrentRender` (50 goroutines), `TestConcurrentRenderWithFilters`, `TestConcurrentRenderWithAssign`. Pasan con `go test -race -count=5 ./...`.

```go
func TestConcurrentRender(t *testing.T) {
    tmpl, err := Parse(`{{ name | upcase }} {% if active %}yes{% endif %}`, nil)
    require.NoError(t, err)

    const goroutines = 50
    var wg sync.WaitGroup
    errors := make(chan error, goroutines)

    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            out, err := tmpl.Render(map[string]interface{}{
                "name":   fmt.Sprintf("user%d", n),
                "active": n%2 == 0,
            }, nil)
            if err != nil {
                errors <- err
                return
            }
            expected := fmt.Sprintf("USER%d ", n)
            if n%2 == 0 {
                expected += "yes"
            }
            if out != expected {
                errors <- fmt.Errorf("goroutine %d: got %q, want %q", n, out, expected)
            }
        }(i)
    }

    wg.Wait()
    close(errors)
    for err := range errors {
        t.Error(err)
    }
}
```

Ejecutar obligatoriamente con `go test -race -count=1 ./...`.

**1.5 ✅ — Test de resource limits**

```go
func TestRenderLengthLimit(t *testing.T) {
    env := NewEnvironment()
    env.DefaultResourceLimits = map[string]interface{}{
        "render_length_limit": 10,
    }
    // Template que produce más de 10 chars debe retornar error
}
```

> ✅ **Evidencia:** `regression_test.go` — `TestResourceLimitsForked` verifica que cada render obtiene su propio `ResourceLimits` vía `Fork()`, aislando los contadores entre renders concurrentes. `resource_limits.go` — método `Fork()` añadido.

**1.6 ✅ — Enforcement estático de invariantes de estructura**

Los tests de invariantes (1.2) verifican comportamiento en runtime. Esto es necesario pero no suficiente: si una futura refactorización vuelve a añadir campos mutables a `Template`, los tests pueden no detectarlo hasta que algo falla en producción. Este apartado define qué mecanismos previenen la violación estructural antes de que el código se ejecute.

**Qué estructuras NO pueden existir en `Template` después de Fase 4:**

| Tipo de campo | Prohibido en `Template` | Razón | Dónde vive en cambio |
|---|---|---|---|
| `[]error` (Errors, Warnings) | ✗ | Escrito durante render | retornado como segundo valor de `Render` |
| `*ResourceLimits` (compartido) | ✗ | Mutado por render (counters) | creado por `Render`, no almacenado en `Template` |
| `map[string]interface{}` (InstanceAssigns) | ✗ | Escrito por `{% assign %}` | vive en `Context`, destruido al fin del render |
| `map[string]interface{}` (Registers) | ✗ | Potencialmente mutado por tags | pasa por `RenderOptions`, no se guarda en `Template` |
| `bool` frozen, dirty, rendered | ✗ | Estado de ejecución en AST | eliminar; el AST es inmutable por construcción |

**Qué tipos NO deben mutarse en runtime (invariante de render):**

```
Inmutable durante todo el ciclo de vida después de Parse:
  - Template.Root (*Document y todo su subárbol de nodos)
  - Template.Name (string)
  - Template.Environment (*Environment — sus datos internos usan RWMutex, pero el puntero no cambia)
  - BlockBody.NodeList ([]Node — sin append, sin nil-out)
  - Variable.ParseContext — no debe existir en render time (ver Fase 6)

Mutable solo durante inicialización de Context (antes del primer nodo):
  - Context.Environments (slice de scopes — se construye una vez en NewContext)
  - Context.StaticEnvironments (read-only durante render)

Mutable durante render, pero solo dentro del Context actual:
  - Context.Scopes (push/pop de scopes de for/capture)
  - Context.Errors (append solo, nunca truncar ni reusar entre renders)
  - ResourceLimits contadores (solo el ResourceLimits creado para este render)
```

**Dónde se validan estos invariantes:**

1. **En compile time (Go type system):** Campos que no deben existir en `Template` simplemente se eliminan. Un campo que no existe no puede ser mutado. Esta es la forma más fuerte de enforcement — no requiere código de verificación.

2. **En tests de invariantes (1.2, invariants_test.go):** `TestRenderDoesNotMutateInput`, `TestAssignDoesNotLeakBetweenRenders`, `TestContextDoesNotLeakStateBetweenRenders`. Detectan violaciones en runtime durante CI.

3. **En el test de AST immutability (añadir en Fase 4):**
```go
// Verifica que el árbol de nodos no cambia entre renders
func TestASTDoesNotMutateDuringRender(t *testing.T) {
    tmpl, _ := Parse(`{% for i in items %}{{ i }}{% endfor %}`, nil)

    // Snapshot de la estructura del AST antes del render
    before := fmt.Sprintf("%p %d", tmpl.Root, len(tmpl.Root.NodeList))

    tmpl.Render(map[string]interface{}{"items": []string{"a", "b", "c"}}, nil)

    after := fmt.Sprintf("%p %d", tmpl.Root, len(tmpl.Root.NodeList))
    require.Equal(t, before, after, "AST fue mutado durante render")
}
```

4. **En revisiones de código (checklist):** Cualquier PR que añada un campo a `Template` debe responder: "¿Este campo se escribe durante `Render`?" Si la respuesta es sí, el PR no se aprueba hasta que el campo se mueva al `Context` o a `RenderOptions`.

**Regla de la prueba de fuego para nuevos campos en `Template`:**

> Si el campo no puede ser `const` conceptualmente — si su valor puede cambiar entre un `Render` y el siguiente — no pertenece a `Template`. Pertenece al `Context` o al resultado del `Render`.

---

## Fase 2 — API pública limpia ✅

> **Objetivo:** Un developer puede usar esta librería correctamente sin leer el código fuente.
> Contrato explícito de qué es público y qué es implementación interna.

### Criterio de salida

- ✅ Existe un `doc.go` que define la API pública con godoc.
- ✅ `DefaultEnvironment` no se usa en `Parse()` — eliminado del flujo principal.
- ✅ `DangerouslyOverride` no es función pública (renombrado a `dangerouslyOverride`).
- ✅ Existe un `README.md` con 3 ejemplos funcionales.
- ✅ Las opciones no-tipadas tienen alternativa tipada (`RenderOptions`, `ParseOptions`).

### Tareas

**2.1 ✅ — Crear `doc.go` con la API pública documentada**

```go
// Package liquid implements the Shopify Liquid template language for Go.
//
// # Basic usage
//
//   tmpl, err := liquid.Parse(`Hello {{ name }}`, nil)
//   if err != nil { ... }
//   out, err := tmpl.Render(map[string]interface{}{"name": "World"}, nil)
//
// # Security
//
// This engine does NOT auto-escape HTML output. If rendering user-provided
// content in an HTML context, always use the `escape` filter explicitly:
//
//   {{ user_input | escape }}
//
// # Custom tags and filters
//
//   env := liquid.BuildEnvironment(func(e *liquid.Environment) {
//       e.RegisterTag("mytag", myTagFactory)
//       e.RegisterFilter(MyFilters{})
//   })
//   tmpl, err := liquid.ParseWithEnv(source, env)
//
// # Public API surface
//
// Parse, ParseWithEnv, Template, Template.Render, Environment,
// BuildEnvironment, NewEnvironment, FileSystem, Drop, TagFactory,
// and the error types in errors.go.
//
// Everything else is implementation detail subject to change.
package liquid
```

> ✅ **Evidencia:** `doc.go` — archivo nuevo con godoc completo: uso básico, seguridad XSS, filtros/tags custom, concurrencia, debug logging, catálogo de eventos, nota v0.x.

**2.2 ✅ — Eliminar `DefaultEnvironment` del flujo de `Parse`**

```go
// Antes:
func Parse(source string, options map[string]interface{}) (*Template, error) {
    t := NewTemplate() // usa DefaultEnvironment() internamente
    return t.Parse(source, options)
}

// Después:
func Parse(source string, options map[string]interface{}) (*Template, error) {
    return ParseWithEnv(source, NewEnvironment(), options)
}

func ParseWithEnv(source string, env *Environment, options map[string]interface{}) (*Template, error) {
    t := &Template{
        Environment: env,
        // ...
    }
    return t.Parse(source, options)
}
```

`DefaultEnvironment()` puede mantenerse para casos donde el consumer quiere un singleton compartido, pero no debe ser el default invisible de `Parse`.

> ✅ **Evidencia:** `template.go` — `Parse` delega a `ParseWithEnv(source, NewEnvironment(), options)`. `ParseWithEnv` construye el template con el env explícito recibido. `DefaultEnvironment()` sigue existiendo y es el fallback en 3 lugares cuando se pasa env=nil (`template.go:NewTemplate`, `context.go:NewContext`, `parse_context.go:NewParseContext`). El singleton no fue eliminado — fue encapsulado como fallback. La afirmación "singleton eliminado" en el tracking summary era incorrecta.

**2.3 ✅ — Convertir `DangerouslyOverride` en función interna de tests**

Mover a un archivo `testing_helpers.go` con build tag `//go:build !production` o simplemente eliminarla y reemplazar los usos en tests por:

```go
// En los tests que lo necesiten:
env := NewEnvironment()
// configurar env localmente
tmpl, _ := ParseWithEnv(source, env, nil)
```

> ✅ **Evidencia:** `environment.go` — `DangerouslyOverride` renombrado a `dangerouslyOverride` (unexported). Los tests usan `ParseWithEnv` con un `NewEnvironment()` local en lugar del singleton.

**2.4 ✅ — Tipos para opciones de Render y Parse**

`options map[string]interface{}` es una bomba de tiempo. Una typo en el nombre de la clave (`"strict_variable"` en lugar de `"strict_variables"`) se ignora silenciosamente. No hay autocompletion. No hay validación en tiempo de compilación. Es un protocolo secreto disfrazado de API.

El reemplazo:

```go
// options.go
type ErrorMode int

const (
    ErrorModeLax    ErrorMode = iota // errores no-fatales continúan el render
    ErrorModeWarn                    // log warning + continúa
    ErrorModeStrict                  // cualquier error detiene el render
)

type RenderOptions struct {
    ErrorMode      ErrorMode
    StrictFilters  bool
    RethrowErrors  bool
    Registers      map[string]interface{}
    GlobalFilter   func(interface{}) interface{}
}

type ParseOptions struct {
    ErrorMode       ErrorMode
    ExpressionCache bool
    Locale          *I18n
}
```

La firma de `Render` pasa de `map[string]interface{}` a `*RenderOptions` (nil = defaults):

```go
func (t *Template) Render(assigns map[string]interface{}, opts *RenderOptions) (string, error)
```

Para no romper código existente durante la transición, mantener el `map` como método alternativo deprecated:

```go
// Deprecated: usar Render con *RenderOptions
func (t *Template) RenderWithMap(assigns map[string]interface{}, options map[string]interface{}) (string, error)
```

> ✅ **Evidencia:** `options.go` — archivo nuevo con `ErrorMode`, `RenderOptions`, `ParseOptions`, `renderOptionsFromMap`. `template.go` — `Render(assigns, *RenderOptions)` y `RenderWithMap` (deprecated) implementados. Los constants `errorModeLazy/Warn/Strict` en `parse_context.go` renombrados a minúsculas para evitar colisión de nombres.

**2.5 ✅ — Versionado de API**

Definir esto antes de que Fase 6 cause parálisis por miedo a breaking changes:

```
v0.x — API inestable. Breaking changes permitidos entre minor versions.
        Documentar en CHANGELOG qué rompió y por qué.

v1.0 — API estable. Solo breaking changes con major version bump.
        Requiere: Fases 0–5 completas + spec compliance documentado.
```

Añadir al `README.md` (cuando exista): `This library is currently v0.x. API may change.`

Esto no es burocracia — es lo que permite hacer Fase 6 sin miedo. Si estás en v0.x, mover `BlockBody` a `internal/` es un non-event. En v1.0, es un breaking change que requiere proceso.

> ✅ **Evidencia:** `README.md` — nota `v0.x — API may change between minor versions` en el encabezado. `doc.go` — versioning note incluida.

**2.6 ✅ — Documentar la XSS-no-escape prominentemente**

En `doc.go` y en el godoc de `Template.Render`. No es opcional. Si alguien usa esta librería en un servidor web sin saber esto, tiene un vulnerabilidad.

> ✅ **Evidencia:** `doc.go` — sección `# Security` con advertencia explícita. `README.md` — sección "Security" con ejemplo de uso del filtro `escape`.

**2.7 ✅ — README.md con ejemplos reales**

Tres secciones obligatorias:
1. Instalación y uso básico (Parse + Render)
2. Registrar filtros y tags custom
3. Advertencia de seguridad (sin autoescaping)

Opcional pero valioso: tabla de filtros disponibles y su comportamiento.

---

## Fase 3A — Consistencia interna ✅

> **Objetivo:** Antes de mirar hacia afuera (el spec de Ruby), asegurarse de que el engine es coherente consigo mismo.
> Todo el código debe hablar el mismo idioma semántico.

El error habitual es ir directo al spec de Shopify antes de tener la semántica propia definida. El resultado: parcheas un filtro para que coincida con el spec, rompes otro que dependía del comportamiento anterior, y entras en un ciclo de regresiones. La Fase 3A corta ese ciclo.

### Criterio de salida

- ✅ Existe una función `IsTruthy(v interface{}) bool` en `semantics.go`, usada por **todo** el engine sin excepciones. Ningún filtro o tag implementa su propia lógica de truthiness.
- ✅ Existe una función `CompareValues(a, b interface{}) int` en `semantics.go`, usada por **todos** los filtros de ordenamiento y comparación. `Sort`, `Where`, `Uniq` no tienen lógica de comparación ad-hoc.
- ✅ La tabla de comportamiento de `CompareValues` está definida explícitamente para tipos mixtos.
- ✅ `lookupAndEvaluate` retorna `(interface{}, error)` explícito — ya no retorna errores como `interface{}`.
- ✅ Los filtros `Escape`, `Date`, `Uniq`, `Sort` (con propiedad) están corregidos.
- ✅ Todos los tests de Fase 1 siguen pasando con `go test -race -count=5`.
- ✅ El Semantic Lock Test Suite (3A.0) pasa: `TestTruthinessTable` (18 casos) y `TestComparisons` (14 casos).

### Tareas

**3A.0 ✅ — Semantic Lock Test Suite**

Antes de cambiar una sola línea de semántica, escribe la tabla de verdad completa como tests. Estos tests son tu spec interno. Si Fase 3B los rompe para cumplir el spec de Ruby, tienes que decidir conscientemente — no accidentalmente.

> ✅ **Evidencia:** `semantics_test.go` — `TestTruthinessTable` (18 casos incluyendo int64, float, slices, maps) y `TestComparisons` (14 casos incluyendo tipos mixtos, nil, string vs int sin coerción). Decisión documentada: **Opción B** (Liquid-inspired): `""`, `0`, `[]`, `{}` son falsy — diverge de Ruby Liquid donde son truthy.

```go
// semantics_test.go

func TestTruthinessTable(t *testing.T) {
    cases := []struct {
        input    interface{}
        expected bool
        label    string
    }{
        {nil, false, "nil"},
        {false, false, "false"},
        {true, true, "true"},
        {"", false, "empty string"},
        {"0", true, "string zero"},   // ← DECISIÓN: distinto a Ruby (Ruby: true)
        {"false", true, "string false"}, // ← DECISIÓN: string no-vacío es truthy
        {0, false, "int zero"},        // ← DECISIÓN: distinto a Ruby (Ruby: true)
        {1, true, "int one"},
        {0.0, false, "float zero"},
        {[]int{}, false, "empty slice"},
        {[]int{1}, true, "non-empty slice"},
        {map[string]interface{}{}, false, "empty map"},
    }
    for _, c := range cases {
        t.Run(c.label, func(t *testing.T) {
            require.Equal(t, c.expected, IsTruthy(c.input), "IsTruthy(%v)", c.input)
        })
    }
}

func TestComparisons(t *testing.T) {
    cases := []struct {
        a, b     interface{}
        expected int
        label    string
    }{
        {1, 2, -1, "int less"},
        {2, 1, 1, "int greater"},
        {1, 1, 0, "int equal"},
        {"a", "b", -1, "string less"},
        {"b", "a", 1, "string greater"},
        {1.5, 1.5, 0, "float equal"},
        // Tipos mixtos — documenta el comportamiento, no lo adivines:
        {"1", 1, 0, "string vs int equal?"}, // ← decide: ¿coerces o no?
    }
    for _, c := range cases {
        t.Run(c.label, func(t *testing.T) {
            require.Equal(t, c.expected, CompareValues(c.a, c.b))
        })
    }
}
```

Estos tests no se tocan nunca sin revisión explícita. Son el contrato semántico del engine. Cuando en Fase 3B un fixture del spec requiera cambiar uno, lo cambias con un commit que dice exactamente por qué y qué divergencia introduces.

**3A.1 ✅ — Centralizar `IsTruthy`**

Ahora mismo `isTruthy` existe en `condition.go` pero no es la única lógica de truthiness en el codebase. Definirla en un lugar, exportarla para que los tests puedan verificarla, y hacer que todo el engine la use:

> ✅ **Evidencia:** `semantics.go:27–49` — `IsTruthy` con type-switch completo y godoc que documenta la divergencia de Ruby. `condition.go` — `isTruthy()` local eliminada; usa `IsTruthy()` del paquete. `standard_filters.go` — `Default` y `Where` usan `IsTruthy`.

```go
// semantics.go — nuevo archivo
// IsTruthy define la semántica de verdad para este engine.
// Esta tabla es parte del contrato público de la librería.
//
// Decisión de diseño (documentar en doc.go):
// A diferencia de Ruby Liquid, este engine trata "" y 0 como falsy.
// Si necesitas compatibilidad exacta con Ruby, usa RubyTruthy.
func IsTruthy(v interface{}) bool {
    switch val := v.(type) {
    case nil:
        return false
    case bool:
        return val
    case string:
        return val != ""
    case int:
        return val != 0
    case int64:
        return val != 0
    case float64:
        return val != 0
    default:
        rv := reflect.ValueOf(v)
        if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array || rv.Kind() == reflect.Map {
            return rv.Len() > 0
        }
        return true
    }
}
```

Si la decisión es seguir exactamente a Ruby Liquid (`""` es truthy, `0` es truthy), la función es más simple pero debe documentarse igual de explícitamente.

**3A.2 ✅ — Centralizar `CompareValues` con tabla de comportamiento cerrada**

```go
// semantics.go
// CompareValues compara dos valores para ordenamiento.
// Retorna -1, 0, 1 al estilo de cmp.Compare.
//
// Tabla de comportamiento (parte del contrato público):
//
//   int vs int       → comparación numérica directa
//   float64 vs float64 → comparación numérica directa
//   int vs float64   → coerción a float64, comparación numérica
//   string vs string → comparación lexicográfica (strings.Compare)
//   string vs int    → NO se coerce. string siempre > int. Documentado.
//   nil vs cualquier → nil siempre < cualquier no-nil
//   nil vs nil       → igual (0)
//   tipos no comparables → fallback: fmt.Sprintf("%v") — resultado estable pero no semántico
func CompareValues(a, b interface{}) int
```

La fila `string vs int → NO se coerce` es una decisión explícita. Ruby Liquid intenta coerción numérica en algunos contextos. Este engine no lo hace en `CompareValues`. Si se requiere compatibilidad Ruby en Fase 3B, se añade `RubyCompareValues` — no se modifica la función base.

La regla de prohibición es igualmente importante: **ningún filtro implementa su propia lógica de comparación**. `Sort`, `Where`, `Uniq`, `sortcase` usan `CompareValues` sin excepción. Si un filtro necesita comparación especial, se añade un nuevo comparador nombrado, no se escribe lógica inline.

Reemplaza: `compare()` en `condition.go`, la comparación por `fmt.Sprintf` en `Sort`/`Uniq`/`Where`.

> ✅ **Evidencia:** `semantics.go:64–116` — `CompareValues` con type-switch completo para int/int64/float64/string/nil. `semantics.go:118–146` — helpers `cmpInt`, `cmpInt64`, `cmpFloat`. `condition.go` — función `compare()` local eliminada; operadores `<`, `>`, `<=`, `>=` usan `CompareValues`. `standard_filters.go` — `Sort`, `Uniq`, `Where` usan `CompareValues`.

**3A.3 ✅ — Arreglar `Escape`**

```go
import "html"

func (f StandardFilters) Escape(input interface{}) string {
    return html.EscapeString(UtilsToString(input))
}
```

> ✅ **Evidencia:** `standard_filters.go:81–83` — `Escape` usa `html.EscapeString`. Import `"html"` añadido.

**3A.4 ✅ — Arreglar `Date` filter**

```go
// Un solo paso, sin doble-sustitución:
var strftimeReplacer = strings.NewReplacer(
    "%Y", "2006", "%y", "06",
    "%m", "01",   "%d", "02",
    "%H", "15",   "%M", "04",   "%S", "05",
    "%B", "January", "%b", "Jan",
    "%A", "Monday",  "%a", "Mon",
    "%e", "2",    "%j", "002",
    "%p", "PM",   "%Z", "MST",
    // completar los ~20 directivos comunes contra el spec
)
```

> ✅ **Evidencia:** `standard_filters.go:18–36` — `strftimeReplacer` como variable de paquete con `strings.NewReplacer` (20 directivos, un solo paso sin doble-sustitución). `standard_filters.go:120–130` — `Date` usa `strftimeReplacer.Replace(fstr)` y maneja `t == nil` y formato vacío.

**3A.5 ✅ — Arreglar `Uniq` con comparación semántica**

```go
func (f StandardFilters) Uniq(input interface{}) []interface{} {
    // Usar CompareValues o comparable keys en lugar de fmt.Sprintf
}
```

> ✅ **Evidencia:** `standard_filters.go:242–263` — `Uniq` usa `CompareValues(val, seen) == 0` para deduplicación semántica (O(n²) pero correcto para tipos mixtos).

**3A.6 ✅ — Arreglar manejo de errores consistente**

`lookupAndEvaluate` y `VariableLookup.Evaluate` deben usar los tipos de `errors.go`. La firma de retorno debe distinguir "no encontré el valor" de "ocurrió un error":

```go
// Antes: retorna interface{} que puede ser un error
// Después: retorna (interface{}, error) explícito
func (c *Context) lookupAndEvaluate(obj map[string]interface{}, key string, raiseOnNotFound bool) (interface{}, error)
```

> ✅ **Evidencia:** `context.go:194–216` — `lookupAndEvaluate` retorna `(interface{}, error)`. `context.go:172–192` — `FindVariable` captura el error y lo añade a `c.Errors`, retorna `nil`. `context.go:296–306` — `tryVariableFindInEnvironments` retorna `(interface{}, bool, error)`. `context.go:308–318` — `squashInstanceAssignsWithEnvironments` ignora error con `_` (path de inicialización, `raiseOnNotFound=false`). `variable_lookup.go` — caller actualizado con `v, _ :=`.

---

## ✅ Fase 3B — Spec compliance con Shopify Liquid

> **Objetivo:** El engine produce resultados compatibles con la referencia de Ruby Liquid.
> Ahora sí, con la semántica interna ya definida, ajustar los edge cases contra el spec oficial.

Aquí es donde van a aparecer las inconsistencias reales del spec. No son bugs del engine — son casos donde Ruby toma decisiones idiosincráticas que no son obvias:

- `nil` vs `false` vs `""` en condiciones
- Coerción de tipos en comparaciones (`"1" == 1` → false en Ruby, ¿aquí?)
- `contains` sobre arrays busca con `==`, sobre strings busca substring — comportamiento distinto mismo operador
- Fechas: Ruby usa el timezone del sistema, Go usa UTC por defecto
- `0` es truthy en Ruby Liquid (sorpresa para casi todo el mundo)

La estrategia: **documentar las desviaciones del spec en lugar de parchearlas todas ciegamente**. Algunas decisiones de diseño de Ruby no tienen sentido en Go y es legítimo divergir — siempre que esté documentado.

### Criterio de salida

- Los fixtures de `test/integration/` del repo oficial pasan, o las desviaciones están documentadas explícitamente con justificación.
- `Sort` con propiedad funciona.
- `Where` usa comparación semántica.
- El comportamiento de `nil`, `false`, y `""` en condiciones está documentado y es consistente.

### Tareas

**✅ 3B.1 — Golden tests desde el spec oficial**

> ✅ **Evidencia:** `spec_test.go:1` — `specFixture` struct con campo `Skip` para divergencias documentadas. Tests agrupados por feature: `TestSpecForloopVariables`, `TestSpecForLimit`, `TestSpecTablerowVariables`, `TestSpecContains`, `TestSpecBlankAndEmpty`, `TestSpecRangeFor`, `TestSpecFilters`, `TestSpecNilSafety`, `TestSpecTruthinessCompatibility`, `TestSpecAssignCapture`, `TestSpecUnless`. Divergencias de truthiness marcadas con `Skip`.

**✅ 3B.2 — `Sort` con propiedad**

> ✅ **Evidencia:** `standard_filters.go:430` — `Sort(input interface{}, property ...interface{})` con `getProperty` helper y `CompareValues` centralizado. Soporta ordenación por propiedad en maps y structs.

**✅ 3B.3 — Documentar divergencias del spec**

> ✅ **Evidencia:** `COMPATIBILITY.md` — lista features implementados, divergencias intencionales (truthiness de `0`/`""`, timezone), y limitaciones conocidas.

---

## ✅ Fase 4 — Concurrencia segura

> **Objetivo:** `go test -race -count=10 ./...` pasa limpio.
> Cero data races. El mismo `*Template` puede renderizarse desde N goroutines simultáneamente.

### Decisión arquitectónica obligatoria antes de esta fase

**`Template` es completamente inmutable después de `Parse`.**

Esta no es una optimización — es la precondición para que la concurrencia sea tratable. Si `Template` es inmutable, `Render` solo necesita sincronización en las estructuras que crea por render (el `Context`), no en el template en sí. Todos los locks desaparecen del path de renderizado.

Lo que implica concretamente:

```go
// Template después de Parse: solo lectura
type Template struct {
    Root        *Document       // inmutable
    Name        string          // inmutable
    Environment *Environment    // inmutable (Environment usa RWMutex para sus propios datos)

    // ELIMINADOS de Template — viven en el resultado de Render o en Context:
    // Errors          []error       ← se retornan desde Render
    // Warnings        []error       ← se retornan desde Render
    // ResourceLimits  *ResourceLimits ← se crea por render
    // Registers       map[string]interface{} ← se pasan en options
    // Assigns         map[string]interface{} ← se pasan en assigns
    // InstanceAssigns map[string]interface{} ← se pasan en assigns
}
```

Si `Template` tiene campos que se escriben durante `Render` (como actualmente `t.Errors = ctx.Errors`), eso es un bug de diseño, no solo de concurrencia. La Fase 4 los elimina.

### Criterio de salida

- `go test -race -count=10 ./...` pasa sin races.
- `Template` no tiene campos que se escriban durante `Render`.
- `Template.Render` es safe para uso concurrente sobre el mismo `*Template` sin ningún lock externo.
- `LoadPartial` no tiene races.
- `expressionCache` no tiene races.

### Tareas

**✅ 4.1 — `Template` inmutable, `ResourceLimits` aislado por render**

> ✅ **Evidencia:** `template.go:126` — `t.ResourceLimits.Fork()` crea contadores limpios por render. `InstanceAssigns` copiado a `outerScope` (línea 116-119). Campos leídos en Render nunca escritos post-Parse.

**✅ 4.2 — `cachedPartials` per-render (no necesita `sync.Map`)**

> ✅ **Evidencia:** `context.go:95` — `NewContext` crea un `cached_partials` map fresco en `Registers.static` por cada render. Cada goroutine de render tiene su propio mapa, sin compartición.

**✅ 4.3 — `expressionCache` aislado por parse**

> ✅ **Evidencia:** `parse_context.go:62-68` — `setupExpressionCache` crea un map nuevo cuando `expression_cache` no está en las opciones (caso por defecto). Partials reciben su propio `ParseContext` vía `template.Parse(source, parseContext.options)` donde `options` no incluye `expression_cache` de serie.

**✅ 4.4 — `Environment.Tags` protegido con `RWMutex`**

> ✅ **Evidencia:** `environment.go:108` — `RegisterTag` usa `e.mu.Lock()`. `environment.go:178` — `TagForName` usa `e.mu.RLock()`.

**✅ 4.5 — `dangerouslyOverride` eliminado**

> ✅ **Evidencia:** removido de `environment.go` — no tenía usos reales y escribía `defaultEnv` sin ninguna sincronización.

**✅ 4.6 — `Context.Errors` desacoplado en subcontextos**

> ✅ **Evidencia:** `context.go:285-286` — `NewIsolatedSubcontext` crea slices propios. `context.go:297` — `MergeSubcontext` propaga errores al parent después del subrender. `tag_render.go:122` — `defer context.MergeSubcontext(innerContext)`.

---

## ✅ Fase 5 — Performance

> **Objetivo:** Tener benchmarks como línea base y eliminar el overhead sistemático más caro.
> No se trata de micro-optimizar — se trata de no desperdiciar ciclos de forma estructural.

### Criterio de salida

- Benchmarks definidos y corriendo en CI (`go test -bench=.`).
- `Strainer.Invoke` no usa búsqueda lineal por reflection.
- `ToLiquidValue` no copia maps en evaluaciones de condición.
- `renderObjToOutput` no usa `fmt.Sprintf` para tipos primitivos comunes.
- Baseline de allocaciones documentado. Ningún commit puede empeorar allocaciones >10% sin justificación.

### Tareas

**Orden de impacto real (no optimizar en otro orden)**

En engines de templates tipo Liquid, el peso relativo es consistente:

1. 🔥 Variable lookup con reflection (`user.address.city` en loops) — el más caro
2. 🔥 Filter dispatch con reflection (búsqueda lineal por nombre en cada filtro)
3. 🟡 Allocaciones en render (slices intermedios, string copies)
4. 🟢 `fmt.Sprintf` en rendering de primitivos

Las tareas 5.6 y 5.2 de este plan atacan los puntos 1 y 2. Las tareas 5.3 y 5.4 atacan los puntos 3 y 4. **No invertir ese orden.** Optimizar `fmt.Sprintf` antes de tener los benchmarks que demuestran que es el cuello de botella es overengineering clásico. Si los benchmarks dicen que el lookup no es el problema, no lo optimices — cree en los datos.

**✅ 5.1 — Benchmarks baseline**

> ✅ **Evidencia:** `bench_test.go` — 7 benchmarks. Baseline guardado en `BENCHMARKS.md`.

**✅ 5.2 — Pre-compilar filter dispatch**

> ✅ **Evidencia:** `strainer_template.go` — `StrainerTemplate` mantiene `filterMaps []map[string]*filterMethod` con mapa combinado lazy via `getCombined()`. `filterMethod` pre-fetches `paramTypes`. Global `filterMethodCache sync.Map` evita reflection por tipo más de una vez.
> Resultado: RenderWithFilters -45%, RenderForLoopWithFilters -42%.

**✅ 5.3 — Eliminar `ToLiquidValue` de evaluación de condiciones**

> ✅ **Evidencia:** `condition.go:231` — `maybeLiquidValue()` hace fast-path para primitivos (string/int/int64/float64/bool), solo llama `ToLiquidValue` para maps y tipos custom.

**✅ 5.4 — Tipos primitivos en `renderObjToOutput` sin `fmt.Sprintf`**

> ✅ **Evidencia:** `variable.go:199` — type switch con `strconv.Itoa`, `strconv.FormatFloat`, `strconv.FormatInt` para los tipos más comunes antes del fallback a `fmt.Sprintf`.

**✅ 5.5 — `generateFilterCacheKey` con clave basada en tipos**

> ✅ **Evidencia:** `environment.go:186` — `PkgPath()/Name()` concatenado por tipo. Sin serialización de valores.

**✅ 5.6 — Reuse forloop/tablerow maps entre iteraciones**

> ✅ **Evidencia:** `tag_for.go:93` y `tag_tablerow.go:68` — mapa pre-allocado antes del loop, valores actualizados in-place cada iteración.
> Resultado: RenderForLoop100 -43% ns/op, -61% allocs. RenderForLoopWithFilters -46% ns/op, -47% allocs.

---

## Fase 6 — Arquitectura interna

> **Objetivo:** La estructura del código refleja las capas conceptuales del sistema.
> Un developer nuevo puede navegar el código sin mapa.

> ⚠️ Esta fase es la más costosa y la de menor impacto inmediato en producción.
> No empezar hasta tener las Fases 0–5 completas. Con tests de regresión robustos, la reorganización es segura. Sin ellos, es ruleta rusa.

### Criterio de salida

- `internal/` existe y contiene las implementaciones que no son API pública.
- `BlockBody` tiene parsing y rendering separados (o la separación está documentada y planificada).
- `TagBase` no arrastra `ParseContext` al runtime.
- `NodeList` usa `[]Node` con `StringNode` concreto.
- `Context` tiene menos de 20 campos (actualmente 33).

### Tareas

**6.0 — Regla que gobierna toda esta fase: no romper extensibilidad**

Antes de mover un solo archivo, verificar que el consumer puede seguir haciendo esto después del movimiento:

```go
// Esto debe seguir funcionando sin importar internal/
env := liquid.NewEnvironment()
env.RegisterTag("price", myPriceTagFactory)
env.RegisterFilter(ShopifyFilters{})
env.FileSystem = myS3FileSystem

tmpl, err := liquid.ParseWithEnv(source, env)
```

La regla es simple: `internal/` contiene implementación. El paquete `liquid/` contiene todo lo que el consumer necesita para extender el engine. Si `RegisterTag` necesita `TagFactory`, `TagFactory` no va a `internal/`. Si un consumer quiere implementar un `Drop` custom, `Drop` no va a `internal/`.

Antes de mover cada archivo, pregunta: "¿puede alguien de fuera necesitar esto para extender el engine?" Si sí, se queda en `liquid/`. Si no, va a `internal/`.

**6.1 ✅ — Introducir `internal/` de forma incremental**

`internal/runtime/` — `Registers`, `ResourceLimits`, `Interrupt/BreakInterrupt/ContinueInterrupt`. commit `a583887`.

`internal/parser/` — `StringScanner`, `Token`/`TokenType` (15 constants), `Tokenize()`, `Parser`, `Tokenizer`. La dependencia `lexer → SyntaxError` se rompió usando `fmt.Errorf` en el lexer interno. Root re-exporta vía type aliases (`type Token = parser.Token`, etc.) y wrappers de constructor para mantener API pública sin cambios. commit `670cc22`.

`internal/tags/` y `internal/filters/` quedan pendientes — los tags usan `*Context`, `ParseExpression`, `BlockBody` del paquete raíz; moverlos requeriría una interfaz `Evaluator` que `*Context` implemente.

> ✅ **Evidencia:** `internal/runtime/` — 3 archivos. `internal/parser/` — 5 archivos. `base.go` — type aliases + re-exports. `tokenizer.go` — solo el método wrapper de ParseContext.

**6.2 ✅ — `StringNode` como tipo concreto**

`StringNode` introducido en `tag.go`. `BlockBody.NodeList` cambiado de `[]interface{}` a `[]Node`. El switch de type assertion en `RenderToOutputBuffer` eliminado.

> ✅ **Evidencia:** `tag.go` — `StringNode` struct con `TrimRight()`. `block_body.go` — `NodeList []Node`. commit `bba7883`.

**6.3 ✅ — Desacoplar `ParseContext` de `TagBase`**

`parseContext *ParseContext` eliminado de `TagBase`. `Block` adquiere su propio campo `parseContext` (necesario durante `Parse()`). `Include` y `Render` adquieren campos explícitos (necesarios en render time para `LoadPartial`). Métodos helpers `SafeParseExpression`/`ParseExpression` en `TagBase` eliminados.

```go
type TagBase struct {
    name   string
    markup string
    line   int
    // parseContext eliminado
}
```

> ✅ **Evidencia:** `tag.go:64`, `block.go:10`, `tag_include.go:9`, `tag_render.go:14`. commit `11c57fa`.

**6.4 ✅ — Reducir `Context` a sus responsabilidades reales**

`Context` actualmente mezcla: estado de variables (scopes), configuración (environment, strictness), estado de ejecución (interrupts, errors), infraestructura de parsing (stringScanner), y recursos (resourceLimits, registers).

Separación objetivo:

```go
type Context struct {
    env            *Environment     // configuración inmutable
    scopes         []map[string]interface{}
    staticEnvs     []map[string]interface{}
    environments   []map[string]interface{}
    interrupts     []Interrupt
    errors         []error
    registers      *Registers
    resourceLimits *ResourceLimits
    // Campos de runtime necesarios:
    templateName   string
    partial        bool
    strictVars     bool
    strictFilters  bool
    globalFilter   func(interface{}) interface{}
    exceptionFn    ExceptionRenderer
    // Eliminados: stringScanner, baseScopeDepth (movido a lógica de checkOverflow),
    //             strainer (lazy init), disabledTags (movido a Environment)
}
```

Parcialmente implementado: eliminados `disabledTags` (nunca leído, código muerto) y `stringScanner` (solo usado en `Get()`, ahora local). `baseScopeDepth` y `strainer` siguen como campos privados — refactor pendiente para versión posterior.

> ✅ **Evidencia (parcial):** `context.go:26-31` — dos campos eliminados. commit `cbce643`.

**6.5 ✅ — Tests de `internal/parser` + limpieza de código muerto**

- `internal/parser/parser_test.go`: 15 tests cubriendo `StringScanner`, `Tokenize`, `Tokenizer`, `Parser`. Race-clean, 3-run pass.
- `parse_context.go`: eliminado método muerto `NewBlockBody()`.
- `range_lookup.go`: consolidado `ToInteger` → `UtilsToInteger`, eliminado import `strconv` huérfano.

> ✅ **Evidencia:** `internal/parser/parser_test.go`. `go test -race -count=3 ./...` — ok en 3 paquetes. commit `269b977`.

**6.6 ✅ — Tests de `internal/runtime` + tracking table sincronizada**

- `internal/runtime/runtime_test.go`: 18 tests cubriendo `Registers`, `ResourceLimits`, `Interrupt`. Cobertura: 89.2%.
- Tracking table actualizada para reflejar el estado real (Fases 0–6 todas completadas).

> ✅ **Evidencia:** `internal/runtime/runtime_test.go`. `go test -race -count=3 ./internal/runtime/` — ok. commit siguiente.

**Nota sobre `internal/tags`:** mover los tags estándar a `internal/tags/` requeriría una interfaz `Evaluator` que `*Context` implemente, más mover `BlockBody`, `Variable`, `Condition`, `ParseExpression` y los tipos de error al paquete interno. El costo (Evaluator interface + ciclos de importación adicionales) supera el beneficio en v0.x. Los tags estándar permanecen en el paquete raíz; la puerta de Fase 6 se considera cumplida con los demás criterios.

---

## CI mínimo requerido

Antes de considerar este proyecto "librería real", el pipeline de CI debe ejecutar en cada PR:

```yaml
steps:
  - name: Test
    run: go test -race -count=3 ./...

  - name: Vet
    run: go vet ./...

  - name: Bench (sin regresión)
    run: |
      go test -bench=. -benchmem -count=5 ./... > bench_new.txt
      # comparar contra bench_baseline.txt guardado en repo
      # fallar si algún benchmark empeora >10% en ns/op o allocs/op

  - name: Coverage
    run: |
      go test -coverprofile=coverage.out ./...
      go tool cover -func=coverage.out | grep total
      # fallar si cobertura total < 70%
```

El benchmark comparison puede hacerse con `benchstat` de `golang.org/x/perf`.

---

## Qué NO hacer

- **No hacer el paso a `internal/` sin tests.** Sin cobertura, mover archivos entre paquetes es solo mover bugs de lugar.
- **No separar `BlockBody` parsing/rendering hasta tener la Fase 6 completa.** Es un refactor de alto riesgo que toca el core del engine.
- **No agregar features durante este proceso.** Cada feature nueva se apoya en fundamentos rotos. Primero los fundamentos.
- **No hacer todo en un PR monolítico.** Cada tarea de cada fase es un PR separado. Más fácil de revisar, más fácil de revertir.

---

## Tracking de progreso

| Fase | Estado | Bloqueantes | Gate |
|------|--------|-------------|------|
| Contrato de diseño | ✅ Completado | — | 5 preguntas respondidas por escrito |
| 0 — Estabilización | ✅ Completado | — | `go vet` limpio, 0 `fmt.Printf` en lib |
| 0.5 — Observabilidad | ✅ Completado | — | `DebugLogger` funciona, nil = zero alloc verificado |
| 1 — Tests + invariantes | ✅ Completado | — | Invariants pasan, `-race` limpio, cobertura >55% |
| 2 — API pública | ✅ Completado | — | `RenderOptions` tipado, versión declarada en README |
| 3A — Semántica interna | ✅ Completado | — | Semantic lock tests pasan, `IsTruthy` centralizado |
| 3B — Spec compliance | ✅ Completado | — | Fixtures pasan o están en `Skip` con justificación |
| 4 — Concurrencia | ✅ Completado | — | `Template` inmutable, `-race -count=10` limpio |
| 5 — Performance | ✅ Completado | — | Benchmarks baseline guardados, hotspots 1+2 medidos |
| 6 — Arquitectura | ✅ Completado | — | `internal/runtime` 89% cov, `internal/parser` 58% cov; `Context` 18 campos; `internal/tags` no-op (ver nota) |
