# Runnable Go demo

## Purpose and prerequisites

The Phase 5A demo exercises the complete mapped public Go API through a short,
deterministic command. Normal package use and the demo require only Go; they do
not launch or require Node.js, npm, PowerShell, the network, or repository
evidence files.

Use the Go version declared by `go.mod` (`go1.26.5` for the recorded run). The
PowerShell walkthrough additionally requires Windows PowerShell 5.1, but the
Go command itself is an ordinary cross-platform Go program.

## Quick start

From the repository root, run:

```console
go run ./cmd/micromustache-demo
```

The final line is emitted only after every section succeeds:

```text
DEMO_STATUS: PASS
```

The command can also be built and run without repository-relative runtime
data:

```console
go build -o micromustache-demo ./cmd/micromustache-demo
./micromustache-demo
```

The tracked output from an actual successful run is
[`evidence/demo-output.txt`](../evidence/demo-output.txt).

## Sections and public API coverage

| Section | What it demonstrates | Exported APIs called |
| --- | --- | --- |
| Basic Render | Nested data, multiple interpolations, and Unicode output | `Render` |
| Tokenize | Stable literal count and raw path order | `Tokenize` |
| Get and GetRef | Nested bracket/index lookup and a caller-built segmented ref | `Get`, `GetRef` |
| Compile and Renderer Reuse | One compiled renderer used with two scopes and direct construction from public tokens | `Compile`, `NewRenderer`, `Renderer.Render` |
| Synchronous Resolver | Stable resolver path order through top-level and compiled routes | `RenderFunc`, `Renderer.RenderFunc` |
| Asynchronous Resolver | Immediate fixed resolvers, multiple paths, context use, and interpolation-order output | `RenderFuncAsync`, `Renderer.RenderFuncAsync` |

The async section reports call counts, not goroutine entry or completion order.
It uses no sleep or artificial delay. All error returns are checked. If a
section fails, the command writes `DEMO_ERROR` with the section name to stderr,
exits non-zero, and does not emit `DEMO_STATUS: PASS`.

## Reproducibility walkthrough

On Windows PowerShell 5.1, run:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/run-demo.ps1
```

The walkthrough safely builds the command in the system temporary directory,
runs it twice, requires empty stderr and zero exit status, validates section
uniqueness and order, and compares the two complete UTF-8 outputs byte for
byte. It then validates the preserved original-test manifest and reads the
tracked differential and benchmark JSON evidence. Temporary binaries and
outputs are removed in `finally`.

Evidence reading is deliberately outside the Go demo. This keeps the runnable
command independent of the current working directory and avoids duplicating
validation counts in source code. The walkthrough does not automatically run
the long differential or benchmark processes; it prints their exact commands.

Successful completion ends with:

```text
WALKTHROUGH_STATUS: PASS
```

## Full validation and benchmark evidence

Reproduce the complete differential comparison:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-differential.ps1
```

Reproduce the correctness-gated benchmark baseline:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-benchmark.ps1
```

Node.js is needed only for those validation and benchmark tools because they
compare the port with the fixed TypeScript implementation. It is not a product
runtime dependency. See [the API mapping](API_MAPPING.md) for documented
compatibility differences, [differential testing](DIFFERENTIAL_TESTING.md) for
the comparison boundary, and [benchmarking](BENCHMARKING.md) for measurement
limitations.

The benchmark values describe one recorded environment. They are not a
universal speed claim, production latency guarantee, or proof of complete
compatibility.

## Troubleshooting

- If `go` is not found, install or select the Go version required by `go.mod`;
  do not edit the module to bypass the requirement.
- Run the direct `go run` command from the repository root so the package path
  resolves. A built demo binary has no repository-root requirement.
- If the walkthrough reports different outputs, run the direct command twice
  and keep both raw outputs; do not normalize or discard the mismatch.
- If tracked evidence is missing or malformed, restore the repository rather
  than hard-coding replacement counts into the demo.
- Node preparation is unnecessary for the quick demo. It is required only
  when reproducing the full cross-runtime validation or benchmark.
