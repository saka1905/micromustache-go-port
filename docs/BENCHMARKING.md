# Cross-runtime benchmarking

## Purpose and architecture

Phase 4B establishes a reproducible measurement baseline for the fixed Node.js
implementation and current Go port. It does not impose a performance target or
optimize the product.

The tracked suite in `testdata/benchmark/workloads.json` is read independently
by `oracle/node/benchmark.mjs` and `cmd/micromustache-go-benchmark`. The Node
runner loads `oracle/upstream/dist/micromustache.cjs`; the Go runner calls only
exported package APIs. Runtimes execute in separate processes and never invoke
one another. `cmd/micromustache-benchmark-report` validates the outputs and
generates the evidence reports.

## Workload schema and coverage

Every workload declares a unique id, API, category, small/medium/large size,
template or path/ref, JSON data, optional data variants, a declarative resolver
table, whether target setup is included, successful-result requirements, and
measured input facts. Input facts are checked before execution:

- UTF-8 template bytes and Unicode character count;
- interpolation and path counts from the public tokenizer;
- recursively counted data nodes or main elements.

The 26 workloads cover all mapped operations:

| API | Workloads |
| --- | ---: |
| `Tokenize` / `tokenize` | 4 |
| `Get` / `get` | 4 |
| `GetRef` / `getRef` | 2 |
| `Render` / `render` | 3 |
| `RenderFunc` / `renderFn` | 2 |
| `RenderFuncAsync` / `renderFnAsync` | 2 |
| `Compile` / `compile` | 2 |
| `NewRenderer` / `Renderer` constructor | 1 |
| `Renderer.Render` / `Renderer.render` | 3 |
| `Renderer.RenderFunc` / `Renderer.renderFn` | 1 |
| `Renderer.RenderFuncAsync` / `Renderer.renderFnAsync` | 2 |

Tokenize, end-to-end render, and compiled render each have small, medium, and
large inputs. Templates range from 8–34 bytes for small interpolating cases,
97–181 for medium cases, and 895 bytes with 64 occurrences for large cases.
Path-only cases record zero template bytes and one lookup path. The current
shared data has 19 recursively counted nodes; tokenization has no data.

## Correctness gate

Before any timed process, both runners execute every workload once in
`validate` mode. The reporter requires the same API identity, stable SHA-256 of
the normalized success result, and resolver call count. A missing, duplicate,
unexpected, malformed, failed, or mismatched result stops the run before
measurement.

Benchmark mode repeats the same validation before its own calibration. Known
Phase 4A differences and skips are not benchmark workloads. Error, rejection,
cancellation, and deadline behavior remains correctness scope rather than a
normal-performance baseline.

## Timed region and setup exclusion

Workload loading, JSON parsing, validation, module load, binary startup, data
conversion, options and fixed-resolver preparation, compiled renderer setup,
calibration, environment collection, aggregation, and report generation are
outside measured samples.

The target API boundary is preserved:

- top-level `Render`, `RenderFunc`, and async variants include their normal
  compile/tokenize path;
- `Compile` measures compilation without rendering;
- `NewRenderer` measures construction from tokens produced before timing;
- `Get` includes path parsing while `GetRef` receives a pre-parsed ref;
- Renderer method workloads reuse one renderer created before timing;
- the data-rotation workload selects already prepared scopes;
- resolver workloads use minimal prebuilt fixed-value resolvers without
  logging, sleep, network, or filesystem access.

Every result is assigned to a runner sink so the operation and result cannot be
discarded as unused. The loop and minimal sink assignment remain unavoidable
runner overhead and are not subtracted.

## Warmup, calibration, and samples

Each runtime calibrates each workload independently. Iterations start at one
and double until a batch reaches at least 200 ms. The cap is 16,777,216
iterations. Failure to reach the duration before the cap is an error.

After calibration, three complete batches are warmup only. Seven subsequent
batches are recorded. If a measured batch falls below 200 ms, iterations
double and that workload's recorded set restarts, preventing short samples from
entering evidence. Every runtime process has a 300-second finite timeout.

Round 1 executes Node then Go. Round 2 executes Go then Node. Every runtime and
round uses a new process, processes are sequential, and the 14 samples per
runtime/workload are combined. Performance values are expected to vary and are
not required to hash-identically across executions.

## Aggregation and interpretation

Raw JSON retains round, runtime, sample index, iterations, elapsed nanoseconds,
and ns/op. Per runtime/workload aggregation records two rounds, 14 samples,
total iterations, min, nearest-rank p25, median, nearest-rank p75, max,
`1e9 / median` ops/s, and `(p75-p25)/median` variability.

The observed ratio is `Go median / Node median`. Above 1 means the Go median
took more time in this run; below 1 means it took less. The ratio is descriptive
and never a PASS/FAIL threshold. Median is the primary summary; raw samples and
variability remain available rather than reducing evidence to an average.

## Environment evidence

The report records UTC/JST generation time, repository parent and dirty state,
fixed upstream commit/version, workload path/hash, configuration, Go and Node
versions, OS/architecture, CPU model, logical processor count, installed
memory, active Windows power plan when available, execution order, commands,
warnings, and a content hash. Unavailable machine facts are reported as
`unavailable`; hostname, username, absolute private paths, serial numbers, and
network identifiers are excluded.

## Async interpretation and limitations

Async resolvers return immediately and are awaited sequentially per benchmark
iteration. These workloads observe Promise and goroutine scheduling/allocation
overhead for an immediate success. They do not model I/O latency, throughput,
parallel capacity, cancellation, or failure selection.

The baseline is one observation on one environment. JIT, garbage collection,
OS scheduling, background activity, CPU temperature, and power policy can
change values. It does not prove universal speed, production latency,
scalability, complete compatibility, or performance after future changes.

## Reproduction

Prepare the fixed Node oracle first if its ignored dependencies/build are not
already present, then run:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-benchmark.ps1
```

The verifier rechecks original tests, the fixed snapshot, Node oracle,
differential regression, benchmark correctness, required API coverage, sample
schema, metrics, evidence privacy, and temporary-artifact cleanup before
writing `evidence/benchmark-summary.json` and
`evidence/benchmark-summary.md`.
