# Repository working rules

This repository contains the post-kickoff Go port of `alexewerlof/micromustache` for Port Mortem 2026.

## Before work

- Read `README.md` and `DECISIONS.md`, then inspect `git status` before changing files.
- Give each Goal one clear objective.
- Separate verified facts, assumptions, and decisions.

## Change and history policy

- Make small, verifiable commits after validation.
- Do not create a single bulk or dump commit containing unrelated phases.
- Do not add external dependencies without explicit approval.
- Prefer the Go standard library.
- Do not remove any upstream public API without an explicit recorded decision.
- Do not use `unsafe`-equivalent workarounds to conceal missing core port logic.
- Do not use Node.js as a proxy or wrapper for the product implementation.
- Node.js may be used only as a verification oracle.
- Update `docs/API_MAPPING.md` and `DECISIONS.md` when the public Go API changes.
- Unimplemented public operations must return `ErrNotImplemented`; do not return plausible success values.

## Upstream tests

- Do not modify upstream tests.
- Once imported, files under `tests/original/` are immutable reference copies.
- Preserve and verify the SHA-256 manifest generated from the fixed upstream commit after kickoff.

## Git and security

- Commit and push only after relevant validation passes or failures are documented.
- Do not amend or rewrite history without explicit approval.
- Never store secrets, credentials, access tokens, or real private data.

## Final report

Report changed files, validation commands and results, actions not performed, and remaining unverified items.
