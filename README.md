# micromustache-go-port

This repository is the Port Mortem 2026 Track C submission package for a dependency-free TypeScript-to-Go port of `alexewerlof/micromustache`.

## Fixed target

- Upstream: https://github.com/alexewerlof/micromustache
- Upstream branch: `master`
- Fixed upstream commit: `da3420db27b7a2fdfbb768811a1280b34952dc95`
- Upstream package version: `8.0.3`
- Upstream license: MIT
- Source language: TypeScript
- Target language: Go
- Track: C - TypeScript → Go
- Eligibility: PASS with WARN

See [docs/SUBMISSION_DRAFT.md](docs/SUBMISSION_DRAFT.md) for submission-form copy, [docs/SUBMISSION_COMPLIANCE.md](docs/SUBMISSION_COMPLIANCE.md) for the final requirement review, [docs/API_MAPPING.md](docs/API_MAPPING.md) for the TypeScript-to-Go public API mapping and known differences, [docs/DIFFERENTIAL_TESTING.md](docs/DIFFERENTIAL_TESTING.md) for the full comparison harness, [docs/BENCHMARKING.md](docs/BENCHMARKING.md) for the measurement boundary, [docs/DEMO.md](docs/DEMO.md) for the runnable walkthrough, [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for attribution, and [DECISIONS.md](DECISIONS.md) for decisions.

## Current status

- Phase: **5B - final submission package and compliance review complete**
- Go public API: **Implemented for the complete fixed upstream surface**
- Implemented: **`Compile`, `NewRenderer`, `Renderer.Render`, `Renderer.RenderFunc`, `Renderer.RenderFuncAsync`, `RenderFuncAsync`, `RenderFunc`, `Render`, `Tokenize`, `GetRef`, and `Get`**
- Differential corpus: **218 cases across every mapped public operation**
- Differential result: **202 PASS, 13 EXPECTED_DIFFERENCE, 3 SKIP, 0 FAIL**
- Benchmark workloads: **26 across all 11 mapped public operations**
- Benchmark correctness: **PASS before timing; 728 raw measured samples across two runtime-order rounds**
- Demo: **Six deterministic sections exercise all 11 mapped public operations; two complete runs are byte-for-byte verified**
- Remaining work: **Phase 5C demo video, final rules/announcement recheck, and official form submission**
- Original tests: **Fixed upstream suite measured 264/264 PASS; 16 imported files remain unchanged and hash-verified**
- Node oracle: **Implemented and verified for validation only**
- Original tests against Go: **Not started**
- Differential testing: **PASS; two deterministic full runs produced the same normalized result hash**
- Benchmark: **Baseline recorded for one environment; no performance threshold or universal claim**

Integrity and license evidence:

- [Original-test SHA-256 manifest](tests/original.sha256)
- [Preserved upstream MIT License](third_party/micromustache/LICENSE)
- [Node oracle protocol](oracle/node/PROTOCOL.md)
- [Upstream source manifest](oracle/upstream.sha256)
- [Node oracle verifier](scripts/verify-node-oracle.ps1)
- [Public API mapping](docs/API_MAPPING.md)
- [Differential testing guide](docs/DIFFERENTIAL_TESTING.md)
- [Machine-readable differential evidence](evidence/differential-summary.json)
- [Human-readable differential evidence](evidence/differential-summary.md)
- [Benchmarking guide](docs/BENCHMARKING.md)
- [Machine-readable benchmark evidence](evidence/benchmark-summary.json)
- [Human-readable benchmark evidence](evidence/benchmark-summary.md)
- [Recorded demo output](evidence/demo-output.txt)

## Quick start and final check

Go 1.24 or later is required. The product and demo use only the Go standard library and require neither Node.js nor external Go modules.

```console
go build ./...
go test -count=1 ./...
go run ./cmd/micromustache-demo
```

Run the bounded final-submission check on Windows PowerShell 5.1 with:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-submission.ps1
```

It builds and tests the Go product, runs the demo from a temporary directory, checks protected manifests and tracked evidence, audits documentation and licenses, and ends with `SUBMISSION_VERIFICATION: PASS`. It deliberately does not install Node dependencies or rerun the long differential and benchmark jobs.

## Go-only product boundary

Product code uses only the Go standard library. It does not launch or embed Node.js, proxy requests to TypeScript, access the network, or use validation evidence at runtime. Node.js exists only in the separate oracle, differential, and benchmark tooling. This is a semantic Go port, not a drop-in npm replacement, and complete compatibility is not claimed.

The fixed TypeScript unit suite measured 264/264 PASS against the fixed TypeScript source. The 16 imported test and snapshot files are unchanged and hash-verified; this is provenance evidence, not a claim that those TypeScript tests execute directly against Go.

## Implemented public API

All 11 mapped operations are implemented: `Compile`, `NewRenderer`, `Renderer.Render`, `Renderer.RenderFunc`, `Renderer.RenderFuncAsync`, `RenderFuncAsync`, `RenderFunc`, `Render`, `Tokenize`, `GetRef`, and `Get`. Exact Go signatures, behavior notes, and approved language-boundary differences are in [docs/API_MAPPING.md](docs/API_MAPPING.md).

Run the Go-only demo:

```console
go run ./cmd/micromustache-demo
```

Its successful final line is `DEMO_STATUS: PASS`. Normal package use and the
demo require Go only; Node.js is used only by cross-runtime validation and
benchmark tooling. On Windows PowerShell 5.1, run the build, two-run
determinism check, and evidence walkthrough with:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/run-demo.ps1
```

Reproduce the full comparison and deterministic rerun with:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/prepare-node-oracle.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1
```

Reproduce the correctness-gated two-round benchmark baseline with:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/prepare-node-oracle.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-benchmark.ps1
```

This Git repository was created after the Port Mortem 2026 kickoff at 2026-08-01 03:00 JST (2026-07-31 18:00 UTC). It does not contain pre-kickoff implementation code or history copied from the preparation or upstream repositories.

## Known differences and limitations

JavaScript prototype-chain lookup, getters, coercion, Promise scheduling, and error classes do not have exact Go equivalents. The port uses explicit Go errors and `context.Context`; resolver-body entry order and simultaneous async error selection can be scheduler-dependent. These boundaries and the 13 approved differential differences are documented in [docs/API_MAPPING.md](docs/API_MAPPING.md) and [docs/DIFFERENTIAL_TESTING.md](docs/DIFFERENTIAL_TESTING.md).

The Node oracle and benchmark runner are development evidence tools only. Benchmark observations come from one documented environment and are neither a universal speed claim nor proof of complete compatibility.

## Repository layout

- Root `*.go`: public package and implementation
- `cmd/micromustache-demo`: deterministic Go-only demo
- `internal`: validation-only differential and benchmark support
- `oracle`: fixed Node reference and upstream source manifest
- `tests/original`: preserved upstream tests and snapshots
- `testdata`: declarative differential and benchmark inputs
- `scripts`: short and full verification entry points
- `evidence`: tracked machine- and human-readable results
- `docs`: provenance, API, methods, demo, and submission records

## Licensing and attribution

The Go port is available under the [MIT License](LICENSE). The fixed upstream is also MIT licensed; its revision, copyright, imported tests, and snapshot provenance are recorded in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md), with the upstream license preserved at [third_party/micromustache/LICENSE](third_party/micromustache/LICENSE).
