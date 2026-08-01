# micromustache-go-port

This repository is the planned Port Mortem 2026 Track C submission for a TypeScript-to-Go port of `alexewerlof/micromustache`.

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

See [docs/UPSTREAM.md](docs/UPSTREAM.md) for the fixed metadata, [docs/ORIGINAL_TESTS.md](docs/ORIGINAL_TESTS.md) for the preserved-test record, [docs/NODE_ORACLE.md](docs/NODE_ORACLE.md) for the validation-only oracle, [docs/API_MAPPING.md](docs/API_MAPPING.md) for the TypeScript-to-Go public API mapping and known differences, [docs/DIFFERENTIAL_TESTING.md](docs/DIFFERENTIAL_TESTING.md) for the full comparison harness, and [DECISIONS.md](DECISIONS.md) for decisions.

## Current status

- Phase: **4A - full differential validation**
- Go public API: **Implemented for the complete fixed upstream surface**
- Implemented: **`Compile`, `NewRenderer`, `Renderer.Render`, `Renderer.RenderFunc`, `Renderer.RenderFuncAsync`, `RenderFuncAsync`, `RenderFunc`, `Render`, `Tokenize`, `GetRef`, and `Get`**
- Differential corpus: **218 cases across every mapped public operation**
- Differential result: **202 PASS, 13 EXPECTED_DIFFERENCE, 3 SKIP, 0 FAIL**
- Remaining work: **benchmark, demo, and final submission documentation**
- Original tests: **Stored unchanged and hash-verified**
- Node oracle: **Implemented and verified for validation only**
- Original tests against Go: **Not started**
- Differential testing: **PASS; two deterministic full runs produced the same normalized result hash**
- Benchmark: **Not started**

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

Reproduce the full comparison and deterministic rerun with:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1
```

This Git repository was created after the Port Mortem 2026 kickoff at 2026-08-01 03:00 JST (2026-07-31 18:00 UTC). It does not contain pre-kickoff implementation code or history copied from the preparation or upstream repositories.

## Planned order

1. Fix upstream metadata.
2. Store the original tests and their SHA-256 manifest.
3. Fix the Node.js oracle.
4. Add the Go API skeleton.
5. Port the path tokenizer and resolver.
6. Port synchronous rendering.
7. Port compile and cache behavior.
8. Port the asynchronous API.
9. Build differential testing.
10. Complete benchmarks, documentation, and the demo.

The Node oracle is a development and test reference only. The Go package never calls Node at runtime, uses it as a proxy or fallback, or requires it in the final build. Phase 4A compares all mapped operations through a shared declarative corpus. The passing harness is not a claim of complete compatibility: approved differences and explicitly unrepresentable cases remain documented in [docs/API_MAPPING.md](docs/API_MAPPING.md) and [docs/DIFFERENTIAL_TESTING.md](docs/DIFFERENTIAL_TESTING.md).
