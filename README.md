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

See [docs/UPSTREAM.md](docs/UPSTREAM.md) for the fixed metadata and [DECISIONS.md](DECISIONS.md) for the initial decisions.

## Current status

- Phase: **2A - repository initialization**
- Implementation: **Not started**
- Upstream tests: **Not imported yet**
- Differential testing: **Not started**
- Benchmark: **Not started**

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

No micromustache behavior is implemented in this initialization phase.
