# Demo recording guide

This is a factual two-to-three-minute recording outline. It is not the final
submission text, and Phase 5A does not record or upload a video.

## Suggested screen sequence

1. Show the repository URL and the README fixed-target section.
2. Point to Track C, upstream package `8.0.3`, and fixed commit
   `da3420db27b7a2fdfbb768811a1280b34952dc95`.
3. Run `go run ./cmd/micromustache-demo`.
4. Keep all six section PASS results and `DEMO_STATUS: PASS` visible.
5. Show the differential evidence line with 218 cases and FAIL 0.
6. Show the benchmark evidence line with 26 workloads and 728 raw samples.
7. Show `git log --oneline --reverse` to make the post-kickoff incremental
   history visible.
8. Return to `https://github.com/saka1905/micromustache-go-port`.

The PowerShell walkthrough can replace steps 3 through 6 because it runs the
same demo twice and reads the tracked evidence:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/run-demo.ps1
```

## Suggested narration

- “This ports micromustache 8.0.3 from TypeScript to Go for Track C.”
- “The source reference is fixed at commit
  `da3420db27b7a2fdfbb768811a1280b34952dc95`.”
- “All mapped runtime exports and public Renderer methods have Go APIs.”
- “The fixed upstream unit suite passed 264 of 264 tests during target
  investigation, and the 16 imported test files remain hash-verified.”
- “The differential corpus has 218 cases: 202 matches, 13 documented
  differences, 3 representation skips, and 0 failures.”
- “The benchmark retains 728 raw samples across 26 correctness-gated
  workloads.”
- “Normal Go package use and this demo do not require Node.js.”
- “Known JavaScript and Go semantic differences are documented rather than
  hidden.”

## Accuracy boundaries

Do not describe the port as completely compatible, a drop-in replacement,
bug-free, production-ready, or faster in every environment. Do not claim that
all JavaScript object, prototype, getter, sparse-array, UTF-16, Promise, or
microtask semantics are reproduced.

If a benchmark table is shown, retain its ratio direction and limitations.
Some recorded `Get` workloads had lower Go medians, while many other recorded
operations had lower Node medians. The measurements are observations from one
environment, not a universal performance ranking.
