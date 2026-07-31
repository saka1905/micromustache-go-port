# Public API mapping

## Scope and evidence

This document fixes the Phase 3A Go API surface against
[`alexewerlof/micromustache`](https://github.com/alexewerlof/micromustache) commit
`da3420db27b7a2fdfbb768811a1280b34952dc95` (package version `8.0.3`). The
authoritative source snapshot is stored under `oracle/upstream/` and protected
by `oracle/upstream.sha256`.

**Upstream fact:** the runtime exports are `render`, `renderFn`,
`renderFnAsync`, `compile`, `get`, `getRef`, `tokenize`, and `Renderer`.
`Renderer` exposes `render`, `renderFn`, and `renderFnAsync` methods.

**Go design decision:** Phase 3E implements `Compile`, `NewRenderer`, and
`Renderer.Render` in addition to the earlier top-level functions, tokenization,
and lookup. Resolver renderer methods and asynchronous behavior still return a
zero value with an error matching `ErrNotImplemented` through `errors.Is`.

## Function and type mapping

| Fixed upstream TypeScript type | Go type |
| --- | --- |
| `Scope` | `Scope` (`map[string]Value`) |
| `GetOptions` | `GetOptions` |
| `RendererOptions` | `RendererOptions` |
| `ResolveFn` | `Resolver` |
| `ResolveFnAsync` | `AsyncResolver` |
| `Tokens` | `Tokens` |
| `TokenizeOptions` | `TokenizeOptions` |
| `CompileOptions` | `CompileOptions` |
| path reference `string[]` | `Ref` (`[]string`) |

| Fixed upstream TypeScript API | Go API | Phase 3E result |
| --- | --- | --- |
| `render(template, scope?, options?): string` | `Render(template string, scope Scope, options CompileOptions) (string, error)` | implemented |
| `renderFn(template, resolveFn, scope?, options?): string` | `RenderFunc(template string, resolve Resolver, scope Scope, options CompileOptions) (string, error)` | implemented |
| `renderFnAsync(template, resolveFn, scope?, options?): Promise<string>` | `RenderFuncAsync(ctx context.Context, template string, resolve AsyncResolver, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `compile(template, options?): Renderer` | `Compile(template string, options CompileOptions) (*Renderer, error)` | implemented |
| `get(scope, path, options?): any` | `Get(scope Scope, path string, options GetOptions) (Value, error)` | implemented |
| `getRef(scope, ref, options?): any` | `GetRef(scope Scope, ref Ref, options GetOptions) (Value, error)` | implemented |
| `tokenize(template, options?): Tokens` | `Tokenize(template string, options TokenizeOptions) (Tokens, error)` | implemented |
| `new Renderer(tokens, options?)` | `NewRenderer(tokens Tokens, options RendererOptions) (*Renderer, error)` | implemented |
| `Renderer.render(scope?): string` | `(*Renderer).Render(scope Scope) (string, error)` | implemented |
| `Renderer.renderFn(resolveFn, scope?): string` | `(*Renderer).RenderFunc(resolve Resolver, scope Scope) (string, error)` | `"", ErrNotImplemented` |
| `Renderer.renderFnAsync(resolveFn, scope?): Promise<string>` | `(*Renderer).RenderFuncAsync(ctx context.Context, resolve AsyncResolver, scope Scope) (string, error)` | `"", ErrNotImplemented` |

**Go design decision:** `renderFn` becomes the Go-idiomatic `RenderFunc`.
Asynchronous calls accept `context.Context` for cancellation and deadlines and
otherwise keep the same resolver role. Go callers pass explicit zero-value
scope and option arguments where TypeScript callers may omit them.

**Go design decision:** upstream `compile` returns `Renderer`, so Go `Compile`
returns `*Renderer`. A separate `CompiledTemplate` type would create an
upstream distinction that does not exist and is therefore not introduced.

## Values, scopes, and resolvers

`Value` is `any`, and a JavaScript object at the API boundary maps to
`Scope`, defined as `map[string]Value`. `Ref` is `[]string`.

The following states remain representable and distinct:

| JavaScript state | Go representation |
| --- | --- |
| missing property | key absent from `Scope` |
| `undefined` | present key whose value is `Undefined{}` |
| `null` | present key whose value is `nil` |
| `false` | `false` |
| numeric zero | `0` (or another zero-valued numeric type) |
| empty string | `""` |

`Get` and `GetRef` resolve these values without coercing or stringifying them.
Top-level `Render` applies the Phase 3C conversion boundary described below.

The synchronous resolver is
`func(path string, scope Scope) (Value, error)`. `RenderFunc` calls it in Phase
3D as described below. The asynchronous resolver is `func(ctx context.Context,
path string, scope Scope) (Value, error)` and remains uncalled; Promise
ordering, rejection behavior, and cancellation semantics remain implementation
work.

## Options and defaults

| Upstream option | Go field | Upstream default |
| --- | --- | --- |
| `validateRef` | `GetOptions.ValidateRef bool` | `false` |
| `maxRefDepth` | `GetOptions.MaxRefDepth int` | `10` |
| `explicit` | `RendererOptions.Explicit bool` | `false` |
| `validatePath` | `RendererOptions.ValidatePath bool` | `false` |
| `maxPathLen` | `TokenizeOptions.MaxPathLen int` | `1000` |
| `tags` | `TokenizeOptions.Tags Tags` | `["{{", "}}"]` |

`RendererOptions` embeds `GetOptions`. `CompileOptions` embeds both
`RendererOptions` and `TokenizeOptions`. Zero numeric fields and empty tag
fields mean unspecified at this API boundary.

**Go design decision:** `Tokenize` applies the upstream path-length and tag
defaults. `Get`, `GetRef`, and `Render` apply the upstream depth default.
`Render` applies `Explicit`; `ValidatePath` does not change top-level output
because the fixed top-level operation parses all paths before lookup whether
that flag is false or true.

`RenderFunc` applies tags and path-length options during tokenization,
`ValidatePath` before any resolver call, and `Explicit` during shared
stringification. `ValidateRef` and `MaxRefDepth` are lookup options and do not
apply because the resolver owns path interpretation.

`Compile` applies tags and `MaxPathLen` while producing tokens. Its renderer
snapshot retains only `RendererOptions`; tags and the template string are not
needed after tokenization. `Explicit`, `ValidateRef`, and `MaxRefDepth` apply
to every later `Renderer.Render`. `ValidatePath` selects eager constructor
parsing instead of the normal lazy first-render parsing.

## Tokens and source positions

**Upstream fact:** the public `Tokens` value exposes only `strings: string[]`
and `paths: string[]`. Go maps those fields to `Strings []string` and
`Paths []string`. The fixed upstream public type exposes no source offset,
index, raw token text, or parsed-reference field.

**Upstream fact:** internal JavaScript string indices and error positions use
JavaScript string operations and therefore count UTF-16 code units.

**Go design decision:** `Tokenize` counts `maxPathLen` and reported template
positions in UTF-16 code units. Node measurements confirmed that an emoji uses
two units. This matches JavaScript for valid UTF-8 Go strings without exposing
new token offset fields.

## Measured tokenizer behavior

**Upstream fact:** `Tokenize` finds delimiter pairs and returns one more
`Strings` element than `Paths` elements. Each path is trimmed using JavaScript
whitespace rules, but dot and bracket syntax is not parsed or validated at this
stage. A path such as `a.` is therefore accepted by `Tokenize` and rejected
later by `Get` or by a future validating `Renderer`.

Default tags are `{{` and `}}`; custom tags must be non-empty, distinct, and
must not contain one another. The default `maxPathLen` is 1000 UTF-16 code
units. Empty paths, nested opening tags, a missing close tag within the limit,
and invalid tag or length options return errors. A close tag without a prior
open tag is literal text.

The Go signature makes the input a string and makes tags an exactly-two-field
`Tags` value, so JavaScript wrong-input-type and wrong-tuple-length cases are
prevented by the type system. Go zero-valued tags and `MaxPathLen == 0` mean
"unspecified" and select defaults; the struct cannot distinguish that state
from an explicitly supplied JavaScript zero.

`validateRef`, `validatePath`, and `maxRefDepth` are not tokenization options.
The fixed JavaScript function ignores those extra object properties; Go's
typed `TokenizeOptions` does not expose them.

## Path grammar and reference segments

**Upstream fact:** `Get` calls the path parser and passes its result to
`GetRef`. `GetRef` itself does not parse path strings.

The private Go parser ports the fixed source grammar:

- Initial and dot segments use JavaScript's ASCII `[$_\w]+` form.
- Leading dot and surrounding JavaScript whitespace are accepted; trailing
  dots and empty dot segments are syntax errors.
- Single quote, double quote, and backtick bracket keys preserve their content,
  including dots, brackets, Unicode, and backslashes.
- Backslashes are literal and do not unescape quotes. The closing quote is the
  first matching quote that can be followed by optional whitespace and `]`.
- Numeric brackets accept an optional plus sign, whitespace, and leading
  zeroes. They normalize to a decimal segment with at most 16 significant
  digits. Negative and floating-point numeric brackets are syntax errors;
  quoted negative, floating-point, and leading-zero keys remain exact strings.
- An empty or whitespace-only path produces an empty `Ref` and therefore
  returns the input scope.

Parsing allocates a new segment list and adds no cache in Phase 3B.

## Top-level synchronous rendering

The fixed upstream top-level function performs `compile(template,
options).render(scope)`. Phase 3C reproduces the observable one-shot path
without implementing the public compiler or renderer object:

1. `Tokenize` produces literal strings and raw paths.
2. Every path is parsed to a fresh `Ref` before any scope lookup occurs.
3. Every parsed reference is resolved in template order with `GetRef`.
4. Literal strings and converted values are concatenated in template order.

The second step is observable. For `{{missing}}{{a.}}` with `ValidateRef`, the
fixed implementation reports the invalid `a.` path before attempting the
missing lookup; Go preserves that order. An error returns an empty result plus
the applicable error.

### Deterministic string conversion

| Go value | Rendered form |
| --- | --- |
| `nil` | empty by default; `null` with `Explicit` |
| `Undefined{}` or a missing property | empty by default; `undefined` with `Explicit` |
| string | unchanged |
| boolean | `true` or `false` |
| `float32` / `float64` | JavaScript-style decimal/exponent form; negative zero is `0`; NaN and infinities use JavaScript spellings |
| built-in signed/unsigned integer within ±(2^53−1) | base-10 integer |
| `[]any` | recursive comma join; nil/undefined array elements are empty |
| `Scope` / `map[string]any` | `[object Object]` |

The measured exponent thresholds are decimal form for absolute values from
`1e-6` through values below `1e21`, and exponent form outside that interval.
Object conversion never iterates map keys, so output is deterministic.
`Explicit` applies to a resolved top-level interpolation; it does not turn
nil/undefined array elements into words because JavaScript array join does not.

Go values outside this table return an error matching `ErrUnsupportedValue`
instead of silently producing a plausible but unsupported string. The explicit
boundary includes structs, pointers, functions, channels, complex values,
typed slices/maps other than the listed types, named scalar types, cyclic or
overly deep arrays, and integers outside JavaScript's safe range. This is a Go
API safety decision, not a claim that fixed JavaScript throws the same error
class for analogous JavaScript values.

## Top-level synchronous resolver rendering

The fixed upstream top-level `renderFn` compiles the template and then calls
`Renderer.renderFn`. Phase 3D reproduces its observable one-shot behavior
without invoking `Compile`; `Renderer.RenderFunc` remains unimplemented:

1. `Tokenize` produces literal strings and raw trimmed paths.
2. When `ValidatePath` is true, every path is parsed before the resolver is
   validated or called. When false, paths are not parsed.
3. A valid resolver is called synchronously once per raw path occurrence, from
   left to right, with `(path, scope)`.
4. All resolver results are collected before shared stringification and
   concatenation begin.

Repeated paths are called repeatedly and are not cached. Empty templates and
templates without interpolation call a valid resolver zero times. A resolver
error stops the loop immediately, so later paths are not called and no
stringification occurs. Conversely, if all calls succeed, a later unsupported
Go value is detected only after every resolver call finishes.

`RenderFunc` passes the same `Scope` value supplied by the caller and does not
interpret a raw path itself unless validation was requested. Measurements
confirmed that `a.`, which is invalid for the parser, is passed to the resolver
and can render successfully when `ValidatePath` is false.

Resolver-returned values go through exactly the same private stringification
helper as `Render`, including `Explicit`, array/object handling, number forms,
and `ErrUnsupportedValue`. The resolver owns missing-value semantics: returning
`Undefined{}` is the Go representation of upstream `undefined`.

A returned Go error is wrapped as resolver context containing the raw path and
zero-based occurrence index. Wrapping retains the original error for
`errors.Is` and `errors.As`, but that added text is not an upstream JavaScript
message. A nil Go `Resolver` returns `ErrInvalidResolver`; upstream instead
rejects a non-function with `TypeError`, so only the safety intent corresponds.
Resolver panics are not converted into success and are not recovered.

## Compiled templates and data rendering

Fixed `compile` calls `tokenize(template, options)` immediately and constructs
a `Renderer` from the resulting `Tokens`. It does not retain the template text.
The renderer's data path is:

1. Obtain all parsed refs from the template-scoped cache, parsing once if the
   cache is not initialized.
2. Call `GetRef` for each cached ref and the current scope in template order.
3. Collect every value, then use the shared stringification and concatenation
   helper.

`ValidatePath` initializes the ref cache during `Compile`/`NewRenderer`;
otherwise the first `Renderer.Render` initializes it. The cache contains only
parsed template refs and a possible parse error. Every Render call performs
fresh lookup and stringification, so changing, adding, or removing scope data
is visible immediately. Repeated and adjacent paths retain their occurrence
order.

The Go `Renderer` stores:

- defensive copies of `Tokens.Strings` and `Tokens.Paths`;
- a by-value `RendererOptions` snapshot;
- `sync.Once`, parsed refs, and the template parse error;
- an initialization marker for safe nil/zero receiver rejection.

It does not store the original template, `CompileOptions.Tags`, any scope,
lookup values, rendered strings, or a global cache. `sync.Once` makes the lazy
template initialization safe when multiple goroutines make read-only Render
calls concurrently.

**Known deliberate difference:** fixed JavaScript retains its token arrays and
options object by reference. Measurements showed token string mutation changed
later output and changing `options.explicit` changed an existing renderer. Go
copies/snapshots constructor inputs so caller mutation cannot alter an existing
renderer. Tags are consumed during tokenization in both implementations and do
not retokenize an existing renderer.

`NewRenderer` requires `len(Strings) == len(Paths)+1`. Other JavaScript
wrong-object/type cases are prevented by Go's `Tokens` type. Invalid shape is
`ErrInvalidTokens`. A nil or zero-value Go `Renderer` has no JavaScript
constructor equivalent and returns `ErrInvalidRenderer` rather than panicking.

Top-level `Render` now directly follows fixed source by calling `Compile` and
then `Renderer.Render`. Therefore ordinary results and error classification are
shared rather than independently duplicated.

### Compile-time and render-time errors

| Condition | Phase |
| --- | --- |
| invalid tags, unclosed tag, `MaxPathLen` | `Compile` tokenization |
| invalid path with `ValidatePath` | `Compile` / `NewRenderer` |
| invalid path without `ValidatePath` | first `Renderer.Render` |
| `MaxRefDepth`, `ValidateRef`, missing validated ref | every `Renderer.Render` |
| unsupported Go value | every `Renderer.Render`, after all lookups |
| invalid token shape | `NewRenderer` |
| nil or zero-value Renderer | `Renderer.Render` |

All refs are parsed before any lookup. All values are looked up before
stringification. A cached path error remains template-scoped, while data errors
are recomputed and can disappear when later scope data changes.

## Go map and slice traversal

`GetRef` traverses `Scope`, `map[string]any`, and `[]any`. Maps use exact own
keys. Slices accept canonical decimal indices (`0`, `1`, and so on) and the
`length` property. Numeric bracket parsing can normalize `[01]` to `1`, whereas
the quoted key `["01"]` remains `01` and does not address slice index 1.

| Lookup state | Go result without validation | With `ValidateRef` |
| --- | --- | --- |
| missing key/index | `Undefined{}` | `ErrReference` |
| present `Undefined{}` at terminal segment | `Undefined{}` | `Undefined{}` |
| present `nil` at terminal segment | `nil` | `nil` |
| traversal beyond missing, `Undefined{}`, `nil`, or a primitive | `Undefined{}` | `ErrReference` |
| `false`, numeric zero, empty string, empty slice, or empty map | original value | original value |

`MaxRefDepth == 0` selects the upstream default of 10 in Go; a negative value
returns `ErrInvalidOption`. JavaScript can distinguish an omitted value from an
explicit zero and rejects explicit zero, while the Go value struct cannot.
`maxPathLen` does not apply to `Get` or `GetRef`.

## Error policy

Go exposes stable sentinels usable with `errors.Is`: `ErrInvalidTemplate`,
`ErrInvalidPath`, `ErrInvalidOption`, `ErrReference`, `ErrUnsupportedValue`,
`ErrInvalidResolver`, `ErrInvalidTokens`, and `ErrInvalidRenderer`. The
associated source-derived error text retains the fixed wording for
corresponding measured inputs. Resolver, token-shape, and zero-renderer safety
boundaries are Go-specific mappings.
JavaScript's `SyntaxError`, `TypeError`, generic `Error`, `RangeError`, and
`ReferenceError` classes do not have direct Go equivalents; the mapping is by
sentinel and message.

## Node oracle observations and compatibility warnings

Phase 3B measured cases covered delimiter splitting, raw dot/bracket paths, custom
tags, whitespace, malformed templates, UTF-16 path limits and positions,
leading/trailing dots, empty segments, quoted dots and brackets, literal
backslashes, Unicode keys, numeric-looking keys, arrays, missing values,
validation, depth limits, and empty references. The fixed oracle returned
`undefined` for missing/null/undefined/primitive intermediate traversal and
preserved terminal `null`, `undefined`, `false`, zero, empty string, empty
array, and empty object.

Phase 3C additionally measured 53 top-level render requests. They covered
literal ordering, repeated and adjacent paths, custom tags, Unicode template
and value preservation, quoted Unicode keys, explicit missing/null/undefined,
booleans, negative zero, NaN, infinities, arrays, nested arrays, objects,
method-looking object keys, exponent thresholds, tokenization/path/reference
errors, and invalid-path-before-lookup ordering. The fixed path grammar
rejected an unquoted Japanese identifier; Unicode in values and quoted keys
remains supported. An own non-callable `toString` property made JavaScript
coercion throw `TypeError`; Go maps have no prototype method dispatch and still
produce `[object Object]`. This known deliberate object-model difference is not
reported as upstream-equivalent behavior. These targeted measurements do not
constitute the later general differential harness.

Phase 3D measured 37 declarative top-level `renderFn` requests plus 11 fixed
call-observation cases. They confirmed success/failure output, raw path and
scope arguments, left-to-right call order, one call per occurrence, repeated
calls, first-error stopping, tokenize-before-resolver order,
`ValidatePath`-before-resolver order, explicit values, arrays, objects, custom
tags, Unicode, and source error messages. The observer was a fixed temporary
script with no request-provided code and was deleted after measurement.

Phase 3E measured 48 declarative `compile.render` requests plus 16 fixed
compile-stage and renderer-reuse observations. They covered output and error
messages, compile-versus-render failure stages, first invalid path, validation
limits, repeated renders with same/different/mutated data, missing-to-present
transitions, nested map/array changes, invalid tokens, and upstream token/options
reference retention. Temporary measurement inputs were deleted and do not form
the later differential harness.

**Unresolved compatibility warning:** fixed JavaScript lookup uses the `in`
operator. It can traverse prototype properties, execute getters, expose array
constructors, and observe an inherited `__proto__`; direct Node measurements
confirmed those behaviors. Go deliberately performs own map/slice traversal,
does not use reflection, and never calls getters or methods. Structs, pointers,
prototype chains, sparse-array holes, custom array properties, and root arrays
outside the current `Scope` type are not modeled. Own map keys named
`__proto__`, `constructor`, and `prototype` remain accessible.

**Unresolved compatibility warning:** Go strings can contain invalid UTF-8,
while JavaScript strings are sequences of UTF-16 code units. Phase 3B matches
measured Unicode behavior for valid UTF-8 strings; invalid UTF-8 has no claimed
byte-for-byte JavaScript equivalent.

## Error and runtime boundary

`ErrNotImplemented` remains the stable sentinel for `RenderFuncAsync`,
`Renderer.RenderFunc`, and `Renderer.RenderFuncAsync`.
Each unimplemented function and method wraps it with the API name.

The Node oracle is a development and differential-validation reference only.
The Go package does not import, launch, proxy, or fall back to Node.js, and it
has no external dependency.

## Later implementation order

1. Asynchronous rendering.
2. Differential testing against the fixed Node oracle.
