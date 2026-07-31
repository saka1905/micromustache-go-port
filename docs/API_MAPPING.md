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

**Go design decision:** Phase 3B implements `Tokenize`, `GetRef`, and `Get`.
Rendering, compilation, resolver callbacks, and asynchronous behavior still
return a zero value with an error matching `ErrNotImplemented` through
`errors.Is`.

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

| Fixed upstream TypeScript API | Go API | Phase 3B result |
| --- | --- | --- |
| `render(template, scope?, options?): string` | `Render(template string, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `renderFn(template, resolveFn, scope?, options?): string` | `RenderFunc(template string, resolve Resolver, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `renderFnAsync(template, resolveFn, scope?, options?): Promise<string>` | `RenderFuncAsync(ctx context.Context, template string, resolve AsyncResolver, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `compile(template, options?): Renderer` | `Compile(template string, options CompileOptions) (*Renderer, error)` | `nil, ErrNotImplemented` |
| `get(scope, path, options?): any` | `Get(scope Scope, path string, options GetOptions) (Value, error)` | implemented |
| `getRef(scope, ref, options?): any` | `GetRef(scope Scope, ref Ref, options GetOptions) (Value, error)` | implemented |
| `tokenize(template, options?): Tokens` | `Tokenize(template string, options TokenizeOptions) (Tokens, error)` | implemented |
| `new Renderer(tokens, options?)` | `NewRenderer(tokens Tokens, options RendererOptions) (*Renderer, error)` | `nil, ErrNotImplemented` |
| `Renderer.render(scope?): string` | `(*Renderer).Render(scope Scope) (string, error)` | `"", ErrNotImplemented` |
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
Rendering-time conversion remains unimplemented.

The synchronous resolver is
`func(path string, scope Scope) (Value, error)`. The asynchronous resolver is
`func(ctx context.Context, path string, scope Scope) (Value, error)`. Phase 3B
still does not call either resolver. Promise ordering, rejection behavior, and
cancellation semantics remain implementation work.

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
defaults. `Get` and `GetRef` apply the upstream depth default. Options used only
by rendering remain unapplied because rendering is unimplemented.

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
`ErrInvalidPath`, `ErrInvalidOption`, and `ErrReference`. The associated error
text retains the fixed source wording for corresponding measured inputs.
JavaScript's `SyntaxError`, `TypeError`, generic `Error`, `RangeError`, and
`ReferenceError` classes do not have direct Go equivalents; the mapping is by
sentinel and message.

## Node oracle observations and compatibility warnings

Measured cases covered delimiter splitting, raw dot/bracket paths, custom
tags, whitespace, malformed templates, UTF-16 path limits and positions,
leading/trailing dots, empty segments, quoted dots and brackets, literal
backslashes, Unicode keys, numeric-looking keys, arrays, missing values,
validation, depth limits, and empty references. The fixed oracle returned
`undefined` for missing/null/undefined/primitive intermediate traversal and
preserved terminal `null`, `undefined`, `false`, zero, empty string, empty
array, and empty object.

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

`ErrNotImplemented` remains the stable sentinel for rendering, compilation,
constructor, and Renderer-method skeletons. Each unimplemented function and
method wraps it with the API name.

The Node oracle is a development and differential-validation reference only.
The Go package does not import, launch, proxy, or fall back to Node.js, and it
has no external dependency.

## Later implementation order

1. Synchronous rendering.
2. Compile and cache behavior.
3. Asynchronous rendering.
4. Differential testing against the fixed Node oracle.
