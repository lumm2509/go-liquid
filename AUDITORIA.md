# Auditoría Técnica — `liquid`

> Módulo: `github.com/go-liquid`
> Fecha: 2026-03-20
> Archivos auditados: 54 archivos `.go`, `go.mod`, `go.sum`

---

## Resumen ejecutivo

Port funcional de Shopify Liquid en Go. El motor de parsing, los 22 tags, la gestión de scopes y los resource limits están presentes y operativos. No es un desastre arquitectónico total — hay intención de diseño visible en varios lugares.

El problema real es que este código nunca terminó de decidir qué es: ¿librería? ¿módulo interno? ¿prototipo? Esa ambigüedad permea cada capa: todo es público, hay estado global mutable, los límites entre fases de parsing y rendering están violados, y el único test verifica que `"Hello World"` se renderiza correctamente.

---

## 🗂️ Estructura de Carpetas

**54 archivos en un único directorio plano. Sin subdirectorios. Sin paquetes internos.**

```
liquid/
├── base.go
├── block.go
├── block_body.go
├── condition.go
├── context.go
├── document.go
├── drop.go
├── environment.go
├── errors.go
├── expression.go
├── file_system.go
├── i18n.go
├── interrupts.go
├── lexer.go
├── parse_context.go
├── parser.go
├── partial_cache.go
├── range_lookup.go
├── registers.go
├── resource_limits.go
├── standard_filters.go
├── standard_tags.go
├── strainer_template.go
├── string_scanner.go
├── tag.go
├── tag_assign.go
├── tag_capture.go
... (22 tag_*.go)
├── template.go
├── template_factory.go
├── tokenizer.go
├── utils.go
├── variable.go
├── variable_lookup.go
├── basic_test.go
```

En Go, un paquete flat de menos de 15-20 archivos es perfectamente aceptable. A 54 archivos, la estructura plana es una señal de que nadie paró a pensar en la organización. El desarrollador que abre este proyecto por primera vez no tiene ninguna pista visual de dónde empieza qué. Todo está al mismo nivel de importancia aparente: utilidades, parsing, runtime, filtros, infraestructura — un mar de archivos sin jerarquía.

La estructura obvia que debería existir:

```
liquid/
├── template.go          ← API pública
├── environment.go       ← API pública
├── drop.go              ← API pública (interfaz para usuarios)
├── file_system.go       ← API pública (interfaz para usuarios)
├── errors.go            ← API pública
└── internal/
    ├── parser/          ← lexer, tokenizer, string_scanner, parser, expression
    ├── runtime/         ← context, registers, resource_limits, interrupts
    ├── tags/            ← todos los tag_*.go
    ├── filters/         ← standard_filters, strainer_template
    └── util/            ← utils, condition, range_lookup, variable_lookup
```

El paquete `internal/` en Go impide que los consumidores de la librería importen implementaciones internas directamente. Es la herramienta diseñada exactamente para este caso. No se usa.

---

## 🧱 Capas Semánticas

Existen cinco capas conceptuales en el código. El problema es que **no están separadas, están entremezcladas**:

| Capa | Responsabilidad | Archivos |
|---|---|---|
| **API Pública** | Lo que el consumer usa | `template.go`, `environment.go`, `drop.go`, `file_system.go`, `errors.go` |
| **Parsing** | Lexing, tokenización, construcción del AST | `lexer.go`, `tokenizer.go`, `parser.go`, `string_scanner.go`, `parse_context.go`, `expression.go` |
| **AST / Nodos** | Representación del árbol parseado | `document.go`, `block.go`, `block_body.go`, `variable.go`, `variable_lookup.go`, `range_lookup.go`, todos los `tag_*.go` |
| **Runtime** | Ejecución, resolución de variables, scopes | `context.go`, `registers.go`, `resource_limits.go`, `interrupts.go` |
| **Infraestructura** | Filtros, filesystem, i18n, caché | `standard_filters.go`, `strainer_template.go`, `file_system.go`, `partial_cache.go`, `i18n.go` |

### Violaciones concretas de boundaries:

**1. `BlockBody` mezcla parsing Y rendering en la misma struct.**
`BlockBody.Parse()` construye el árbol. `BlockBody.RenderToOutputBuffer()` lo ejecuta. Son fases completamente distintas del pipeline, con responsabilidades distintas, viviendo en el mismo tipo. Esto dificulta testear cada fase por separado y hace que el tipo tenga dos razones para cambiar.

**2. `TagBase` lleva `parseContext` al runtime.**
```go
// tag.go
type TagBase struct {
    name         string
    markup       string
    line         int
    parseContext *ParseContext  // ← parse-time state en un nodo del AST
}
```
`ParseContext` es un objeto de la fase de parsing. Contiene `expressionCache`, `stringScanner`, `Environment`, `Warnings`. Los tags lo cargan consigo durante el rendering. Esto significa que el estado de parsing sigue vivo durante la ejecución, ocupando memoria y creando un acoplamiento innecesario entre fases.

**3. `Context` (runtime) contiene un `StringScanner` (herramienta de parsing).**
```go
// context.go
type Context struct {
    // ...campos de runtime...
    stringScanner  *StringScanner  // ← ¿qué hace un scanner de parsing en el runtime?
}
```
El `StringScanner` existe para el método `Context.Get(expression string)` que parsea una expresión en tiempo de ejecución. Eso en sí mismo es un diseño cuestionable — parsear durante el rendering, en lugar de parsear todo por adelantado — y contamina el `Context` con infraestructura de parsing.

**4. `variable_lookup.go` mezcla cuatro niveles de abstracción en un método.**
`accessProperty` maneja en secuencia: maps, slices, commands (size/first/last), struct fields via reflection, struct methods via reflection, Y propiedades especiales de strings de color (`#FFFFFF.red`). Son niveles de abstracción completamente distintos — acceso genérico a datos vs. magia de dominio específica — en el mismo método sin ninguna separación.

---

## 🚪 API Pública — Puntos de Entrada para Consumidores

Esta es la API que un developer debería usar para consumir esta librería:

```go
// Parsear un template
template, err := liquid.Parse(source, nil)

// Renderizar con variables
output, err := template.Render(assigns, nil)

// Crear un environment con tags/filtros personalizados
env := liquid.BuildEnvironment(func(e *liquid.Environment) {
    e.RegisterTag("mytag", myTagFactory)
    e.RegisterFilter(MyFilters{})
    e.FileSystem = myFileSystem
})
```

El problema es que el consumidor no tiene ninguna guía sobre esto. No hay documentación de la API pública, no hay README, no hay ningún ejemplo. Pero más importante:

**Todo está exportado. Todo.**

```go
// Tipos que un consumer NUNCA debería necesitar pero que son públicos:
liquid.BlockBody         // Nodo interno del AST
liquid.Tokenizer         // Herramienta interna de parsing
liquid.ParseContext      // Estado interno de parsing
liquid.StringScanner     // Herramienta interna de lexing
liquid.Strainer          // Implementación interna de filtros
liquid.StrainerTemplate  // Implementación interna de filtros
liquid.RangeLookup       // Nodo interno del AST
liquid.VariableLookup    // Nodo interno del AST
liquid.Parser            // Herramienta interna de parsing
liquid.Token             // Tipo interno del lexer
liquid.TokenType         // Tipo interno del lexer
liquid.Document          // Nodo raíz interno
liquid.TagBase           // Implementación interna de tags
```

Exportar todo esto tiene dos consecuencias directas:
1. **El consumer no sabe qué usar.** Abres el `godoc` y ves 50 tipos. ¿Cuáles son para mí y cuáles son detalles de implementación?
2. **No puedes cambiar la implementación sin breaking changes.** Si `Tokenizer` es público, cualquier cambio en su estructura es un breaking change para algún consumer. Exportar innecesariamente es deuda técnica que se cobra con intereses.

### Estado Global Mutable — El Problema más Silencioso

```go
// environment.go
var (
    defaultEnv     *Environment
    defaultEnvOnce sync.Once
)

func DefaultEnvironment() *Environment {
    defaultEnvOnce.Do(func() {
        defaultEnv = NewEnvironment()
    })
    return defaultEnv
}
```

`DefaultEnvironment()` es un singleton global. `Parse()` (la función de nivel de paquete) lo usa internamente. Esto significa:

- Si el consumer llama `DefaultEnvironment().RegisterTag(...)`, contamina el environment de TODOS los templates parseados con `Parse()` en ese proceso.
- En tests concurrentes, el estado del singleton se comparte entre tests.
- `DangerouslyOverride` — nótese el nombre — muta el puntero global sin ninguna protección de concurrencia real.

```go
// environment.go — exportado públicamente
func DangerouslyOverride(env *Environment, fn func()) {
    old := defaultEnv
    defaultEnv = env           // ← mutación de estado global
    defer func() { defaultEnv = old }()
    fn()
}
```

El autor sabía que era peligroso (lo dice el nombre), lo implementó de todas formas, y lo exportó. Esto no es API de librería, es un mecanismo para tests que se filtró a producción.

---

## 🔥 Problemas Críticos

- **`Sort` filter no ordena.** Devuelve una copia sin ordenar. Bug silencioso. Cualquier template que use `| sort` recibe datos desordenados sin error.

  ```go
  func (f StandardFilters) Sort(input interface{}, property ...interface{}) []interface{} {
      res := make([]interface{}, rv.Len())
      for i := 0; i < rv.Len(); i++ {
          res[i] = rv.Index(i).Interface()
      }
      return res // ← copia sin ordenar
  }
  ```

- **`fmt.Printf` en el path de renderizado de producción.** Una librería no escribe a stdout. Nunca.

  ```go
  // strainer_template.go:117
  fmt.Printf("Warning: Filter '%s' (Pascal: '%s') not found. Returning input.\n", ...)

  // block_body.go:220
  fmt.Printf("Liquid Render Error (Line %d): %v\n", node.LineNumber(), err)

  // environment.go:86
  fmt.Printf("Warning: can't modify frozen environment, skipping tag %s\n", name)

  // base.go:82
  fmt.Printf("Liquid default error handler: %v\n", err)  // DefaultErrorHandler global
  ```
  Son al menos 4 sitios distintos. No es un olvido, es un patrón.

- **Bug de `append` que puede corromper estado compartido.**

  ```go
  // context.go
  allEnvs := append(c.Environments, c.StaticEnvironments...)
  ```
  Si `c.Environments` tiene capacidad spare, este `append` muta el backing array compartido. En subcontextos que comparten environments esto puede corromper estado silenciosamente.

- **Un solo test para 54 archivos.** `TestBasicRender` renderiza `"Hello {{ name }}"`. No es una suite de tests, es un placeholder.

- **`generateFilterCacheKey` usa `fmt.Sprintf("%v", filters)` como hash.**

  ```go
  // environment.go
  func generateFilterCacheKey(filters []interface{}) string {
      return fmt.Sprintf("%v", filters) // ← colisiones posibles, no es un hash real
  }
  ```
  Dos conjuntos de filtros distintos que se formateen igual tendrán la misma clave. El doble-lock pattern del caché se construye sobre una clave no fiable.

---

## ⚠️ Problemas Importantes

- **Errores retornados como `interface{}`.** `lookupAndEvaluate` y `VariableLookup.Evaluate` retornan `fmt.Errorf(...)` donde se espera un valor de variable. Los 15 tipos de error definidos no se usan en los paths más críticos.

  ```go
  if c.StrictVariables && raiseOnNotFound && !exists {
      return fmt.Errorf("undefined variable %s", key) // ← error como valor de variable
  }
  ```

- **`defer c.Pop()` descarta el error.** `context.go:130`. El error de stack underflow se pierde.

- **`checkOverflow` no detiene la ejecución.**

  ```go
  func (c *Context) checkOverflow() {
      if c.baseScopeDepth+len(c.Scopes) > 100 {
          c.HandleError(fmt.Errorf("StackLevelError: Nesting too deep"), 0)
          // ← sin return. Continúa ejecutando después del overflow.
      }
  }
  ```

- **Regex compiladas dentro de métodos en el hot path.**

  ```go
  func (f StandardFilters) StripHtml(input interface{}) string {
      re := regexp.MustCompile(`<[^>]*>`) // ← compilar aquí es incorrecto
  ```

- **`lookupAndEvaluate` puede panic.** Accede a `results[0]` sin verificar que `results` no esté vacío.

- **`DangerouslyOverride` es API pública.** Un mecanismo para tests que muta estado global exportado como función pública.

---

## 🧠 Problemas de Diseño

- **`BlockBody` hace parsing Y rendering.** Dos fases del pipeline en una sola struct. Imposible testear por separado.

- **`TagBase` arrastra `ParseContext` al runtime.** Parse-time state en cada nodo del AST durante el rendering. Ver sección de capas semánticas.

- **`Context` es un objeto dios.** Scopes, environments, interrupts, resource limits, filtros, strainer, disabled tags, string scanner, base scope depth, exception renderer, template name — todo en un solo tipo de 33 campos. Es el punto de acoplamiento de todo el sistema.

- **`NodeList []interface{}` mezcla `string` y `Node`.** Fuerza type assertions en el loop de rendering principal. Es un diseño de Ruby portado literalmente sin adaptarlo al type system de Go. La solución es `StringNode` implementando `Node`.

- **Module path incoherente.** `github.com/go-liquid` — un engine de Liquid dentro de un path de `ledger-io`. Fork no renombrado o paquete no extraído correctamente.

- **`accessProperty` mezcla abstracciones distintas.** Maps, slices, commands (size/first/last), struct fields, struct methods, Y propiedades de strings de color (`#FFFFFF.red`) en un único método. El color parsing es magia de dominio específico que no tiene nada que hacer junto al acceso genérico de propiedades.

- **`SliceCollection` inyecta `"id"` silenciosamente.** Añade una clave `"id"` a los valores al iterar sobre maps. Comportamiento no documentado que altera la estructura de datos del caller.

- **Singleton `DefaultEnvironment` con mutación global.** El estado global compartido para una librería de templates que puede renderizar concurrentemente es un diseño incorrecto. Ver sección de API pública.

- **`Date` filter con conversión `strftime` via `strings.ReplaceAll`.** Cubre 9 de ~40 directivos. Si el string de formato ya contiene `2006`, `01`, etc., habrá doble-sustitución. La conversión no es reversible ni correcta en el caso general.

---

## 🔑 Niveles de Abstracción

El problema de niveles de abstracción más claro está en `accessProperty` (`variable_lookup.go`):

```
Nivel 1 — Acceso a colecciones:   map[key], slice[idx]
Nivel 2 — Comandos semánticos:    .size, .first, .last
Nivel 3 — Reflection genérica:    struct fields, struct methods
Nivel 4 — Magia de dominio:       "#FFFFFF".red, "#FFFFFF".green
```

Estos cuatro niveles conviven en un método de 120 líneas sin ninguna separación. El nivel 4 (color parsing) es específico de Shopify Liquid para themes y no tiene absolutamente nada que hacer en el código de resolución de propiedades genéricas.

Lo mismo pasa en `Render.RenderToOutputBuffer` (`tag_render.go`): hace loading del partial, construcción del subcontexto, resolución de variables, manejo del `for` loop Y el renderizado — todo en 55 líneas sin ninguna delegación.

El síntoma general: **funciones que describen un proceso completo en lugar de una abstracción específica**. Cuando una función hace A, luego B, luego C, luego D, no es una función — es un script.

---

## 🧹 Code Smells

- **Comentarios en español e inglés mezclados** en los mismos archivos. Sin consistencia.

- **`context_variable_name` en snake_case en Go.** `tag_render.go`. Las variables locales en Go son `camelCase`.

- **Código de debug comentado en producción.** `strainer_template.go:105-110`.

  ```go
  // } else {
  //     // DEBUG: Uncomment if needed
  //     // fmt.Printf("Method %s (Pascal: %s) not found...")
  ```

- **`Escape` manual reimplementando `html.EscapeString`.** La stdlib tiene exactamente esto.

- **Comparación por serialización de string en `Uniq` y `Where`.**

  ```go
  key := fmt.Sprintf("%v", val)              // Uniq
  fmt.Sprintf("%v", val) == fmt.Sprintf("%v", target) // Where
  ```
  Incorrecto semánticamente y frágil.

- **`Default` con lógica redundante.** Verifica `isTruthy` después de ya haber chequeado `nil`, string vacío y colecciones vacías explícitamente. Esos casos ya están cubiertos.

- **`StaticRegisters = Registers` como alias.** `registers.go:94`. Un alias de tipo que no añade semántica, solo ruido.

- **Comentarios que parafrasean el código.** `// Buscar en los scopes (de arriba hacia abajo)` sobre un for range sobre `c.Scopes`. Si el código lo dice, el comentario sobra.

- **`// Replaced panic with log to prevent server crashes`** en `base.go:81`. El comentario documenta que alguien reemplazó un panic con un `fmt.Printf`, como si eso fuera una mejora. No lo es — ambas opciones son incorrectas para una librería. La correcta es devolver el error.

---

## 🧪 Testing

- **1 test para todo el proyecto.** 54 archivos, 0 cobertura real.

  ```go
  func TestBasicRender(t *testing.T) {
      source := "Hello {{ name }}"
      // ...
      require.Equal(t, "Hello World", output)
  }
  ```

- **Sin tests para ningún tag.** `if`, `for`, `case`, `unless`, `capture`, `assign`, `render`, `include`, `cycle`, `tablerow` — ninguno verificado.

- **Sin tests para ningún filtro.** El `Sort` roto lleva quién sabe cuánto tiempo en producción porque no hay tests.

- **Sin tests de regresión contra el spec oficial.** El repositorio de Ruby Liquid tiene cientos de fixtures. Ninguno verificado.

- **Sin tests de concurrencia.** El singleton `DefaultEnvironment`, el `Registers` compartido en subcontextos, y el `expressionCache` compartido en `ParseContext` son candidatos naturales a race conditions que no tienen ni un test.

- **Sin tests de resource limits.** El mecanismo de seguridad más importante de la librería no tiene ninguna cobertura.

---

## 🚀 Performance

- **Regex compiladas en hot path.** `StripHtml` compila en cada llamada. Mover a `var` de paquete.

- **Reflection sin cacheo de tipo en `Map`, `Where`, `Uniq`, `Sort`.** Para colecciones grandes con structs, la reflection se repite en cada elemento.

- **`ToLiquidValue` recorre maps recursivamente sin límite de profundidad.** Para inputs anidados profundamente, puede ser O(n) con stack overflow teórico en estructuras circulares.

- **`squashInstanceAssignsWithEnvironments` es O(scopes × environments).** Llamado en cada construcción de `Context`.

- **`generateFilterCacheKey` serializa los filtros con `fmt.Sprintf`.** La generación de la clave para el caché es O(n) y genera strings garbage en cada llamada con filtros adicionales.

---

## 🔐 Seguridad

- **No hay escape de HTML por defecto.** `{{ variable }}` no escapa HTML. Cualquier aplicación web que use esta librería para renderizar input de usuario sin aplicar `| escape` explícitamente tiene XSS. Debe estar documentado en mayúsculas.

- **`LocalFileSystem` no valida symlinks.** Valida path traversal con `..` y verifica que el path absoluto esté dentro del root, pero un symlink que apunte fuera del root pasaría la validación.

- **Reflection sin bound checking real.** El fallback de `reflect.Zero(targetType)` en `Strainer.Invoke` puede resultar en comportamiento incorrecto silencioso.

- **`DangerouslyOverride` es thread-unsafe.** Muta `defaultEnv` (puntero global) sin ningún mutex. En un servidor HTTP bajo carga concurrente, usarlo es una race condition garantizada.

---

## 💡 Recomendaciones Directas

1. **Mueve implementaciones internas a `internal/`.** `BlockBody`, `Tokenizer`, `ParseContext`, `StringScanner`, `Parser`, `Strainer`, `StrainerTemplate`, `VariableLookup`, `RangeLookup`, `Document` no deben ser importables por consumers. Usa `internal/`.

2. **Escribe un `README.md` con la API pública.** Tres ejemplos: parsear, renderizar, registrar tags/filtros custom. El consumer actual no tiene guía de ningún tipo.

3. **Arregla `Sort`.** Implementa o marca como `panic("not implemented")`. Un filtro que miente es peor que uno que falla.

4. **Elimina todos los `fmt.Printf`.** Mínimo 4 instancias en `strainer_template.go`, `block_body.go`, `environment.go`, `base.go`. Usa el `ExceptionRenderer` o devuelve errores.

5. **Arregla el bug de `append`:**
   ```go
   // Reemplazar:
   allEnvs := append(c.Environments, c.StaticEnvironments...)
   // Por:
   allEnvs := make([]map[string]interface{}, len(c.Environments)+len(c.StaticEnvironments))
   copy(allEnvs, c.Environments)
   copy(allEnvs[len(c.Environments):], c.StaticEnvironments)
   ```

6. **Elimina el singleton `DefaultEnvironment` de la función `Parse` pública.** La función `Parse(source, options)` debe recibir el environment explícitamente, o usar `NewEnvironment()` por defecto sin singleton mutable.

7. **Reemplaza `Escape` con `html.EscapeString`** de la stdlib.

8. **Mueve las regex a nivel de paquete.** Son constantes de compilación.

9. **Haz que `checkOverflow` retorne** después de registrar el error.

10. **Escribe tests.** Mínimo: un test por tag, un test por filtro, tests de resource limits, tests de concurrencia sobre el environment. Los fixtures del [repositorio oficial de Ruby Liquid](https://github.com/Shopify/liquid/tree/main/test) son el punto de partida obvio.

11. **Implementa `StringNode` que implemente `Node`** y elimina el `[]interface{}` del `NodeList`.

12. **Divide `accessProperty` en responsabilidades claras.** El color parsing pertenece a su propio método o a un handler específico, no junto al acceso genérico de propiedades.

---

## 🧨 Veredicto Final

**Nota: 4/10**

La arquitectura base tiene intención. El pipeline parsing → AST → rendering está conceptualmente correcto, el sistema de tags es extensible, los resource limits existen, el manejo de subcontextos aislados para `render` es correcto. Hay trabajo real aquí.

Pero el código nunca terminó de cruzar la línea entre "funciona en mis tests manuales" y "es una librería confiable". Las señales son claras: `Sort` que no ordena, cuatro `fmt.Printf` en producción, un singleton global mutable sin protecciones reales, estado de parsing filtrándose al runtime, y exactamente un test que verifica `"Hello World"`.

El mayor riesgo operacional es que **no tienes visibilidad sobre qué más está roto**. Con `Sort` silenciosamente incorrecto ya en producción sin que nadie lo detectara, la pregunta correcta no es "¿qué bugs hay?" sino "¿cuántos bugs como este hay que aún no hemos encontrado?".

La respuesta, con esta cobertura de tests, es: no lo sabes.

---

# Auditoría — Extensión Crítica

> Esta sección continúa la auditoría anterior. No repite puntos ya cubiertos.
> Ejes: DX real, concurrencia, testing como riesgo operativo, plan de acción, ciclos de CPU.

---

## ⚙️ API Ergonomics (DX)

### El developer promedio no puede usar esta librería correctamente en el primer intento

No porque sea complicada. Sino porque sus defaults son peligrosos y sus fallos son silenciosos.

**1. XSS por diseño, sin documentación.**

```go
output, err := template.Render(map[string]interface{}{"name": userInput}, nil)
```

`userInput = "<script>alert(1)</script>"` se renderiza sin modificar. No hay autoescaping. No hay advertencia en la API. No hay `nil` option llamada `"auto_escape"`. No hay nada. Cualquier developer que venga de Django, Jinja2, Go templates, o Twig asume que el engine escapa por defecto — porque todos los engines modernos lo hacen. Esta librería no es esos engines y no lo dice en ningún lado.

**2. API completamente no-tipada.**

```go
// Parse
Parse(source string, options map[string]interface{})

// Render
template.Render(assigns map[string]interface{}, options map[string]interface{})
```

Las opciones válidas para `Render` son `"strict_variables"`, `"strict_filters"`, `"rethrow_errors"`, `"registers"`. Las de `Parse` incluyen `"environment"`, `"locale"`, `"error_mode"`, `"expression_cache"`, `"include_options_blacklist"`. Ninguna está documentada. Ninguna tiene un tipo. Si escribes `"strict_variable": true` (singular, sin 's') no hay error — simplemente se ignora silenciosamente y el modo estricto nunca se activa.

Un developer no tiene forma de descubrir qué opciones existen sin leer el código fuente. Eso no es una API — es un protocolo secreto.

**3. Errores reportados por dos canales distintos con semántica distinta.**

```go
output, err := template.Render(assigns, nil)
// err puede ser nil Y template.Errors puede tener errores
```

`Render` retorna un `error` para fallos fatales, pero también escribe en `ctx.Errors` para errores non-fatales que luego se copian a `t.Errors`. Un developer que hace `if err != nil { return }` y asume que el render fue correcto puede estar ignorando errores reales que quedaron en `template.Errors`. No hay documentación de cuándo usar uno vs. el otro.

**4. `Sort` silencioso. `Filter not found` silencioso. Variables missing silenciosas.**

Tres categorías de fallo que no retornan error:
- `| sort` devuelve datos desordenados sin aviso.
- Un filtro inexistente devuelve el input sin modificar (más un `fmt.Printf` a stdout que el consumer no ve).
- Una variable que no existe retorna `nil` (a menos que `strict_variables` esté activado, pero está desactivado por defecto y su activación requiere conocer la opción no-documentada).

La combinación es perfecta para debugging imposible: tu template no falla, simplemente produce resultados incorrectos.

**5. `panic` silenciado en Parse y Render.**

```go
// template.go
func (t *Template) Parse(source string, options map[string]interface{}) (templateResult *Template, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("liquid parse error: %v", r)
            templateResult = nil
        }
    }()
```

Panics reales — nil pointer dereferences, index out of bounds, type assertion failures — se convierten en errores genéricos `"liquid parse error: runtime error: ..."`. El stack trace se pierde. El contexto se pierde. Un bug real en el engine se convierte en un error opaco que no ayuda a nadie a entender qué pasó o dónde.

**6. `ResourceLimits` ilimitados por defecto.**

`DefaultResourceLimits` es un `map[string]interface{}` vacío. Esto significa que `RenderLengthLimit`, `RenderScoreLimit` y `AssignScoreLimit` son todos 0, lo que el código interpreta como "sin límite":

```go
if rl.RenderScoreLimit > 0 && rl.renderScore > rl.RenderScoreLimit {
```

En producción, un template malicioso o mal construido puede generar output infinito o iterar indefinidamente sin que nada lo detenga. El mecanismo de protección existe pero requiere configuración explícita que no está documentada en ningún lugar visible.

**7. `BlockBody.frozen = true` durante el rendering muta el AST parseado.**

```go
func (b *BlockBody) RenderToOutputBuffer(...) error {
    b.frozen = true // ← mutación del árbol en cada render
```

Renderizar un template muta su representación interna. Esto hace que un `*Template` parseado una vez no sea safe para ser usado concurrentemente ni para ser re-parseado después de renderizar. El consumer no tiene ninguna pista de que `Render` tiene side effects sobre la estructura interna del template.

**Conclusión:** La API no solo es ruidosa — es peligrosa. Es trivial usarla mal sin darte cuenta: XSS por default, estado global compartido sin aviso, filtros silenciosamente incorrectos, dos canales de error con semántica inconsistente, panics convertidos en errores opacos, y limits de seguridad desactivados por defecto. Eso no es solo mala DX, es un riesgo operativo activo.

---

## 🧵 Concurrencia

**Este código está escrito como si Go no tuviera goroutines.**

### Race condition #1 — `ResourceLimits` compartido entre renders concurrentes

```go
// template.go
ctx := NewContext(
    environments,
    t.InstanceAssigns,
    registers,
    rethrowErrors,
    t.ResourceLimits,  // ← puntero directo al ResourceLimits del Template
    ...
)
```

`t.ResourceLimits` es un campo del `Template`. `NewContext` lo recibe por puntero y lo usa directamente. Si diez goroutines llaman `t.Render()` simultáneamente, todas comparten el mismo `*ResourceLimits`. El campo `renderScore` se incrementa en el loop de rendering sin ningún mutex ni atomic:

```go
func (rl *ResourceLimits) IncrementRenderScore(amount int) {
    rl.renderScore += amount  // ← race condition bajo concurrencia
```

`go test -race` lo va a detectar inmediatamente. En producción, bajo carga, el `renderScore` se corrompe — los límites dejan de funcionar correctamente, y el mecanismo de seguridad principal de la librería falla en exactamente el momento en que más se necesita.

### Race condition #2 — `expressionCache` sin mutex

```go
// parse_context.go
expressionCache map[string]interface{}
```

Este mapa se escribe durante el parsing en `ParseExpression`. `ParseContext` puede ser compartido entre el template principal y sus partials (via `parseContext.options` que se pasa a `template.Parse`). Si dos goroutines parsean partials desde el mismo contexto, ambas escriben en el mismo mapa sin ninguna protección. En Go, writes concurrentes a un map causan panic o corrupción de memoria.

### Race condition #3 — `cachedPartials` en `LoadPartial`

```go
// partial_cache.go
cachedPartials, ok := context.Registers.Get("cached_partials").(map[string]interface{})
// ...
cachedPartials[cacheKey] = template  // ← escritura sin mutex
```

`cachedPartials` es un `map[string]interface{}` que vive en los `Registers` del contexto. `Registers.static` se comparte entre contextos padre e hijo (`sub.Registers.static = c.Registers.static` via `c.Registers.static` pasado a `NewRegisters`). Si dos goroutines renderizan el mismo template con partials concurrentemente, ambas pueden llegar a `LoadPartial` para el mismo partial, leer que no está en caché, y ambas intentar escribirlo simultáneamente.

### Race condition #4 — `defaultEnv` global sin mutex

```go
// environment.go
var defaultEnv *Environment

func DangerouslyOverride(env *Environment, fn func()) {
    old := defaultEnv
    defaultEnv = env           // ← escritura no atómica de puntero global
    defer func() { defaultEnv = old }()
    fn()
}
```

La lectura de `defaultEnv` en `DefaultEnvironment()` y la escritura en `DangerouslyOverride` no están sincronizadas. En el modelo de memoria de Go, esto es una data race. Que se llame "Dangerously" no lo hace aceptable.

### Race condition #5 — `Environment.Tags` sin mutex para lecturas

`RegisterTag` escribe en `e.Tags` sin mutex. `TagForName` lee de `e.Tags` sin mutex. Si un goroutine llama `RegisterTag` mientras otro renderiza un template (lo que invoca `TagForName`), es una data race. El `sync.RWMutex` que existe en `Environment` protege `strainerTemplateClassCache` correctamente — pero no se aplica a `Tags`.

### Race condition #6 — `Context.Errors` compartido por referencia

```go
// context.go
sub.Errors = c.Errors    // ← mismo slice subyacente
sub.Warnings = c.Warnings
```

El subcontexto comparte el mismo slice de errores que el contexto padre. Si el rendering principal y un partial renderizan concurrentemente (lo cual puede ocurrir en implementaciones de rendering paralelo), ambos hacen `c.Errors = append(c.Errors, err)` sobre el mismo slice. El `append` a un slice compartido sin mutex es una race condition que puede causar que se pierdan errores o que el slice se corrompa.

### `BlockBody.frozen` — write durante render, read durante parse

```go
func (b *BlockBody) RenderToOutputBuffer(...) error {
    b.frozen = true  // escritura
```
```go
func (b *BlockBody) Parse(...) (bool, error) {
    if b.frozen {    // lectura
        return false, fmt.Errorf("can't modify frozen Liquid::BlockBody")
    }
```

No hay mutex entre estas dos operaciones. Trivial de detonar en cualquier patrón de uso donde el parsing y el rendering se solapan.

**Resumen de races confirmadas:**

| Estructura | Tipo de race | Severidad |
|---|---|---|
| `ResourceLimits.renderScore` | Write/write concurrente | Alta — corrompe límites de seguridad |
| `expressionCache` (map) | Write/write concurrente | Alta — panic en Go runtime |
| `cachedPartials` (map) | Read/write concurrente | Alta — panic en Go runtime |
| `defaultEnv` (puntero global) | Write/read sin sync | Media — comportamiento indefinido |
| `Environment.Tags` (map) | Write/read concurrente | Media — panic en Go runtime |
| `Context.Errors` (slice) | Append concurrente | Baja — pérdida de errores |
| `BlockBody.frozen` (bool) | Write/read sin sync | Baja — falso positivo en parse |

---

## 🧪 Testing (Realidad)

**No tienes tests. Tienes un placeholder que finge ser test.**

```go
func TestBasicRender(t *testing.T) {
    source := "Hello {{ name }}"
    output, err := template.Render(map[string]interface{}{"name": "World"}, nil)
    require.Equal(t, "Hello World", output)
}
```

Esto no verifica el engine. Verifica que Go puede concatenar strings. Cualquier parser que devuelva el template sin procesarlo pasaría este test.

### Bugs que existen hoy, confirmados, sin detección posible:

- **`Sort` no ordena.** En producción. Sin detección.
- **`ResourceLimits` tiene race condition bajo concurrencia.** En producción. Sin detección.
- **`expressionCache` tiene race condition.** En producción. Sin detección.
- **`checkOverflow` no detiene la ejecución.** En producción. Sin detección.
- **`append` sobre `c.Environments` puede corromper estado.** En producción. Sin detección.

### Áreas completamente a ciegas:

| Componente | Cobertura |
|---|---|
| 22 tags (`if`, `for`, `case`, `unless`, `render`, etc.) | 0% |
| 30+ filtros | 0% |
| Condiciones con `and`/`or` anidados | 0% |
| Whitespace control (`{%-`, `-%}`) | 0% |
| Resource limits | 0% |
| Partials (`render`, `include`) | 0% |
| Error modes (`strict`, `lazy`, `warn`) | 0% |
| `Drop` interface | 0% |
| FileSystem (path traversal prevention) | 0% |
| I18n | 0% |
| Concurrencia | 0% |
| Color filters | 0% |
| Scope isolation en subcontextos | 0% |

### El riesgo concreto:

Cualquier cambio en `condition.go`, `variable_lookup.go`, `standard_filters.go`, o cualquiera de los 22 tags puede romper comportamiento en producción sin que ningún test lo detecte. No hay red de seguridad. Cada deploy es un experimento.

La pregunta correcta no es "¿cuántos bugs hay?". Es: dado que `Sort` lleva tiempo roto sin que nadie lo detecte, **¿cuántos otros filtros o tags tienen bugs equivalentes que tampoco has encontrado?**

---

## 🛠️ Plan de Acción Inmediato

Orden estricto de prioridad. Sin teoría.

### Fase 0 — Estabilización (antes de tocar nada más)

**0.1 Freeze el API pública documentada.**
Define en un `doc.go` exactamente qué es público intencionalmente:
```
Public API: Parse, Template, Template.Render, Environment, BuildEnvironment,
            RegisterTag, RegisterFilter, FileSystem, Drop, TagFactory, los tipos de error.
Everything else: implementación interna.
```
Esto establece el contrato. Sin esto, cualquier reorganización puede romper consumers accidentalmente.

**0.2 Ejecuta `go test -race ./...` ahora.**
Va a detectar múltiples races inmediatamente. Úsalas como baseline para saber el alcance real del problema de concurrencia.

### Fase 1 — Bugs críticos (esta semana)

**1.1 Arregla `ResourceLimits` para concurrencia.**
`Template.Render` debe copiar o crear un nuevo `ResourceLimits` por cada render, no compartir el puntero del template:
```go
// Crear nuevo ResourceLimits por render, no compartir el del template
renderLimits := t.ResourceLimits.Clone()
ctx := NewContext(..., renderLimits, ...)
```

**1.2 Arregla `expressionCache` con mutex o per-parse.**
Opción simple: cada `NewParseContext` crea su propio cache. Opción mejor: `sync.Map` para el cache compartido.

**1.3 Arregla `cachedPartials` con mutex.**
`map[string]interface{}` → `sync.Map` o wrapper con `sync.RWMutex`.

**1.4 Arregla `Sort`.**
Implementación mínima:
```go
import "sort"
// para []interface{} con elementos comparables como strings o números
```

**1.5 Elimina los 4+ `fmt.Printf` del código de librería.**
Cada uno reemplazado por el `ExceptionRenderer` correspondiente o retorno de error.

### Fase 2 — Tests (en paralelo con Fase 1)

**2.1 Golden tests contra el spec oficial.**
Descarga los fixtures de `https://github.com/Shopify/liquid/tree/main/test/integration`.
Conviértelos a tabla de tests en Go:
```go
var cases = []struct{ template, expected string; data map[string]interface{} }{
    {"{{ name | upcase }}", "WORLD", map[string]interface{}{"name": "world"}},
    // ...
}
```
Target mínimo: cubrir los 22 tags y los 30 filtros con al menos 3 casos cada uno (happy path, edge case, nil input).

**2.2 Tests de concurrencia.**
```go
func TestConcurrentRender(t *testing.T) {
    tmpl, _ := Parse("{{ name | upcase }}", nil)
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            tmpl.Render(map[string]interface{}{"name": "world"}, nil)
        }()
    }
    wg.Wait()
}
```
Ejecutar con `-race`. Cero races aceptables.

**2.3 Test de regresión para `Sort`.**
```go
func TestSortFilter(t *testing.T) {
    tmpl, _ := Parse(`{{ items | sort | join: "," }}`, nil)
    out, _ := tmpl.Render(map[string]interface{}{"items": []string{"c","a","b"}}, nil)
    require.Equal(t, "a,b,c", out)
}
```

### Fase 3 — Reorganización (después de tener tests)

**3.1 `internal/` para implementaciones internas.**
Mueve a `internal/`: `blockbody`, `tokenizer`, `parser`, `lexer`, `stringscanner`, `parsecontext`, `strainer`, `strainertemplate`, `variablelookup`, `rangelookup`, `document`, `tagbase`.

**3.2 Elimina el singleton `DefaultEnvironment` de `Parse`.**
`Parse` debe recibir un `*Environment` explícito, con `nil` creando un `NewEnvironment()` fresco, no el singleton global.

**3.3 Elimina `DangerouslyOverride` de la API pública.**
Si necesitas override para tests, usa `t.Cleanup` con un environment local.

### Fase 4 — Benchmarks mínimos

```go
func BenchmarkRenderSimple(b *testing.B) { /* {{ name }} */ }
func BenchmarkRenderWithFilters(b *testing.B) { /* {{ name | upcase | strip }} */ }
func BenchmarkRenderForLoop(b *testing.B) { /* {% for i in items %}...{% endfor %} */ }
func BenchmarkRenderWithPartials(b *testing.B) { /* {% render 'partial' %} */ }
func BenchmarkParseAndRender(b *testing.B) { /* Parse + Render completo */ }
```

Ejecuta con `-benchmem`. Los resultados actuales son la baseline. Cualquier commit que empeore >5% en allocations por operación requiere justificación.

---

## ⚡ Performance a Nivel de Ciclos

Cada decisión debe considerar ciclos de CPU. Hay al menos 1 ciclo que puedes salvar en cada uno de estos puntos. Aquí los sistémicos:

**1. `ToLiquidValue` copiando maps en cada evaluación de condición.**

```go
// condition.go
return operation(c, ToLiquidValue(leftVal), ToLiquidValue(rightVal))
```

`ToLiquidValue` para maps hace una copia completa de toda la estructura:
```go
for _, key := range rv.MapKeys() {
    newMap[k] = ToLiquidValue(rv.MapIndex(key).Interface()) // recursivo
}
```

Esto se ejecuta en CADA evaluación de condición en CADA render. Para un template con 20 condiciones y objetos con maps de 10 campos, son 400+ copias de maps por render. La mayoría innecesarias porque las condiciones solo necesitan leer valores, no copiarlos.

**2. `Strainer.Invoke` — O(n) lineal por cada aplicación de filtro.**

```go
for _, filter := range s.filters {
    m := val.MethodByName(method)     // reflection en cada filtro
    if !m.IsValid() {
        m = val.MethodByName(pascalMethod) // segunda reflection
    }
    if m.IsValid() { /* invocar */ }
}
```

Para N filtros registrados y F filtros aplicados en un template, cada render hace N×F lookups por reflection. `MethodByName` en reflection no es barato — hace una búsqueda por nombre. La solución es un mapa de nombre → método pre-compilado en `StrainerTemplate`, construido una vez cuando se registran los filtros.

**3. `fmt.Sprintf("%v", filters)` como cache key en `CreateStrainer`.**

```go
func generateFilterCacheKey(filters []interface{}) string {
    return fmt.Sprintf("%v", filters)
}
```

Llamado en cada `Context.Strainer()` cuando hay filtros adicionales. `fmt.Sprintf` con `%v` usa reflection internamente, aloca un string, y el resultado puede colisionar. Reemplazar con una clave basada en los type pointers:
```go
// Ejemplo: usar reflect.TypeOf(f).PkgPath() + reflect.TypeOf(f).Name()
```

**4. `SliceCollection` — aloca `[]interface{}` nuevo en cada iteración de `for`.**

```go
// tag_for.go
segment := SliceCollection(collection, nil, nil)
```

Para un `for` sobre un `[]string{"a","b","c"}`, `SliceCollection` aloca un nuevo `[]interface{}` copiando cada elemento. Para loops frecuentes sobre colecciones grandes, esto es presión significativa al GC. Si la colección ya es un slice, debería iterarse directamente.

**5. `tokensToMarkup` — reconstruye strings para re-parsearlos.**

```go
// condition.go
leftMarkup := tokensToMarkup(tokens[:i])
left, _ := ParseExpression(leftMarkup, ...)
```

El proceso es: string original → tokenizar → reconstruir string → re-parsear. Se pierde información al tokenizar solo para reconstruirla. El parser de condiciones debería trabajar directamente con los tokens en lugar de reconstruir markup intermedio.

**6. `Variable.ParseContext` — retiene toda la infraestructura de parsing en memoria durante el render.**

```go
type Variable struct {
    ParseContext *ParseContext  // ← todo el estado de parsing vivo durante rendering
```

`ParseContext` contiene `expressionCache` (todo el cache de expresiones), `stringScanner`, `templateOptions` (todos los options del template), `partialOptions`. Todo esto se mantiene vivo en memoria mientras cada `Variable` del AST existe, es decir, mientras el template exista. Para templates de larga vida con muchas variables, esto es retención de memoria innecesaria.

**7. `renderObjToOutput` — `fmt.Sprintf("%v", obj)` para tipos no-string.**

```go
output.WriteString(fmt.Sprintf("%v", obj))
```

Para `int`, `float64`, `bool` — los tipos más comunes después de `string` — hay rutas más baratas:
```go
case int:    output.WriteString(strconv.Itoa(v))
case float64: output.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
case bool:   if v { output.WriteString("true") } else { output.WriteString("false") }
```

`fmt.Sprintf` con reflection para estos tipos es 3-5x más lento que las conversiones directas de `strconv`.

**8. `condition.go:198` — `fmt.Printf` en el path de evaluación de condiciones.**

```go
fmt.Printf("Warning: Unknown operator %s\n", op)
```

Está en `interpretCondition`, que se llama en cada evaluación de condición. Si hay un operador desconocido, va a stdout en cada render de cada template que lo contenga. Una llamada a `fmt.Printf` en un hot path es un lock global en la implementación de Go's `os.Stdout`. En un servidor de alta carga, esto es contención.

---

## 🧨 Conclusión Extendida

La auditoría anterior le dio un 4/10. Con la información adicional sobre concurrencia y DX, esa nota es generosa.

El código tiene race conditions que van a manifestarse bajo carga real en un servidor HTTP. No son teóricas — son estructurales: el `ResourceLimits` compartido entre goroutines concurrentes es un bug que `go test -race` va a detectar en el primer test de concurrencia que alguien escriba. El `expressionCache` compartido sin mutex es otro. Ninguno de los dos requiere condiciones especiales para detonarse — solo llamar `Render` desde más de una goroutine sobre el mismo template.

La API activamente facilita el uso incorrecto. Un developer que sigue el happy path de esta librería va a producir endpoints con XSS, sin resource limits, usando el singleton global sin saberlo, y con `Sort` silenciosamente roto. No tiene que hacer nada malo — tiene que hacer exactamente lo que la API le sugiere.

Y no hay tests que hayan detectado ninguno de estos problemas porque no hay tests.

La pregunta que cualquiera debería hacerse antes de usar este código en producción es simple: **¿qué versión de este código estaría cómodo deployando en un servidor bajo carga real, con inputs de usuarios no confiables, desde múltiples goroutines concurrentes?**

La respuesta honesta es: ninguna versión que exista actualmente.

El camino hacia la producción real requiere, en orden: arreglar las race conditions, eliminar los `fmt.Printf`, arreglar `Sort`, agregar al menos 100 tests con cobertura real de tags y filtros, y ejecutar `go test -race` como paso obligatorio en el CI. Todo lo demás puede esperar.
