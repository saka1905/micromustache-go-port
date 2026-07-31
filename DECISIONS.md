# Decisions

## D-001 Target selection

- Selected `alexewerlof/micromustache`.
- Fixed upstream commit: `da3420db27b7a2fdfbb768811a1280b34952dc95`.
- Language pair: TypeScript → Go.
- Port Mortem 2026 track: C.
- All 264 upstream unit tests passed during target investigation.
- No clear direct Go port was found within the recorded investigation scope; this is not a claim that none exists anywhere.
- JavaScript-specific object, prototype, getter, stringification, Promise, RegExp, and UTF-16 behavior remains a compatibility warning.
- The complete upstream public API is in scope; it will not be reduced silently.

## D-002 Repository separation

- The preparation repository and submission repository are separate Git repositories.
- This submission repository was created after kickoff.
- No pre-kickoff implementation, upstream Git history, or preparation-repository history was copied into it.

## D-003 Incremental history

- The first commit contains repository foundations only.
- Original tests, the Node.js oracle, and Go implementation will be separate later commits.
- Large initial dump commits are intentionally avoided.

## D-004 Original-test preservation

- Original tests were obtained from fixed upstream commit `da3420db27b7a2fdfbb768811a1280b34952dc95`.
- They are stored unchanged under `tests/original/` with their upstream-relative directory structure.
- A per-file SHA-256 manifest is stored at `tests/original.sha256`.
- The acquisition and manifest times are recorded honestly as post-kickoff times; the manifest was not generated at kickoff.
- Original tests remain immutable, and future adapters will be placed outside `tests/original/`.
- The temporary clone used `core.autocrlf=false` to prevent newline conversion.

## D-005 Third-party license preservation

- The authoritative upstream MIT License is stored unchanged at `third_party/micromustache/LICENSE`.
- Upstream package metadata and the fixed commit identifier are stored beside it for reproducibility.
- The upstream license is kept separate from this port project's own `LICENSE`.

## D-006 Validation-only Node oracle

- The original TypeScript implementation from fixed commit `da3420db27b7a2fdfbb768811a1280b34952dc95` is preserved as a comparison oracle.
- Node.js is used only during development and testing, never as a Go runtime dependency.
- The oracle must not proxy, fall back for, or conceal an unimplemented Go feature.
- Input and output use deterministic NDJSON with one response per request id.
- Special JavaScript values use an unambiguous recursive type envelope.
- Error comparison includes stable `name` and `message` fields and excludes stack traces.

## D-007 Generated build artifacts

- `node_modules` and generated build artifacts are not committed.
- They are regenerated with `npm.cmd ci` from the fixed `package-lock.json` and the measured upstream build command.
- Only the raw upstream source and required build configuration are preserved in `oracle/upstream/`.
- `oracle/upstream.sha256` detects changes to every preserved snapshot file.

## D-008 Go API mapping

- The Phase 3A API covers every runtime export and public `Renderer` method at fixed upstream commit `da3420db27b7a2fdfbb768811a1280b34952dc95`.
- Upstream `renderFn` maps to Go `RenderFunc`; asynchronous variants accept `context.Context`.
- Upstream `compile` returns `*Renderer`; no separate `CompiledTemplate` abstraction is introduced.
- JavaScript objects map to `Scope`, values map to `Value`, segmented references map to `Ref`, and JavaScript `undefined` uses an explicit `Undefined` marker.
- Public tokens contain only literal strings and paths, matching the fixed upstream public type.
- The complete mapping and unresolved UTF-16 position warning are recorded in `docs/API_MAPPING.md`.

## D-009 Skeleton before behavior

- Phase 3A defines types and signatures only; it does not implement parsing, path resolution, rendering, compilation, caching, or asynchronous execution.
- During Phase 3A, every public operation returned a zero value and an error matching `ErrNotImplemented` through `errors.Is`; later phases replace that sentinel only as each operation becomes implemented.
- Skeleton methods do not invoke resolvers or produce plausible output that could be mistaken for a completed implementation.
- The Go package has no Node.js runtime path and no external dependency.
- Any necessary public API change must record its rationale in `DECISIONS.md` and update `docs/API_MAPPING.md`.

## D-010 Path representation

- The reference grammar is ported from `oracle/upstream/src/parse.ts` at fixed commit `da3420db27b7a2fdfbb768811a1280b34952dc95`, not from a general Mustache grammar.
- `Tokenize` preserves trimmed path text. `Get` privately converts dot and bracket notation to a `Ref` segment list, while `GetRef` accepts an existing segment list.
- Quoted keys retain dots, brackets, quotes, backslashes, Unicode, and numeric-looking text. Numeric bracket indices normalize optional plus signs and leading zeroes without losing quoted-key distinctions.
- Invalid paths return errors matching `ErrInvalidPath`; tokenization errors match `ErrInvalidTemplate`. Source-derived error messages are retained.
- Parsing allocates a new reference and never changes the input path, map, slice, or caller-supplied `Ref`. No parser cache is added in this phase.

## D-011 Go value traversal boundary

- `Scope`, `map[string]any`, and `[]any` are the supported traversal containers. Slice lookup supports canonical non-negative indices and `length`.
- A missing property returns `Undefined{}` unless `ValidateRef` requests an `ErrReference`. A present `Undefined{}` and a present `nil` remain distinguishable during traversal.
- JavaScript prototype-chain lookup is not reproduced. Own map keys named `__proto__`, `constructor`, or `prototype` remain ordinary keys.
- Structs, pointers, reflection, getters, and arbitrary methods are not traversed or invoked. Sparse-array holes and custom array properties are not modeled by `[]any`.
- Node.js is used only to measure the fixed implementation. The Go package uses the standard library and never calls the oracle at runtime.

## D-012 Synchronous rendering pipeline

- Phase 3C implements only the top-level `Render` function. `RenderFunc`, `RenderFuncAsync`, `Compile`, `NewRenderer`, all `Renderer` methods, caching, a general differential harness, and benchmarks remain unimplemented.
- The fixed upstream control flow is preserved: tokenize the template, parse every path, resolve every reference, then stringify and concatenate. Parsing every path before the first lookup preserves the observed invalid-path-before-missing-reference error order.
- `nil` and `Undefined{}` are swallowed by default and become `null` and `undefined` when `Explicit` is true. Within arrays they always contribute an empty element, matching JavaScript `Array.prototype.toString`/join behavior.
- Supported deterministic coercions are strings, booleans, floating-point values (including negative zero, NaN, and infinities), JavaScript-safe Go integers, `[]any`, `Scope`, and `map[string]any`. Arrays join recursively with commas; supported map objects stringify as `[object Object]` without depending on map iteration order.
- Values with no deliberately defined JavaScript coercion mapping return an error matching `ErrUnsupportedValue`. This includes structs, pointers, functions, channels, complex values, other slice/map types, named scalar types, cyclic arrays, excessive array nesting, and integers outside JavaScript's safe range.
- The Go renderer does not invoke Node.js, reflection-based methods, getters, callbacks, the file system, the network, or mutable global caches. Concurrent calls are safe when callers only read shared scope data.
- Fifty-three UTF-8 NDJSON cases were measured through the fixed Node oracle for output, error wording, error order, explicit values, arrays, objects, custom tags, Unicode values and quoted keys, method-looking object keys, and number-format thresholds. This targeted measurement is not the later full differential harness.

## D-013 Synchronous resolver rendering

### Upstream facts

- Top-level `renderFn` tokenizes first, then calls the supplied resolver synchronously once for every raw trimmed path occurrence from left to right. Repeated paths are not cached or deduplicated.
- Resolver calls receive the raw path and the same scope supplied to `renderFn`. Resolution stops at the first thrown error, and all successfully returned values are collected before stringification starts.
- Paths are parsed before resolver invocation only when `validatePath` is true. Without it, even a syntactically invalid path is passed to the resolver. `validateRef` and `maxRefDepth` do not participate in resolver rendering.

### Go decisions

- Phase 3D implements only top-level `RenderFunc`. It shares private path-validation and stringification helpers with `Render`; `RenderFuncAsync`, `Compile`, `NewRenderer`, all `Renderer` methods, and caches remain unimplemented.
- A resolver-returned error is wrapped with the raw path and occurrence index while preserving the original error for `errors.Is` and `errors.As`. The added context is Go-specific and is not presented as an upstream JavaScript message.
- A nil `Resolver` returns an error matching `ErrInvalidResolver` after tokenization and optional path validation. This is a Go-specific safety boundary corresponding only conceptually to upstream's non-function `TypeError`.
- Resolver results use the same deterministic stringification as `Render`. Unsupported Go values still return `ErrUnsupportedValue`; no alternative coercion is introduced.
- `RenderFunc` does not recover resolver panics, invoke Node.js, access the file system or network, start goroutines, or add global mutable state. It does not mutate the template, scope, or returned values itself.
- Thirty-seven declarative `renderFn` oracle requests and eleven fixed call-observation cases confirmed representative output, errors, raw paths, scope identity, order, count, repetition, stopping, and validation timing. The temporary observation inputs are not a full differential harness.
