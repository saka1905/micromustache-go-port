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

## D-014 Compiled template representation

### Upstream facts

- Fixed `compile` tokenizes immediately and passes the resulting public `Tokens` plus renderer options to the `Renderer` constructor. The template string itself is not retained.
- `Renderer` retains tokens and options. Parsed refs are cached per template: eagerly in the constructor when `validatePath` is true, otherwise lazily on the first data render. Render data and output are never cached.
- Upstream retains JavaScript object/array references. Direct measurement confirmed that later options and token-array mutation can affect an existing renderer, while tag mutation after tokenization does not retokenize it.

### Go decisions

- `Compile` tokenizes once and delegates to `NewRenderer`. `Renderer` stores defensive copies of token strings and paths, a by-value `RendererOptions` snapshot, and a `sync.Once`-protected parsed-ref result. It stores no template string, scope, lookup values, or render output.
- Go deliberately prevents caller mutation of constructor tokens/options from changing an existing renderer. This differs from upstream reference retention and provides deterministic concurrent read-only use.
- Lazy parsing preserves upstream error timing when `ValidatePath` is false; eager parsing preserves constructor failure when it is true. The template-scoped cache is internal to one renderer, with no package-level or global cache.
- `NewRenderer` retains its existing signature. Invalid token shape returns `ErrInvalidTokens`; nil and zero-value renderers return `ErrInvalidRenderer` instead of panicking. Both are Go mappings/safety boundaries, not JavaScript error classes.
- `Renderer.Render` reuses parsed refs, calls `GetRef` for every occurrence on every invocation, and shares the same stringification helper as top-level `Render`. The top-level function now follows fixed source directly through `Compile(...).Render(...)`.
- The implementation uses only the Go standard library and does not call Node.js, the file system, or the network.

## D-015 Compile-time and render-time errors

- Tokenization errors, including invalid tags, unclosed tags, and `MaxPathLen`, occur during `Compile`.
- Invalid paths occur during `Compile` only with `ValidatePath`; otherwise they occur on the first `Renderer.Render` and are cached as the renderer's template error.
- `MaxRefDepth`, `ValidateRef`, unsupported Go values, and other data-dependent lookup/stringification errors occur during each `Renderer.Render`. They are not cached.
- All paths are parsed before any lookup, and all lookup values are collected before stringification. This preserves invalid-path-before-data-error and lookup-before-unsupported-value ordering.
- Source-derived errors retain existing sentinels and messages. `ErrInvalidTokens` and `ErrInvalidRenderer` remain distinguishable with `errors.Is`; they are not described as identical to upstream `TypeError`.
- Forty-eight declarative `compile.render` oracle requests and sixteen fixed stage/reuse observations confirmed representative output, first-error selection, compile/render stage boundaries, data non-caching, token/options reference behavior, and repeated rendering.

## D-016 Compiled synchronous resolver rendering

### Upstream facts

- `Renderer.renderFn` reads the raw trimmed occurrences in `tokens.paths`; it does not access or parse the renderer's refs cache. `validatePath` therefore affects it only through eager constructor validation.
- The resolver receives each raw path and the current scope from left to right, once per interpolation occurrence. Repeated paths are not deduplicated.
- A thrown resolver error stops later calls immediately. When all calls succeed, every value is collected before stringification, so a stringification error occurs after all resolver calls.
- The renderer retains no resolver or resolver result, and the same instance can be called again with different resolvers and scopes after success or failure.

### Go decisions

- `Renderer.RenderFunc` reuses the renderer's defensive-copy token paths, literal strings, and options snapshot. It does not retokenize, use parsed refs as resolver paths, or retain the resolver, scope, values, errors, or output.
- Top-level `RenderFunc` now follows fixed source through `Compile` and the same private compiled resolver helper. Existing raw-path, validation, call-order, stringification, and Go error-classification behavior is preserved.
- Nil resolvers use the existing `ErrInvalidResolver`. Resolver errors are wrapped with API, raw path, and occurrence index while preserving the cause for `errors.Is` and `errors.As`; this context is Go-specific.
- Resolver values use the existing deterministic stringification boundary. Unsupported Go values remain `ErrUnsupportedValue` and are not presented as the same error class as JavaScript stringification errors.
- No resolver result cache, global mutable state, goroutine, Node.js runtime call, file-system/network access, or external dependency is added. Concurrent calls are safe when the caller's resolver and scope permit concurrent read-only use.
- Forty declarative `compile.renderFn` oracle requests and fourteen fixed call/reuse observations confirmed representative output, raw path and scope, order/count, stopping, validation stage, stringification timing, and reuse after errors.

## D-017 Asynchronous resolver execution

### Upstream facts

- Top-level `renderFnAsync` compiles first and delegates to `Renderer.renderFnAsync`. The renderer synchronously invokes every async resolver from left to right through `resolveRefs`, then passes the returned promises to `Promise.all`.
- All resolver promises are created before the returned promise settles. Completion order may differ from invocation order, while `Promise.all` restores interpolation-index order before stringification.
- Repeated paths create independent promises and are not deduplicated. A rejection does not prevent later path resolvers from having started; the first rejection to settle is reported, including a later-index rejection that settles earlier.
- Stringification starts only after every promise fulfills. A stringification error therefore occurs after all resolver completions and follows interpolation order.

### Go decisions

- Top-level `RenderFuncAsync` compiles before validating the Go-only context and resolver boundaries. It dispatches one goroutine per raw path occurrence in left-to-right order, collects index-tagged results through a per-call channel buffered to the occurrence count, and uses the existing stringification helper.
- Goroutine completion order does not affect output order. The first observed resolver error is wrapped with raw path and index while preserving the cause for `errors.Is` and `errors.As`; simultaneous completion remains scheduler-dependent, analogous only in intent to Promise settlement ordering.
- `context.Context` is a Go-specific cancellation and deadline boundary. Nil context uses `ErrInvalidContext`; nil resolver continues to use `ErrInvalidResolver`. Cancellation and deadline causes remain available through `errors.Is`.
- Cancellation is checked before dispatch and while collecting. It may prevent later resolver dispatch, which has no JavaScript equivalent. Already-started user resolvers cannot be forcibly terminated; if they ignore the context they may continue after `RenderFuncAsync` returns.
- The result channel has enough capacity for every dispatched resolver, so internal result sends cannot remain blocked after an early error or cancellation. No global worker pool, semaphore, cache, mutable state, or package-level wait group is introduced.
- Resolver panics are not recovered or converted into successful values. Go scheduler execution of user resolver bodies may interleave after ordered dispatch; this is documented rather than claimed as identical JavaScript call-stack timing.
- Thirty-two declarative `renderFnAsync` oracle requests and seventeen fixed delay/event observations confirmed all-start behavior, invocation and completion ordering, repeated occurrences, fastest rejection selection, output order, validation timing, and stringification timing.

## D-018 Compiled asynchronous resolver rendering

### Upstream facts

- Fixed `Renderer.renderFnAsync` reads the raw trimmed occurrences in its retained `tokens.paths`, invokes `resolveRefs`, passes all returned promises to `Promise.all`, and stringifies the fulfilled values in interpolation order.
- It does not retokenize the template or read the parsed-ref cache during a call. `validatePath` can affect the method only through eager constructor validation; `validateRef` and `maxRefDepth` do not participate.
- A renderer retains template tokens and options but not the resolver, scope, promises, fulfillment values, rejection, or rendered output. The same instance remains reusable after resolver rejection or stringification failure.

### Go decisions

- `Renderer.RenderFuncAsync` reuses only the renderer's defensive-copy token strings, raw paths, options snapshot, initialization state, and any compile-time validation already completed. It does not call `Compile`, retokenize, or parse paths during the method call.
- The top-level and compiled async APIs share the Phase 3G private dispatch, indexed collection, error wrapping, context handling, and stringification helper. The API name in Go-specific error context remains appropriate to the caller.
- Context, resolver, scope, result channel, goroutines, resolved values, errors, and output are per-call values and are never retained by the renderer. Repeated use after resolver error, cancellation, deadline, and unsupported value therefore starts from fresh call state.
- Completion order does not change template-order output, and repeated paths remain independent calls. Existing warnings remain: actual resolver-body entry and simultaneous channel-error selection are scheduler-dependent, and a context-ignoring resolver already started cannot be forcibly terminated.
- The implementation adds no external dependency, Node.js runtime path, file-system/network access, global cache, semaphore, or worker pool. Sixty-four concurrent read-only calls are covered when caller resolvers and scopes are themselves concurrency-safe.
- Thirty-eight declarative compiled/constructor async oracle requests and sixteen fixed delay/event/reuse observations confirmed representative values, errors, validation stages, all-start behavior, reverse completion, output ordering, repeated occurrences, rejection selection, and reuse.

## D-019 Public API implementation complete

- Every fixed upstream runtime export and public `Renderer` method now has an implemented Go mapping. No mapped public operation returns `ErrNotImplemented` for normal input.
- `ErrNotImplemented` remains exported for source compatibility with earlier incremental phases, but no current public operation returns it. The now-unused private wrapper was removed.
- Public API implementation does not establish complete compatibility. Known differences remain documented in `docs/API_MAPPING.md`, and broad compatibility claims wait for the later differential harness.
- Work after Phase 3H focuses on validation, evaluation, demo, and submission evidence rather than adding unrequested public operations.

## D-020 Differential validation architecture

- The fixed Node oracle is the behavior reference. A validation-only Go runner calls only the actual exported Go APIs; it does not duplicate product logic or call Node.
- The same fixed, declarative NDJSON corpus is sent independently to both processes. Construction and sequence operations allow compile and renderer reuse to be compared without accepting executable request code.
- The comparator collects responses by id, validates protocol completeness, and compares normalized semantic results rather than raw process output. Stack traces, absolute paths, timestamps, and completion durations are excluded.
- An unapproved result or error difference is `FAIL`. An approved language/API-boundary difference must carry a case-level difference id and must actually differ; a stale difference annotation also fails.
- Node remains validation-only. The product package has no Node import, subprocess, proxy, fallback, or external Go dependency.

## D-021 Evidence reports

- Each successful full run writes a machine-readable JSON report and a human-readable Markdown summary under `evidence/`.
- Evidence records the corpus SHA-256, a timestamp-free deterministic result SHA-256, a summary SHA-256, fixed upstream and Go base commits, working-tree state, oracle source/version, execution environment, and reproduction command.
- The verifier executes the complete corpus twice and requires identical classifications, counts, normalized results, and deterministic hashes before copying the first run into tracked evidence.
- Any `FAIL`, process failure, timeout, malformed or missing response, unexpected/duplicate id, unsupported difference id, or deterministic mismatch exits non-zero. A report containing `FAIL` is not accepted as successful evidence.

## D-022 Cross-runtime benchmark architecture

- A tracked declarative workload suite is the only shared input. The Node runner directly calls the fixed CJS artifact, while the Go runner directly calls exported Go APIs; neither runtime calls the other.
- A separate validation process runs each workload once before any timed process. API identity, semantic result digest, and resolver call count must match or measurement stops.
- The suite contains successful compatibility cases only. Phase 4A expected differences, skips, errors, cancellation, deadline, and rejection paths are not converted into performance cases.
- Benchmark commands, runners, aggregation, and evidence remain outside the product package. No product cache, optimization, telemetry, dependency, or public API change is introduced.

## D-023 Timed regions and sampling

- Module/binary startup, JSON reading, workload validation, input conversion, resolver creation, renderer setup, correctness checks, calibration, and report generation are outside measured samples.
- Top-level render and resolver APIs include their normal compile/tokenize work. `Compile` and `NewRenderer` measure construction only. Renderer methods reuse one renderer created before timing; `GetRef` receives a pre-parsed ref.
- Each runtime calibrates independently by doubling iterations until one batch reaches 200 ms, subject to a finite iteration cap. Three calibrated batches warm the runtime before seven measured batches in each round.
- Round 1 runs Node then Go; round 2 runs Go then Node, with a new process for every runtime/round. Runtimes are never measured in parallel.
- Async workloads use immediately successful fixed resolvers and sequentially await calls. They measure runtime-specific immediate Promise/goroutine overhead, not I/O latency or parallel capacity.

## D-024 Benchmark evidence and claims

- JSON evidence retains every raw sample and records min, nearest-rank p25, median, nearest-rank p75, max, total iterations, ops/s, IQR/median, and `Go median / Node median` for each workload.
- Markdown uses median as the primary display and states the ratio direction. No speed threshold is a PASS condition, and slower observations are retained.
- Evidence records the parent commit and modified working tree honestly because it is generated before commit. It also records workload hash, fixed upstream, runtime/config/environment details, order, commands, correctness, warnings, and a content hash.
- Results describe one documented environment and run policy. They do not establish universal superiority, production latency, I/O performance, scalability, exact repeatability of timing values, or complete compatibility.

## D-025 Runnable demo and evidence walkthrough

- The Phase 5A demo is one Go command under `cmd/micromustache-demo`. It uses only the standard library and exported product APIs, contains no copy of product parsing or rendering logic, and has no Node.js, npm, shell, network, filesystem, time, random, environment, or machine-identity dependency.
- Six fixed sections exercise all eleven mapped operations: render, tokenization, parsed and string path lookup, compilation, direct renderer construction, renderer reuse, and top-level/compiled synchronous and asynchronous resolver paths. Async output records deterministic values and counts, never goroutine entry or completion order.
- Section functions validate their own expected values before emitting PASS. Any error is reported with its section name to stderr, exits non-zero, and prevents the final `DEMO_STATUS: PASS` line.
- Tracked differential and benchmark values are not duplicated in Go source. Evidence reading remains in `scripts/run-demo.ps1` so a built demo binary works outside the repository and normal package use stays independent of validation artifacts.
- The PowerShell 5.1 walkthrough builds only in a validated system temporary directory, requires two byte-for-byte identical UTF-8 runs, checks section order and evidence integrity, prints but does not automatically run the full differential and benchmark commands, and removes temporary artifacts in `finally`.
- `evidence/demo-output.txt` records an actual successful direct run. It contains no timestamp, hostname, username, absolute path, environment details, or performance claim.
