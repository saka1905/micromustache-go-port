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

**Go design decision:** Phase 3A defines the complete corresponding public
surface but implements no micromustache behavior. Every operation returns a
zero value together with an error matching `ErrNotImplemented` through
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

| Fixed upstream TypeScript API | Go API | Phase 3A result |
| --- | --- | --- |
| `render(template, scope?, options?): string` | `Render(template string, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `renderFn(template, resolveFn, scope?, options?): string` | `RenderFunc(template string, resolve Resolver, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `renderFnAsync(template, resolveFn, scope?, options?): Promise<string>` | `RenderFuncAsync(ctx context.Context, template string, resolve AsyncResolver, scope Scope, options CompileOptions) (string, error)` | `"", ErrNotImplemented` |
| `compile(template, options?): Renderer` | `Compile(template string, options CompileOptions) (*Renderer, error)` | `nil, ErrNotImplemented` |
| `get(scope, path, options?): any` | `Get(scope Scope, path string, options GetOptions) (Value, error)` | `nil, ErrNotImplemented` |
| `getRef(scope, ref, options?): any` | `GetRef(scope Scope, ref Ref, options GetOptions) (Value, error)` | `nil, ErrNotImplemented` |
| `tokenize(template, options?): Tokens` | `Tokenize(template string, options TokenizeOptions) (Tokens, error)` | zero `Tokens`, `ErrNotImplemented` |
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

No Phase 3A operation resolves, coerces, compares, or stringifies these values.

The synchronous resolver is
`func(path string, scope Scope) (Value, error)`. The asynchronous resolver is
`func(ctx context.Context, path string, scope Scope) (Value, error)`. Phase 3A
does not call either resolver. Promise ordering, rejection behavior, and
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

**Go design decision:** the upstream defaults are documented now but are not
applied in Phase 3A. Applying defaults would be behavior, and behavior remains
deliberately unimplemented.

## Tokens and source positions

**Upstream fact:** the public `Tokens` value exposes only `strings: string[]`
and `paths: string[]`. Go maps those fields to `Strings []string` and
`Paths []string`. The fixed upstream public type exposes no source offset,
index, raw token text, or parsed-reference field.

**Upstream fact:** internal JavaScript string indices and error positions use
JavaScript string operations and therefore count UTF-16 code units.

**Unresolved compatibility warning:** if a later Go implementation exposes or
compares error positions, its byte/rune/UTF-16 conversion policy must be
specified and checked against the Node oracle. Phase 3A exposes no position and
does not choose that policy.

## Error and runtime boundary

`ErrNotImplemented` is the stable sentinel for the skeleton. Each function and
method wraps it with the API name, so callers must use
`errors.Is(err, ErrNotImplemented)` rather than compare error strings.

The Node oracle is a development and differential-validation reference only.
The Go package does not import, launch, proxy, or fall back to Node.js, and it
has no external dependency.

## Later implementation order

1. Tokenization and path resolution.
2. Synchronous rendering.
3. Compile and cache behavior.
4. Asynchronous rendering.
5. Differential testing against the fixed Node oracle.
