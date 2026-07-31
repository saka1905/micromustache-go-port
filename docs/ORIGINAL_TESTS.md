# Original upstream tests

## Purpose and scope

This snapshot preserves the original micromustache tests as immutable evidence for later adapter and differential-test work.

- Unit tests: 9 files from upstream `src/*.spec.ts`, stored under `tests/original/src/`
- Distribution tests: 7 files from upstream `dist-test/`, including `run.sh`, stored under `tests/original/dist-test/`
- Fixtures: none; the fixed commit contains no fixture directory and the selected tests do not reference fixture files
- Total: 16 files, 30,626 bytes

Source implementation modules and generated `dist/` artifacts referenced by the tests are outside this evidence snapshot and were not copied.

## Provenance

- Competition kickoff: 2026-08-01 03:00 JST
- Upstream: https://github.com/alexewerlof/micromustache
- Upstream commit: `da3420db27b7a2fdfbb768811a1280b34952dc95`
- Snapshot acquisition time: 2026-08-01 05:11:55 JST / 2026-07-31 20:11:55 UTC
- Manifest generation time: 2026-08-01 05:16:35 JST / 2026-07-31 20:16:35 UTC

This manifest was generated after kickoff from the fixed upstream commit. It was not generated at the kickoff time. Originality is demonstrated by both the immutable Git commit identifier and the per-file SHA-256 values.

## Verification and use

The manifest at [tests/original.sha256](../tests/original.sha256) records lowercase SHA-256 values and paths relative to `tests/original/`. Run the read-only verifier from the repository root:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-original-tests.ps1
```

The verifier fails on missing files, extra files, malformed or duplicate manifest entries, and hash mismatches. Files under `tests/original/` must not be edited. A later Goal will place adapters outside the original-test directory.

These tests have not been run against a Go implementation because no Go implementation exists yet. Preparation-stage investigation reported 264/264 upstream unit tests passing, but this submission repository has not installed dependencies or rerun the Node.js test suite.
