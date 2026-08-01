# Port Mortem 2026 Submission Draft

## Project

**Title:** micromustache-go-port

**Track:** C — TypeScript to Go

**Repository:** https://github.com/saka1905/micromustache-go-port

**One-line description:** A dependency-free Go port of the fixed `alexewerlof/micromustache` TypeScript template engine, with deterministic cross-runtime correctness evidence and a runnable Go-only demo.

## Problem

For an AI-assisted cross-language port, generating code is easier than demonstrating behavioral equivalence. `micromustache` is small but behaviorally detailed: paths have their own grammar, rendering observes precise validation and error order, compiled renderers retain selected state, and synchronous and asynchronous resolver APIs have distinct execution semantics. A useful port must preserve those observable behaviors without hiding the TypeScript runtime behind a wrapper.

## Solution

This project maps the complete fixed upstream runtime surface to idiomatic Go. The product uses only the Go standard library and never launches Node.js. A fixed Node oracle remains outside the product as a validation reference, and both implementations consume the same declarative case corpus. The repository also preserves upstream tests, source snapshots, licenses, decisions, evidence reports, a correctness-gated benchmark baseline, and a deterministic demo.

## Technical approach

- Fixed `alexewerlof/micromustache` version `8.0.3` at commit `da3420db27b7a2fdfbb768811a1280b34952dc95` before implementation.
- Implemented all 11 mapped operations: top-level render/compile/tokenize/get APIs plus compiled renderer methods, including synchronous and asynchronous resolvers.
- Kept product code dependency-free and separated all Node, corpus, report, and benchmark code into validation tooling.
- Compared independent Node and Go processes with stable normalized results and an explicit allowlist for documented language/API-boundary differences.
- Recorded raw benchmark samples only after per-workload correctness checks passed.

## Key results

- **Public API:** all 11 mapped operations implemented.
- **Upstream provenance:** fixed TypeScript unit suite measured 264/264 PASS; 16 imported test/snapshot files remain SHA-256 verified. These TypeScript tests were run against the fixed TypeScript source, not directly against Go.
- **Differential validation:** 218 cases, 202 PASS, 13 approved EXPECTED_DIFFERENCE, 3 SKIP, 0 FAIL. Deterministic normalized hash: `46b9c5498cde0faf7203ee6426679cbbc964e667c2fe36ba70837abf9eddef4b`.
- **Benchmark baseline:** 26/26 correctness checks PASS with 728 retained raw samples over two reversed runtime-order rounds.
- **Demo:** six deterministic sections cover all 11 mapped operations and finish with `DEMO_STATUS: PASS`.
- **Development history:** 15 incremental commits through this Phase 5B package, all within the event window.

## Trade-offs and known differences

This is a semantic Go port, not a drop-in npm replacement. It does not reproduce JavaScript prototype-chain lookup or getter execution. Go scheduler behavior can change resolver-body entry order, and simultaneous async errors do not have JavaScript Promise settlement semantics. Go-specific signatures use explicit errors and `context.Context` cancellation/deadlines, while unsupported JavaScript-specific values and coercions have no exact Go equivalent. These boundaries are documented as approved differences or unsupported cases. The benchmark is a reproducible observation from one machine, not a universal speed claim. Complete compatibility and production readiness are not claimed.

## How to run

With Go 1.24 or later:

```console
go build ./...
go test -count=1 ./...
go run ./cmd/micromustache-demo
```

On Windows PowerShell 5.1, the short final check is:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-submission.ps1
```

Full cross-runtime regeneration first runs `scripts/prepare-node-oracle.ps1`, then `scripts/verify-differential.ps1` or `scripts/verify-benchmark.ps1` as documented in the repository.

## Links

- [README and quick start](../README.md)
- [API mapping and known differences](API_MAPPING.md)
- [Differential evidence methodology](DIFFERENTIAL_TESTING.md)
- [Benchmark methodology](BENCHMARKING.md)
- [Runnable demo guide](DEMO.md)
- [Compliance review](SUBMISSION_COMPLIANCE.md)
- [Decision log](../DECISIONS.md)
- [Third-party notices](../THIRD_PARTY_NOTICES.md)
- Demo video: `[DEMO_VIDEO_URL_TO_BE_ADDED]`
