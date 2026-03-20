# Shopify Liquid Compatibility

This document describes the compliance status of this Go Liquid engine against the [Shopify Liquid spec](https://shopify.github.io/liquid/).

## Implemented Features

### Tags
- `assign`, `capture`, `echo`
- `if`, `elsif`, `else`, `endif`, `unless`, `endunless`
- `case`, `when`, `else`, `endcase`
- `for`, `endfor` — with `limit:`, `offset:`, `reversed`
- `tablerow`, `endtablerow` — with `cols:`
- `break`, `continue`
- `comment`, `endcomment`
- `raw`, `endraw`
- `liquid`
- `increment`, `decrement`
- `render`, `include`

### forloop Variables
All forloop drop variables are exposed per iteration:
`index`, `index0`, `rindex`, `rindex0`, `first`, `last`, `length`

### tablerow Variables
All tablerow drop variables are exposed per iteration:
`col`, `col0`, `row`, `first`, `last`, `length`, `index`, `index0`

### Operators
- `==`, `!=`, `<`, `>`, `<=`, `>=`
- `contains` — substring check for strings, element check for arrays
- `and`, `or`

### Special Symbols
- `blank` — matches empty string, empty array, nil
- `empty` — alias for `blank`

### Filters (Standard)
`abs`, `append`, `at_least`, `at_most`, `capitalize`, `ceil`, `compact`,
`concat`, `date`, `default`, `divided_by`, `downcase`, `escape`, `escape_once`,
`first`, `floor`, `join`, `last`, `lstrip`, `map`, `minus`, `modulo`,
`newline_to_br`, `plus`, `prepend`, `remove`, `remove_first`, `replace`,
`replace_first`, `reverse`, `round`, `rstrip`, `size`, `slice`, `sort`,
`sort_natural`, `split`, `strip`, `strip_html`, `strip_newlines`, `times`,
`truncate`, `truncatewords`, `uniq`, `upcase`, `url_decode`, `url_encode`,
`base64_encode`, `base64_decode`, `where`

## Intentional Divergences from Ruby Liquid

### Truthiness: `0` and `""` are falsy

**Ruby Liquid:** `0` and `""` are truthy (only `nil` and `false` are falsy).
**This engine:** `0` and `""` are falsy (consistent with most languages and common expectations).

**Justification:** Ruby Liquid's truthiness rules surprise nearly every developer coming from any other language. The "only nil and false are falsy" rule is a Ruby-ism. This engine targets Go developers and embeds into Go applications where `0` and `""` being falsy is the standard expectation.

**Affected test cases in `spec_test.go`:** marked with `Skip: "truthiness divergence: ..."`.

### Type coercion in comparisons

**Ruby Liquid:** Does not coerce types in `==` comparisons (`"1" == 1` → false).
**This engine:** Same behavior — no coercion. No divergence here.

### Date formatting

**Ruby Liquid:** Uses system timezone by default for `date` filter output.
**This engine:** Uses UTC. When a timezone-aware time is provided, it is rendered in that timezone.

### `nil` output

**Ruby Liquid:** `nil` renders as empty string.
**This engine:** Same behavior. No divergence.

## Known Limitations

- The `render` tag's strict variable isolation (variables from parent scope are not accessible) is implemented correctly.
- Cycle groups across multiple `for` loops: single-group cycling is supported; named group cycling may have edge cases.
- `form` tag (Shopify-specific): not implemented.
- Theme tags (`layout`, `section`, `schema`, etc.): not implemented (Shopify-specific).
