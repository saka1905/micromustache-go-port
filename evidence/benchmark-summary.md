# Cross-runtime benchmark baseline

This report records a reproducible measurement baseline. It does not set a performance threshold or claim universal superiority.

- Generated: `2026-08-01T16:39:28+09:00` / `2026-08-01T07:39:28Z`
- Repository base commit: `9f843a228ca73b07581fc501da39a1ebc22d4ce4` (working tree modified: `true`)
- Fixed upstream: `da3420db27b7a2fdfbb768811a1280b34952dc95`, package `8.0.3`
- Workloads: `testdata/benchmark/workloads.json` (`8acaa9cb2d24a65abcf546b3c6e1c99403ea35b6542aa8159b0764d42a27b1cb`), total `26`
- Config: warmup `3`, samples `7` per round/runtime/workload, minimum `200 ms`, max iterations `16777216`, process timeout `300 s`
- Runtime order: `round 1: Node -> Go`; `round 2: Go -> Node`
- Environment: Go `go1.26.5`, Node `v24.15.0`, `windows/amd64`, CPU `Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz`, logical processors `12`, memory `34284421120 bytes`, power `電源設定の GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (バランス)`
- Correctness gate: `PASS`
- Content SHA-256: `af7b6ba69096e16f09f114632a0d1200bd00c1ccb44a9b622d85a9bf791d7acf`
- Reproduce: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-benchmark.ps1`

## Results

The ratio is **Go median / Node median**. Values above 1 mean the observed Go median took more time; values below 1 mean it took less time.

| Workload | API | Size | Bytes/chars/interpolations/paths/data nodes | Node median ns/op | Go median ns/op | Node ops/s | Go ops/s | Go/Node | Node IQR/median | Go IQR/median |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `compile-large` | `compile` | large | 895/895/64/64/19 | 5903.50 | 36505.64 | 169391 | 27393 | 6.184 | 0.046 | 0.171 |
| `compile-small` | `compile` | small | 15/15/1/1/19 | 210.91 | 440.56 | 4741376 | 2269828 | 2.089 | 0.035 | 0.016 |
| `get-medium-array-index` | `get` | medium | 0/0/0/1/19 | 851.61 | 558.10 | 1174244 | 1791781 | 0.655 | 0.019 | 0.294 |
| `get-medium-bracket` | `get` | medium | 0/0/0/1/19 | 1240.10 | 737.50 | 806387 | 1355928 | 0.595 | 0.031 | 0.467 |
| `get-medium-deep` | `get` | medium | 0/0/0/1/19 | 1187.17 | 541.24 | 842341 | 1847610 | 0.456 | 0.088 | 0.180 |
| `get-small-shallow` | `get` | small | 0/0/0/1/19 | 316.41 | 319.65 | 3160480 | 3128379 | 1.010 | 0.036 | 0.380 |
| `getref-medium-array` | `getRef` | medium | 0/0/0/1/19 | 66.11 | 269.39 | 15126410 | 3712099 | 4.075 | 0.069 | 0.297 |
| `getref-medium-deep` | `getRef` | medium | 0/0/0/1/19 | 80.74 | 263.01 | 12384934 | 3802077 | 3.257 | 0.056 | 0.332 |
| `render-large-repeated` | `render` | large | 895/895/64/64/19 | 13266.28 | 51643.95 | 75379 | 19363 | 3.893 | 0.053 | 0.291 |
| `render-medium-nested` | `render` | medium | 181/181/12/12/19 | 2899.44 | 7851.02 | 344894 | 127372 | 2.708 | 0.084 | 0.411 |
| `render-small-shallow` | `render` | small | 15/15/1/1/19 | 382.33 | 1015.72 | 2615559 | 984521 | 2.657 | 0.053 | 0.547 |
| `renderer-construct-medium` | `renderer.construct` | medium | 181/181/12/12/19 | 46.86 | 347.23 | 21340554 | 2879940 | 7.410 | 0.070 | 0.094 |
| `renderer-render-large` | `renderer.render` | large | 895/895/64/64/19 | 3530.02 | 3571.75 | 283285 | 279974 | 1.012 | 0.015 | 0.482 |
| `renderer-render-medium-data-rotation` | `renderer.render` | medium | 181/181/12/12/19 | 805.92 | 1308.82 | 1240819 | 764048 | 1.624 | 0.021 | 0.350 |
| `renderer-render-small` | `renderer.render` | small | 15/15/1/1/19 | 104.63 | 251.00 | 9557384 | 3984072 | 2.399 | 0.089 | 0.091 |
| `renderer-renderfn-medium` | `renderer.renderFn` | medium | 181/181/12/12/19 | 554.48 | 815.48 | 1803497 | 1226269 | 1.471 | 0.013 | 0.314 |
| `renderer-renderfnasync-medium` | `renderer.renderFnAsync` | medium | 97/97/6/6/19 | 924.31 | 6561.07 | 1081889 | 152414 | 7.098 | 0.006 | 0.020 |
| `renderer-renderfnasync-small` | `renderer.renderFnAsync` | small | 8/8/1/1/19 | 392.09 | 1397.65 | 2550422 | 715489 | 3.565 | 0.007 | 0.013 |
| `renderfn-medium-repeated` | `renderFn` | medium | 181/181/12/12/19 | 2337.76 | 4372.84 | 427759 | 228684 | 1.871 | 0.065 | 0.398 |
| `renderfn-small-single` | `renderFn` | small | 8/8/1/1/19 | 335.01 | 800.02 | 2984964 | 1249973 | 2.388 | 0.010 | 0.421 |
| `renderfnasync-medium-multiple` | `renderFnAsync` | medium | 97/97/6/6/19 | 1958.93 | 8237.88 | 510484 | 121390 | 4.205 | 0.040 | 0.020 |
| `renderfnasync-small-single` | `renderFnAsync` | small | 8/8/1/1/19 | 667.03 | 1876.58 | 1499180 | 532885 | 2.813 | 0.053 | 0.022 |
| `tokenize-large-repeated` | `tokenize` | large | 895/895/64/64/0 | 5903.93 | 34913.87 | 169379 | 28642 | 5.914 | 0.015 | 0.160 |
| `tokenize-medium-mixed` | `tokenize` | medium | 181/181/12/12/0 | 1117.96 | 2761.28 | 894485 | 362151 | 2.470 | 0.008 | 0.369 |
| `tokenize-small-few` | `tokenize` | small | 34/34/2/2/0 | 302.53 | 616.91 | 3305447 | 1620984 | 2.039 | 0.073 | 0.436 |
| `tokenize-small-plain` | `tokenize` | small | 11/11/0/0/0 | 114.69 | 291.14 | 8719472 | 3434756 | 2.539 | 0.017 | 0.249 |

## Variability and raw evidence

Each runtime/workload combines two rounds and `14` raw samples. Min, p25, median, p75, max, iterations, elapsed nanoseconds, and ns/op remain in the JSON report. Percentiles use nearest-rank; median averages the two center values for even counts.

## Limitations

- Performance values are observations from one environment, not universal guarantees.
- Immediate async workloads measure runtime-specific Promise and goroutine overhead, not I/O latency or parallel capacity.
- Process startup, module load, workload parsing, setup, calibration, and report generation are outside measured API samples.
- The shared workloads cover successful compatibility cases only; error, cancellation, and known language-boundary differences remain correctness concerns rather than normal-performance baselines.
