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
