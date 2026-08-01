# Fixed Node reference oracle

## Purpose and boundary

The Node oracle executes the fixed TypeScript implementation to produce reference behavior during development and testing. It is validation-only.

It may be used to observe upstream values, JavaScript errors, and deterministic expectations for future differential testing. It must not be called by the Go package at runtime, used as a product proxy or fallback, included as a final-build dependency, or used to conceal an unimplemented Go feature.

The differential harness does not exist yet. Phase 3H completes the mapped Go public API through compiled asynchronous resolver rendering, while broad differential validation remains future work.

## Fixed upstream snapshot

- Upstream: https://github.com/alexewerlof/micromustache
- Branch: `master`
- Commit: `da3420db27b7a2fdfbb768811a1280b34952dc95`
- Version: `8.0.3`
- Snapshot acquisition: 2026-08-01 05:32:14 JST / 2026-07-31 20:32:14 UTC
- Snapshot: `oracle/upstream/`
- Manifest: [oracle/upstream.sha256](../oracle/upstream.sha256)
- Snapshot size: 12 files, 306,626 bytes

The snapshot contains the eight non-test TypeScript source files plus `package.json`, `package-lock.json`, `rollup.config.js`, and `tsconfig.json`. Every file matched the fixed clone byte-for-byte before the clone was removed. Original specs are not duplicated; [tests/original](../tests/original) remains their canonical location.

`node_modules`, `dist`, build output, coverage, editor settings, CI configuration, and upstream documentation are not committed. Generated dependencies and outputs are recreated from the fixed lockfile. They may remain locally as ignored validation artifacts, but they are never staged or used by the Go runtime.

## Protocol and codec

The oracle reads one request per UTF-8 NDJSON line and writes one response per line in the same order. `stdout` is reserved for responses and diagnostics use `stderr`. See [oracle/node/PROTOCOL.md](../oracle/node/PROTOCOL.md) for the exact request shapes.

The recursive codec covers `undefined`, `NaN`, positive and negative infinity, negative zero, `null`, booleans, finite numbers, strings, arrays, and plain objects. Every value uses an explicit type envelope, avoiding collisions between ordinary objects and special tags. Errors preserve stable `name` and `message` fields but omit stack traces.

Supported operations cover the complete fixed runtime export surface:

- `render`, `renderFn`, and `renderFnAsync`
- `compile.render`, `compile.renderFn`, and `compile.renderFnAsync`
- `get`, `getRef`, and `tokenize`
- `renderer.render`, `renderer.renderFn`, and `renderer.renderFnAsync`

Resolvers are declarative path-to-action maps. They can return an encoded value, return `undefined`, or raise a named error. Asynchronous operations convert the same actions to Promise fulfillment or rejection. Request-provided JavaScript is never evaluated.

The fixed CommonJS bundle is loaded fresh for each request. Resolver state and upstream internal caches therefore do not leak across NDJSON requests.

## Prepare, run, and verify

Measured environment:

- Node.js: `v24.15.0`
- npm: `11.12.1`
- Windows PowerShell: `5.1`

Prepare dependencies and build outputs:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/prepare-node-oracle.ps1
```

Preparation verifies the source manifest first, runs `npm.cmd ci --no-audit --no-fund` only under `oracle/upstream`, runs the measured upstream command `npm.cmd run build`, confirms CJS/MJS/type entry points, and verifies the lockfile and source manifest again.

Run an NDJSON file:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/run-node-oracle.ps1 -InputFile oracle/cases/smoke.ndjson
```

With no `-InputFile`, the script passes standard input to the oracle without parsing its stdout. It never prepares or downloads dependencies implicitly.

Run all oracle checks after preparation:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-node-oracle.ps1
```

## Measured results

- `npm.cmd ci`: PASS; 777 packages installed from the fixed lockfile for verification only
- `npm.cmd run build`: PASS; CJS, MJS, UMD, source maps, minified bundles, and type declarations generated
- Fixed upstream unit command `npm.cmd run test:unit`: PASS; 9/9 suites and 264/264 tests, 0 snapshots, 4.971 seconds
- Smoke: PASS; 27 requests, 25 success responses and 2 intentional error responses
- Error observations: resolver `RangeError` and invalid-path `SyntaxError`, both with deterministic name/message
- Determinism: PASS; two executions produced byte-identical NDJSON stdout
- Original-test manifest: PASS; 16/16 files
- Source manifest: PASS; 12/12 files before and after preparation
- Phase 3C targeted render measurement: PASS; 53 UTF-8 NDJSON requests covering output, conversion, Unicode keys, object-key coercion, and error order
- Phase 3D targeted renderFn measurement: PASS; 37 declarative UTF-8 NDJSON requests and 11 fixed call-observation cases
- Phase 3E targeted compile/render measurement: PASS; 48 declarative UTF-8 NDJSON requests and 16 fixed stage/reuse observations
- Phase 3F targeted compiled renderFn measurement: PASS; 40 declarative UTF-8 NDJSON requests and 14 fixed call/reuse observations
- Phase 3G targeted renderFnAsync measurement: PASS; 32 declarative UTF-8 NDJSON requests and 17 fixed delay/event observations
- Phase 3H targeted compiled renderFnAsync measurement: PASS; 38 declarative UTF-8 NDJSON requests and 16 fixed delay/event/reuse observations

The unit command was run in the temporary fixed-commit clone, not in `tests/original`, because the original specs depend on their upstream source-relative layout. This preserved the committed test originals while exercising the unmodified upstream command.

npm 11 emitted old-lockfile and deprecated-dependency warnings; no packages or lockfiles were updated. The known full-`npm test` CRLF/ESLint conflict and Node 24 extensionless-ESM distribution-test failure come from preparation-stage investigation and were not rerun or reported as current Phase 2C test results. Only the upstream unit command was run in this phase.

Smoke expected values are not hand-written into the case file. The fixed implementation generates responses at verification time; the verifier checks request/response ids, declared success/error status, codec validity, empty oracle stderr, and identical results across two runs.

The Phase 3C targeted request file was temporary and is not part of the committed oracle corpus. Its measured expectations were converted into Go table tests. This was a bounded implementation check, not the later reusable differential harness.

The Phase 3D request and call-observation files were also temporary and were deleted before commit. The fixed observer accepted no external code and recorded only path, scope identity, order, count, output, and error. Phase 3D did not extend the committed protocol or create the later differential harness.

The Phase 3E request and observation files were temporary and deleted before commit. The fixed observer separated compile and render failures and exercised one renderer repeatedly; it accepted no external code. The committed protocol and smoke corpus remain unchanged.

The Phase 3F request and observation files were temporary and deleted before commit. The fixed observer exercised raw-path delivery, scope identity, call order/count, resolver stopping, stringification timing, validation stages, and reuse of one renderer after resolver and value errors. It accepted no external code and did not extend the committed protocol.

The Phase 3G request and observer files were temporary and deleted before commit. The observer used fixed delays and recorded resolver invocation, fulfillment/rejection order, returned-promise timing, output, and errors. It confirmed all-start `Promise.all` behavior, interpolation-order output, earliest-settled rejection, repeated occurrences, and stringify-after-fulfillment without accepting external code or extending the protocol.

The Phase 3H request and observer files were temporary and deleted before commit. The declarative cases covered compiled and direct-constructor async calls; the fixed observer exercised completion order, rejection selection, validation stage, and reuse of one renderer after rejection and stringification failure. No request-provided code was accepted, and the committed protocol was not extended.
