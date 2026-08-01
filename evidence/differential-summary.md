# Differential validation summary

- Generated: `2026-08-01T15:22:38+09:00` / `2026-08-01T06:22:38Z`
- Fixed upstream: `da3420db27b7a2fdfbb768811a1280b34952dc95`
- Node oracle: `oracle/node/oracle.mjs`, source `oracle/upstream/dist/micromustache.cjs`, package `8.0.3`
- Go base commit: `9211b5351ad0178ee3759b18741bdb95dfe500da` (working tree modified: `true`)
- Corpus: `testdata/differential/cases.ndjson` (`ba7535ebe0aed137baa6cc408d6b26855ac80dd7415264146385f87cfd476cb5`)
- Result: PASS `202`, EXPECTED_DIFFERENCE `13`, SKIP `3`, FAIL `0`, total `218`
- Deterministic result SHA-256: `46b9c5498cde0faf7203ee6426679cbbc964e667c2fe36ba70837abf9eddef4b`
- Summary SHA-256: `6659c249fff14a268dd80c545dd366ee66fbf4a93ded5ee32b85611beeb6a10c`
- Environment: `go1.26.5`, Node `v24.15.0`, `windows/amd64`
- Command: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1`

## API counts

| API | Total | PASS | EXPECTED_DIFFERENCE | SKIP | FAIL |
| --- | ---: | ---: | ---: | ---: | ---: |
| `compile` | 6 | 5 | 1 | 0 | 0 |
| `compile.render` | 10 | 10 | 0 | 0 | 0 |
| `compile.renderFn` | 6 | 6 | 0 | 0 | 0 |
| `compile.renderFnAsync` | 7 | 7 | 0 | 0 | 0 |
| `compile.sequence` | 6 | 6 | 0 | 0 | 0 |
| `get` | 29 | 27 | 2 | 0 | 0 |
| `getRef` | 19 | 18 | 1 | 0 | 0 |
| `render` | 37 | 34 | 3 | 0 | 0 |
| `renderFn` | 15 | 14 | 1 | 0 | 0 |
| `renderFnAsync` | 19 | 16 | 3 | 0 | 0 |
| `renderer.construct` | 5 | 5 | 0 | 0 | 0 |
| `renderer.render` | 10 | 10 | 0 | 0 | 0 |
| `renderer.renderFn` | 6 | 6 | 0 | 0 | 0 |
| `renderer.renderFnAsync` | 8 | 7 | 1 | 0 | 0 |
| `renderer.sequence` | 6 | 6 | 0 | 0 | 0 |
| `skip` | 3 | 0 | 0 | 3 | 0 |
| `tokenize` | 26 | 25 | 1 | 0 | 0 |

## Approved differences observed

- `DIFF-GO-CONTEXT`: 3 — context cancellation and deadlines are Go-only API boundaries
- `DIFF-GO-UNSUPPORTED`: 2 — unsupported Go values and JavaScript Symbol/stringification failures use different error boundaries
- `DIFF-GO-ZERO-OPTION`: 4 — zero selects Go defaults while fixed JavaScript validates an explicit numeric zero
- `DIFF-JS-OWN-TOSTRING`: 1 — fixed JavaScript observes an own toString property during coercion
- `DIFF-JS-PROTOTYPE`: 3 — fixed JavaScript lookup uses the prototype chain while Go uses own map/slice values

## Non-PASS cases

| ID | Classification | Difference | Reason |
| --- | --- | --- | --- |
| `compile-zero-max-path` | EXPECTED_DIFFERENCE | `DIFF-GO-ZERO-OPTION` | zero selects Go defaults while fixed JavaScript validates an explicit numeric zero |
| `get-prototype-tostring` | EXPECTED_DIFFERENCE | `DIFF-JS-PROTOTYPE` | fixed JavaScript lookup uses the prototype chain while Go uses own map/slice values |
| `get-zero-max-depth` | EXPECTED_DIFFERENCE | `DIFF-GO-ZERO-OPTION` | zero selects Go defaults while fixed JavaScript validates an explicit numeric zero |
| `getref-prototype-tostring` | EXPECTED_DIFFERENCE | `DIFF-JS-PROTOTYPE` | fixed JavaScript lookup uses the prototype chain while Go uses own map/slice values |
| `render-own-tostring` | EXPECTED_DIFFERENCE | `DIFF-JS-OWN-TOSTRING` | fixed JavaScript observes an own toString property during coercion |
| `render-prototype-tostring` | EXPECTED_DIFFERENCE | `DIFF-JS-PROTOTYPE` | fixed JavaScript lookup uses the prototype chain while Go uses own map/slice values |
| `render-zero-max-path` | EXPECTED_DIFFERENCE | `DIFF-GO-ZERO-OPTION` | zero selects Go defaults while fixed JavaScript validates an explicit numeric zero |
| `renderFn-unsupported` | EXPECTED_DIFFERENCE | `DIFF-GO-UNSUPPORTED` | unsupported Go values and JavaScript Symbol/stringification failures use different error boundaries |
| `renderFnAsync-context-canceled` | EXPECTED_DIFFERENCE | `DIFF-GO-CONTEXT` | context cancellation and deadlines are Go-only API boundaries |
| `renderFnAsync-deadline` | EXPECTED_DIFFERENCE | `DIFF-GO-CONTEXT` | context cancellation and deadlines are Go-only API boundaries |
| `renderFnAsync-unsupported` | EXPECTED_DIFFERENCE | `DIFF-GO-UNSUPPORTED` | unsupported Go values and JavaScript Symbol/stringification failures use different error boundaries |
| `renderer.renderFnAsync-context` | EXPECTED_DIFFERENCE | `DIFF-GO-CONTEXT` | context cancellation and deadlines are Go-only API boundaries |
| `skip-getter-execution` | SKIP | `` | declarative codec cannot represent a JavaScript getter without accepting executable input |
| `skip-invalid-utf8` | SKIP | `` | JSON and the Node protocol require valid Unicode text and cannot carry invalid UTF-8 bytes |
| `skip-sparse-array` | SKIP | `` | the shared recursive value envelope represents dense arrays only |
| `tokenize-zero-max-path` | EXPECTED_DIFFERENCE | `DIFF-GO-ZERO-OPTION` | zero selects Go defaults while fixed JavaScript validates an explicit numeric zero |
