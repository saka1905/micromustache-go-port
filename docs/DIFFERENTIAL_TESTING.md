# Differential testing

## Architecture and boundary

Phase 4A compares the fixed `alexewerlof/micromustache` package at commit
`da3420db27b7a2fdfbb768811a1280b34952dc95` with the Go port through one
tracked declarative corpus. `scripts/verify-differential.ps1` builds two
validation-only executables in the system temporary directory, runs the Node
oracle and Go runner independently, and passes their responses to the
comparator.

The Node side is `oracle/node/oracle.mjs` backed by the generated fixed CJS
bundle. The Go side is `cmd/micromustache-go-oracle`; its dispatch layer calls
the exported Go APIs and contains no copy of parsing, lookup, rendering, or
stringification behavior. Neither process uses the network or shares mutable
state between requests. The product package never imports or launches Node.

## Protocol and corpus

Both runners accept UTF-8 NDJSON with one request and response per id. They use
the recursive value envelope defined in [the Node protocol](../oracle/node/PROTOCOL.md),
including undefined, null, booleans, finite and special numbers, negative zero,
strings, arrays, objects, and stable error fields. Resolvers are declarative
path-to-action tables; request-provided code is never evaluated.

The fixed corpus is `testdata/differential/cases.ndjson`. Its 218 cases cover
every mapped top-level API, compile, direct renderer construction, every
renderer method, and same-instance reuse. It includes templates, path grammar,
UTF-16 positions, values, sync/async resolvers, errors, cancellation, and reuse
after success or failure. `compile.sequence` and `renderer.sequence` express
ordered reuse steps without general scripting.

## Semantic normalization

The comparator first requires valid JSON, a unique non-empty id, one response
per case on each side, no unexpected ids, a valid success value or error
envelope, zero exit status, empty diagnostic stderr, and completion before the
timeout. Response collection is id-based, so process output order does not
affect comparison.

Successful values retain the complete recursive envelope. Token strings and
paths, resolver calls, and sequence results are compared. Errors are reduced
only to status, semantic category, stable name, and message. Source-derived
messages remain exact. A Go resolver wrapper is compared through its preserved
cause name and message, because the additional path/index prefix is a documented
Go diagnostic. Async `calls` arrays are sorted to exclude scheduler-dependent
resolver-body entry order; values, call multiplicity, paths, errors, and
template-order output remain checked. Stack traces, absolute paths, timestamps,
durations, and completion time are never compared.

## Classification

- `PASS`: normalized Node and Go results are identical.
- `EXPECTED_DIFFERENCE`: the results differ and the case carries an observed,
  allow-listed difference id.
- `SKIP`: the corpus records a concrete representation limitation. A missing
  implementation is never a skip.
- `FAIL`: every other mismatch or harness/protocol/process failure.

An approved id on an equal result is also a failure, preventing stale
exceptions. The approved ids observed in the Phase 4A report are:

| Difference id | Boundary | Cases |
| --- | --- | ---: |
| `DIFF-GO-CONTEXT` | Context cancellation and deadlines are Go-only API boundaries. | 3 |
| `DIFF-GO-UNSUPPORTED` | Fixed Symbols and unsupported Go values fail at different typed boundaries. | 2 |
| `DIFF-GO-ZERO-OPTION` | A Go zero field selects the documented default while JavaScript can distinguish and reject an explicit zero. | 4 |
| `DIFF-JS-OWN-TOSTRING` | JavaScript coercion observes an own `toString`; Go map coercion does not invoke methods. | 1 |
| `DIFF-JS-PROTOTYPE` | JavaScript lookup uses the prototype chain; Go traverses own map/slice values only. | 3 |

The three skips are limited to getter execution, invalid UTF-8, and sparse
arrays because the no-code, JSON value envelope cannot represent them. Their
reasons are stored per case and repeated in the report. The broader compatibility
boundaries remain in [the API mapping](API_MAPPING.md).

## Reports and deterministic rerun

`scripts/run-differential.ps1` writes one JSON report and one Markdown summary.
They record UTC/JST time, commits, dirty-source flag, corpus hash, oracle
script/source/package version, Go/Node/OS versions, command, totals, API counts,
difference counts, normalized case results, and hashes. Raw stdout/stderr and
temporary binaries are not tracked.

`scripts/verify-differential.ps1` performs two complete runs. It requires at
least 150 cases, all required API operations, zero failures, identical counts,
identical normalized results, and an identical timestamp-free deterministic
SHA-256. Only after both runs pass does it copy the first reports to
`evidence/differential-summary.json` and `evidence/differential-summary.md`.

Run it after the fixed Node oracle has been prepared:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1
```

## Harness self-tests and failure conditions

The `internal/differential` tests cover equal and unequal responses, missing,
duplicate, malformed, and unexpected ids, approved and unapproved differences,
stale approvals, process non-zero status, timeout, order-independent collection,
stable report ordering/hashing, recursive codec behavior, and deliberately
broken fixtures under `testdata/differential/fixtures/`.

Any failed child process, timeout, stderr diagnostic, malformed protocol,
missing/unexpected response, unapproved mismatch, stale approval, nonzero
`FAIL` count, missing API, or rerun mismatch causes verification to exit
non-zero.

## Limitations

The report proves agreement only for the fixed corpus and explicit
normalization rules. It is not proof of complete compatibility. JavaScript
getters/prototypes, sparse arrays, invalid UTF-8, arbitrary objects/functions,
exact goroutine entry order, simultaneous async error selection, and behavior
outside the mapped Go value model remain constrained or documented differences.
No performance, benchmark, or production-runtime claim is made by this harness.
